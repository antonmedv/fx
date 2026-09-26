package engine

import "strings"

// TokenKind is the syntax class of a token of JS code.
type TokenKind int

const (
	TokenPlain  TokenKind = iota // identifiers
	TokenString                  // strings, template literal text and regexps
	TokenNumber
	TokenKeyword
	TokenBoolean  // true and false
	TokenNull     // null and undefined
	TokenProperty // name after a dot, like name in `.name` or `x.name`
	TokenPunct    // operators and brackets
	TokenComment
)

// Token is the byte range [Start, End) of code with a syntax class.
type Token struct {
	Start, End int
	Kind       TokenKind
}

// Tokenize splits JS code into tokens for syntax highlighting. Whitespace
// is not a token. It never fails: unterminated strings, templates and
// regexps extend to the end of code.
func Tokenize(code string) []Token {
	var tokens []Token
	emit := func(start, end int, kind TokenKind) {
		if start < end {
			tokens = append(tokens, Token{start, end, kind})
		}
	}
	i := 0
	for i < len(code) {
		c := code[i]
		start := i
		switch {
		case isSpace(c):
			i++

		case c == '"' || c == '\'':
			i = skipString(code, i)
			emit(start, i, TokenString)

		case c == '`':
			i = tokenizeTemplate(code, i, emit)

		case c == '/' && i+1 < len(code) && code[i+1] == '/':
			i = len(code)
			emit(start, i, TokenComment)

		case c == '/' && i+1 < len(code) && code[i+1] == '*':
			if end := strings.Index(code[i+2:], "*/"); end >= 0 {
				i += 2 + end + 2
			} else {
				i = len(code)
			}
			emit(start, i, TokenComment)

		case c == '/' && regexAllowed(code[:i]):
			i = skipRegex(code, i)
			for i < len(code) && isIdent(code[i]) {
				i++ // flags
			}
			emit(start, i, TokenString)

		case c >= '0' && c <= '9':
			i = skipNumber(code, i)
			emit(start, i, TokenNumber)

		case isIdentStart(c):
			for i < len(code) && (isIdent(code[i]) || code[i] >= 0x80) {
				i++
			}
			emit(start, i, wordKind(code, start, i))

		default:
			i++
			emit(start, i, TokenPunct)
		}
	}
	return tokens
}

// tokenizeTemplate emits the template literal starting at code[i], with the
// code of its ${} substitutions tokenized, and returns the index after it.
func tokenizeTemplate(code string, i int, emit func(start, end int, kind TokenKind)) int {
	start := i
	for i++; i < len(code); i++ {
		switch code[i] {
		case '\\':
			i++
		case '`':
			emit(start, i+1, TokenString)
			return i + 1
		case '$':
			if i+1 < len(code) && code[i+1] == '{' {
				emit(start, i, TokenString)
				emit(i, i+2, TokenPunct)
				end := findClose(code, i+2, '}')
				if end < 0 {
					end = len(code)
				}
				for _, t := range Tokenize(code[i+2 : end]) {
					emit(i+2+t.Start, i+2+t.End, t.Kind)
				}
				if end == len(code) {
					return end
				}
				emit(end, end+1, TokenPunct)
				i = end
				start = end + 1
			}
		}
	}
	if start < len(code) {
		emit(start, len(code), TokenString)
	}
	return len(code)
}

func skipNumber(code string, i int) int {
	hex := strings.HasPrefix(code[i:], "0x") || strings.HasPrefix(code[i:], "0X")
	for i < len(code) {
		c := code[i]
		switch {
		case isIdent(c) || c == '.':
			i++
		case (c == '+' || c == '-') && (code[i-1] == 'e' || code[i-1] == 'E') && !hex:
			i++
		default:
			return i
		}
	}
	return i
}

func wordKind(code string, start, end int) TokenKind {
	// A property after `.` or `?.`, but not after the spread `...`.
	if start > 0 && code[start-1] == '.' && (start < 2 || code[start-2] != '.') {
		return TokenProperty
	}
	switch code[start:end] {
	case "true", "false":
		return TokenBoolean
	case "null", "undefined":
		return TokenNull
	}
	if isKeyword(code[start:end]) {
		return TokenKeyword
	}
	switch code[start:end] {
	case "this", "try", "catch", "finally", "for", "while", "break", "continue",
		"switch", "default", "class", "extends", "super", "import", "export":
		return TokenKeyword
	}
	return TokenPlain
}

func isIdentStart(c byte) bool {
	return c == '_' || c == '$' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= 0x80
}
