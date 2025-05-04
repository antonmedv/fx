package engine

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/charmbracelet/x/term"
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

`, jsCode, "`"+snippet+"`")
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
	width, _, wErr := term.GetSize(os.Stdout.Fd())
	if wErr != nil || width <= 0 {
		width = 80
	}

	if i < 0 || i >= len(args) {
		return fmt.Sprintf("Invalid argument index: %d", i)
	}

	code := args[i]

	errPlaceholder := "${e}"

	reserve := 2 + 1 + len(code) + 1 + 2
	maxCtx := (width - reserve) / 2
	if maxCtx < 10 {
		maxCtx = 10
	}

	pre := strings.Join(args[:i], " ")
	post := strings.Join(args[i+1:], " ")

	if len(pre) > maxCtx {
		pre = "..." + pre[len(pre)-maxCtx:]
	}
	if len(post) > maxCtx {
		post = post[:maxCtx] + "..."
	}

	pointer := strings.Repeat("^", len(code))
	spacing := strings.Repeat(" ", len(pre)+1)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n  %s %s %s\n", pre, code, post))
	sb.WriteString(fmt.Sprintf("  %s%s\n", spacing, pointer))

	if jsCode != code {
		sb.WriteString(fmt.Sprintf("\n%s\n", jsCode))
	}

	sb.WriteString(fmt.Sprintf("\n%s\n", errPlaceholder))

	return sb.String()
}
