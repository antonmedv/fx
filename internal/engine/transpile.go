package engine

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/charmbracelet/x/term"
	"github.com/mattn/go-runewidth"
)

func Transpile(args []string, i int) string {
	jsCode := transpile(args[i])
	snippet := formatErr(args, i, jsCode)
	return fmt.Sprintf(`  json = apply((function () {
    const x = this
    try {
      return %s
    } catch (e) {
      throw %s
    }
  }).call(json), json)

`, jsCode, "`"+snippet+"${e}`")
}

var (
	reBracket      = regexp.MustCompile(`^(\.\w*)+\[]`)
	reBracketStart = regexp.MustCompile(`^\.\[`)
	reDotStart     = regexp.MustCompile(`^\.`)
	reMap          = regexp.MustCompile(`^map\(.+?\)$`)
	reAt           = regexp.MustCompile(`^@`)
)

func transpile(code string) string {
	if code == "." {
		return "x"
	}

	if reBracket.MatchString(code) {
		return fmt.Sprintf("(%s)(x)", fold(strings.Split(code, "[]")))
	}

	if reBracketStart.MatchString(code) {
		return "x" + code[1:]
	}

	if reDotStart.MatchString(code) {
		return "x" + code
	}

	if reMap.MatchString(code) {
		s := code[4 : len(code)-1]
		if s[0] == '.' {
			s = "x" + s
		}
		return fmt.Sprintf(`x.map((x, i) => apply(%s, x, i))`, s)
	}

	if reAt.MatchString(code) {
		jsCode := transpile(code[1:])
		return fmt.Sprintf(`x.map((x, i) => apply(%s, x, i))`, jsCode)
	}

	return code
}

func fold(s []string) string {
	if len(s) == 1 {
		return "x => x" + s[0]
	}
	obj := s[0]
	s = s[1:]
	if obj == "." {
		obj = "x"
	} else {
		obj = "x" + obj
	}
	return fmt.Sprintf(`x => %s.flatMap(%s)`, obj, fold(s))
}

func formatErr(args []string, i int, jsCode string) string {
	width, _, err := term.GetSize(os.Stdout.Fd())
	if err != nil || width <= 0 {
		width = 80
	}

	if i < 0 || i >= len(args) {
		return fmt.Sprintf("Invalid argument index: %d", i)
	}

	code := args[i]

	const indentCols = 2
	const sepCols = 1
	reserve := indentCols + sepCols + runewidth.StringWidth(code) + sepCols

	available := width - reserve
	if available < 0 {
		available = 0
	}
	maxCtx := available / 2

	pre := strings.Join(args[:i], " ")
	post := strings.Join(args[i+1:], " ")

	pre = trimLeft(pre, maxCtx)
	post = trimRight(post, maxCtx)

	spacerCols := indentCols + runewidth.StringWidth(pre) + sepCols
	spacer := strings.Repeat(" ", spacerCols)

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat(" ", indentCols))
	if pre != "" {
		sb.WriteString(pre)
		sb.WriteByte(' ')
	}
	sb.WriteString(code)
	if post != "" {
		sb.WriteByte(' ')
		sb.WriteString(post)
	}
	sb.WriteByte('\n')

	sb.WriteString(spacer)
	sb.WriteString(strings.Repeat("^", runewidth.StringWidth(code)))
	sb.WriteByte('\n')

	if jsCode != "" && jsCode != code {
		snippet := jsCode
		if runewidth.StringWidth(snippet) > width {
			snippet = trimRight(snippet, width)
		}
		sb.WriteByte('\n')
		sb.WriteString(snippet)
		sb.WriteByte('\n')
	}

	sb.WriteString("\n")

	return sb.String()
}

// trimLeft returns at most ctx columns of s,
// taking its *last* ctx−1 columns and prefixing “…”
func trimLeft(s string, ctx int) string {
	if runewidth.StringWidth(s) <= ctx {
		return s
	}
	rs := []rune(s)
	widthAccum := 0
	var out []rune
	// walk backward until we fill ctx−1 columns
	for i := len(rs) - 1; i >= 0; i-- {
		w := runewidth.RuneWidth(rs[i])
		if widthAccum+w > ctx-1 {
			break
		}
		widthAccum += w
		out = append([]rune{rs[i]}, out...)
	}
	return "…" + string(out)
}

// trimRight returns at most ctx columns of s,
// taking its *first* ctx−1 columns and suffixing “…”
func trimRight(s string, ctx int) string {
	if runewidth.StringWidth(s) <= ctx {
		return s
	}
	rs := []rune(s)
	widthAccum := 0
	var out []rune
	// walk forward until we fill ctx−1 columns
	for _, r := range rs {
		w := runewidth.RuneWidth(r)
		if widthAccum+w > ctx-1 {
			break
		}
		out = append(out, r)
		widthAccum += w
	}
	return string(out) + "…"
}
