package fuzzy

import (
	"bytes"
	"strings"
	"unicode"
	"unicode/utf8"
)

var delimiterChars = "."

const whiteChars = " \t\n\v\f\r\x85\xA0"

type Result struct {
	Start int
	End   int
	Score int
}

const (
	scoreMatch        = 16
	scoreGapStart     = -3
	scoreGapExtension = -1

	// We prefer matches at the beginning of a word, but the bonus should not be
	// too great to prevent the longer acronym matches from always winning over
	// shorter fuzzy matches. The bonus point here was specifically chosen that
	// the bonus is cancelled when the gap between the acronyms grows over
	// 8 characters, which is approximately the average length of the words found
	// in web2 dictionary and my file system.
	bonusBoundary = scoreMatch / 2

	// Although bonus point for non-word characters is non-contextual, we need it
	// for computing bonus points for consecutive chunks starting with a non-word
	// character.
	bonusNonWord = scoreMatch / 2

	// Edge-triggered bonus for matches in camelCase words.
	// Compared to word-boundary case, they don't accompany single-character gaps
	// (e.g. FooBar vs. foo-bar), so we deduct bonus point accordingly.
	bonusCamel123 = bonusBoundary + scoreGapExtension

	// Minimum bonus point given to characters in consecutive chunks.
	// Note that bonus points for consecutive matches shouldn't have needed if we
	// used fixed match score as in the original algorithm.
	bonusConsecutive = -(scoreGapStart + scoreGapExtension)

	// The first character in the typed pattern usually has more significance
	// than the rest so it's important that it appears at special positions where
	// bonus points are given, e.g. "to-go" vs. "ongoing" on "og" or on "ogo".
	// The amount of the extra bonus should be limited so that the gap penalty is
	// still respected.
	bonusFirstCharMultiplier = 2
)

var (
	// Extra bonus for word boundary after whitespace character or beginning of the string
	bonusBoundaryWhite int16 = bonusBoundary

	// Extra bonus for word boundary after slash, colon, semi-colon, and comma
	bonusBoundaryDelimiter int16 = bonusBoundary + 1

	initialCharClass = charDelimiter

	// A minor optimization that can give 15%+ performance boost
	asciiCharClasses [unicode.MaxASCII + 1]charClass

	// A minor optimization that can give yet another 5% performance boost
	bonusMatrix [charNumber + 1][charNumber + 1]int16
)

type charClass int

const (
	charWhite charClass = iota
	charNonWord
	charDelimiter
	charLower
	charUpper
	charLetter
	charNumber
)

func init() {
	for i := 0; i <= unicode.MaxASCII; i++ {
		char := rune(i)
		c := charNonWord
		if char >= 'a' && char <= 'z' {
			c = charLower
		} else if char >= 'A' && char <= 'Z' {
			c = charUpper
		} else if char >= '0' && char <= '9' {
			c = charNumber
		} else if strings.ContainsRune(whiteChars, char) {
			c = charWhite
		} else if strings.ContainsRune(delimiterChars, char) {
			c = charDelimiter
		}
		asciiCharClasses[i] = c
	}
	for i := 0; i <= int(charNumber); i++ {
		for j := 0; j <= int(charNumber); j++ {
			bonusMatrix[i][j] = bonusFor(charClass(i), charClass(j))
		}
	}
}

func posArray(withPos bool, len int) *[]int {
	if withPos {
		pos := make([]int, 0, len)
		return &pos
	}
	return nil
}

func alloc16(offset int, slab *Slab, size int) (int, []int16) {
	if slab != nil && cap(slab.I16) > offset+size {
		slice := slab.I16[offset : offset+size]
		return offset + size, slice
	}
	return offset, make([]int16, size)
}

func alloc32(offset int, slab *Slab, size int) (int, []int32) {
	if slab != nil && cap(slab.I32) > offset+size {
		slice := slab.I32[offset : offset+size]
		return offset + size, slice
	}
	return offset, make([]int32, size)
}

func charClassOfNonAscii(char rune) charClass {
	if unicode.IsLower(char) {
		return charLower
	} else if unicode.IsUpper(char) {
		return charUpper
	} else if unicode.IsNumber(char) {
		return charNumber
	} else if unicode.IsLetter(char) {
		return charLetter
	} else if unicode.IsSpace(char) {
		return charWhite
	} else if strings.ContainsRune(delimiterChars, char) {
		return charDelimiter
	}
	return charNonWord
}

func charClassOf(char rune) charClass {
	if char <= unicode.MaxASCII {
		return asciiCharClasses[char]
	}
	return charClassOfNonAscii(char)
}

func bonusFor(prevClass charClass, class charClass) int16 {
	if class > charNonWord {
		switch prevClass {
		case charWhite:
			// Word boundary after whitespace
			return bonusBoundaryWhite
		case charDelimiter:
			// Word boundary after a delimiter character
			return bonusBoundaryDelimiter
		case charNonWord:
			// Word boundary
			return bonusBoundary
		}
	}

	if prevClass == charLower && class == charUpper ||
		prevClass != charNumber && class == charNumber {
		// camelCase letter123
		return bonusCamel123
	}

	switch class {
	case charNonWord, charDelimiter:
		return bonusNonWord
	case charWhite:
		return bonusBoundaryWhite
	}
	return 0
}

func normalizeRune(r rune) rune {
	if r < 0x00C0 || r > 0x2184 {
		return r
	}

	n := normalized[r]
	if n > 0 {
		return n
	}
	return r
}

func trySkip(input *Chars, caseSensitive bool, b byte, from int) int {
	byteArray := input.Bytes()[from:]
	idx := bytes.IndexByte(byteArray, b)
	if idx == 0 {
		// Can't skip any further
		return from
	}
	// We may need to search for the uppercase letter again. We don't have to
	// consider normalization as we can be sure that this is an ASCII string.
	if !caseSensitive && b >= 'a' && b <= 'z' {
		if idx > 0 {
			byteArray = byteArray[:idx]
		}
		uidx := bytes.IndexByte(byteArray, b-32)
		if uidx >= 0 {
			idx = uidx
		}
	}
	if idx < 0 {
		return -1
	}
	return from + idx
}

func isAscii(runes []rune) bool {
	for _, r := range runes {
		if r >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

func asciiFuzzyIndex(input *Chars, pattern []rune, caseSensitive bool) (int, int) {
	// Can't determine
	if !input.IsBytes() {
		return 0, input.Length()
	}

	// Not possible
	if !isAscii(pattern) {
		return -1, -1
	}

	firstIdx, idx, lastIdx := 0, 0, 0
	var b byte
	for pidx := 0; pidx < len(pattern); pidx++ {
		b = byte(pattern[pidx])
		idx = trySkip(input, caseSensitive, b, idx)
		if idx < 0 {
			return -1, -1
		}
		if pidx == 0 && idx > 0 {
			// Step back to find the right bonus point
			firstIdx = idx - 1
		}
		lastIdx = idx
		idx++
	}

	// Find the last appearance of the last character of the pattern to limit the search scope
	bu := b
	if !caseSensitive && b >= 'a' && b <= 'z' {
		bu = b - 32
	}
	scope := input.Bytes()[lastIdx:]
	for offset := len(scope) - 1; offset > 0; offset-- {
		if scope[offset] == b || scope[offset] == bu {
			return firstIdx, lastIdx + offset + 1
		}
	}
	return firstIdx, lastIdx + 1
}

type Slab struct {
	I16 []int16
	I32 []int32
}

const (
	caseSensitive = false
	normalize     = true
	forward       = true
)

func fuzzyMatch(input *Chars, pattern []rune) (Result, *[]int) {
	var slab *Slab
	// Assume that pattern is given in lowercase if case-insensitive.
	// First check if there's a match and calculate bonus for each position.
	// If the input string is too long, consider finding the matching chars in
	// this phase as well (non-optimal alignment).
	M := len(pattern)
	if M == 0 {
		return Result{0, 0, 0}, posArray(true, M)
	}
	N := input.Length()
	if M > N {
		return Result{-1, -1, 0}, nil
	}

	// Phase 1. Optimized search for ASCII string
	minIdx, maxIdx := asciiFuzzyIndex(input, pattern, caseSensitive)
	if minIdx < 0 {
		return Result{-1, -1, 0}, nil
	}
	// fmt.Println(N, maxIdx, idx, maxIdx-idx, input.ToString())
	N = maxIdx - minIdx

	// Reuse pre-allocated integer slice to avoid unnecessary sweeping of garbages
	offset16 := 0
	offset32 := 0
	offset16, H0 := alloc16(offset16, slab, N)
	offset16, C0 := alloc16(offset16, slab, N)
	// Bonus point for each position
	offset16, B := alloc16(offset16, slab, N)
	// The first occurrence of each character in the pattern
	offset32, F := alloc32(offset32, slab, M)
	// Rune array
	_, T := alloc32(offset32, slab, N)
	input.CopyRunes(T, minIdx)

	// Phase 2. Calculate bonus for each point
	maxScore, maxScorePos := int16(0), 0
	pidx, lastIdx := 0, 0
	pchar0, pchar, prevH0, prevClass, inGap := pattern[0], pattern[0], int16(0), initialCharClass, false
	for off, char := range T {
		var class charClass
		if char <= unicode.MaxASCII {
			class = asciiCharClasses[char]
			if !caseSensitive && class == charUpper {
				char += 32
				T[off] = char
			}
		} else {
			class = charClassOfNonAscii(char)
			if !caseSensitive && class == charUpper {
				char = unicode.To(unicode.LowerCase, char)
			}
			if normalize {
				char = normalizeRune(char)
			}
			T[off] = char
		}

		bonus := bonusMatrix[prevClass][class]
		B[off] = bonus
		prevClass = class

		if char == pchar {
			if pidx < M {
				F[pidx] = int32(off)
				pidx++
				pchar = pattern[min(pidx, M-1)]
			}
			lastIdx = off
		}

		if char == pchar0 {
			score := scoreMatch + bonus*bonusFirstCharMultiplier
			H0[off] = score
			C0[off] = 1
			if M == 1 && (forward && score > maxScore || !forward && score >= maxScore) {
				maxScore, maxScorePos = score, off
				if forward && bonus >= bonusBoundary {
					break
				}
			}
			inGap = false
		} else {
			if inGap {
				H0[off] = max(prevH0+scoreGapExtension, 0)
			} else {
				H0[off] = max(prevH0+scoreGapStart, 0)
			}
			C0[off] = 0
			inGap = true
		}
		prevH0 = H0[off]
	}
	if pidx != M {
		return Result{-1, -1, 0}, nil
	}
	if M == 1 {
		result := Result{minIdx + maxScorePos, minIdx + maxScorePos + 1, int(maxScore)}
		pos := []int{minIdx + maxScorePos}
		return result, &pos
	}

	// Phase 3. Fill in score matrix (H)
	// Unlike the original algorithm, we do not allow omission.
	f0 := int(F[0])
	width := lastIdx - f0 + 1
	offset16, H := alloc16(offset16, slab, width*M)
	copy(H, H0[f0:lastIdx+1])

	// Possible length of consecutive chunk at each position.
	_, C := alloc16(offset16, slab, width*M)
	copy(C, C0[f0:lastIdx+1])

	Fsub := F[1:]
	Psub := pattern[1:][:len(Fsub)]
	for off, f := range Fsub {
		f := int(f)
		pchar := Psub[off]
		pidx := off + 1
		row := pidx * width
		inGap := false
		Tsub := T[f : lastIdx+1]
		Bsub := B[f:][:len(Tsub)]
		Csub := C[row+f-f0:][:len(Tsub)]
		Cdiag := C[row+f-f0-1-width:][:len(Tsub)]
		Hsub := H[row+f-f0:][:len(Tsub)]
		Hdiag := H[row+f-f0-1-width:][:len(Tsub)]
		Hleft := H[row+f-f0-1:][:len(Tsub)]
		Hleft[0] = 0
		for off, char := range Tsub {
			col := off + f
			var s1, s2, consecutive int16

			if inGap {
				s2 = Hleft[off] + scoreGapExtension
			} else {
				s2 = Hleft[off] + scoreGapStart
			}

			if pchar == char {
				s1 = Hdiag[off] + scoreMatch
				b := Bsub[off]
				consecutive = Cdiag[off] + 1
				if consecutive > 1 {
					fb := B[col-int(consecutive)+1]
					// Break consecutive chunk
					if b >= bonusBoundary && b > fb {
						consecutive = 1
					} else {
						b = max(b, max(bonusConsecutive, fb))
					}
				}
				if s1+b < s2 {
					s1 += Bsub[off]
					consecutive = 0
				} else {
					s1 += b
				}
			}
			Csub[off] = consecutive

			inGap = s1 < s2
			score := max(max(s1, s2), 0)
			if pidx == M-1 && (forward && score > maxScore || !forward && score >= maxScore) {
				maxScore, maxScorePos = score, col
			}
			Hsub[off] = score
		}
	}

	// Phase 4. (Optional) Backtrace to find character positions
	pos := posArray(true, M)
	j := f0
	i := M - 1
	j = maxScorePos
	preferMatch := true
	for {
		I := i * width
		j0 := j - f0
		s := H[I+j0]

		var s1, s2 int16
		if i > 0 && j >= int(F[i]) {
			s1 = H[I-width+j0-1]
		}
		if j > int(F[i]) {
			s2 = H[I+j0-1]
		}

		if s > s1 && (s > s2 || s == s2 && preferMatch) {
			*pos = append(*pos, j+minIdx)
			if i == 0 {
				break
			}
			i--
		}
		preferMatch = C[I+j0] > 1 || I+width+j0+1 < len(C) && C[I+width+j0+1] > 0
		j--
	}
	// Start offset we return here is only relevant when begin tiebreak is used.
	// However finding the accurate offset requires backtracking, and we don't
	// want to pay extra cost for the option that has lost its importance.
	return Result{minIdx + j, minIdx + maxScorePos + 1, int(maxScore)}, pos
}
