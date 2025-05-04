package engine

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/charmbracelet/x/term"
)

func Transpile(args []string, i int) string {
	jsCode := transpile(args[0])
	snippet := formatErr(args, i)
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

func formatErr(args []string, i int) string {
	// Determine terminal width, fallback to 80
	width, _, wErr := term.GetSize(os.Stdout.Fd())
	if wErr != nil || width <= 0 {
		width = 80
	}

	// Extract the code token and use it as JS code placeholder
	code := args[i]
	jsCode := code

	// Placeholder for actual error message
	errPlaceholder := "<error placeholder>"

	// Reserve space: 2 for indent, 1 for spaces, len(code), 1 space separators
	reserve := 2 + 1 + len(code) + 1 + 2
	// Split remaining width for pre and post context
	maxCtx := (width - reserve) / 2
	if maxCtx < 10 {
		maxCtx = 10
	}

	// Join parts before and after the error index
	pre := strings.Join(args[:i], " ")
	post := strings.Join(args[i+1:], " ")

	// Trim to at most maxCtx characters
	if len(pre) > maxCtx {
		pre = "..." + pre[len(pre)-maxCtx:]
	}
	if len(post) > maxCtx {
		post = post[:maxCtx] + "..."
	}

	// Build the pointer line under the code segment
	pointer := strings.Repeat("^", len(code))
	spacing := strings.Repeat(" ", len(pre)+1)

	// Assemble the error message
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n  %s %s %s\n", pre, code, post))
	sb.WriteString(fmt.Sprintf("  %s%s\n", spacing, pointer))

	// Include original JS code if it differs
	if jsCode != code {
		sb.WriteString(fmt.Sprintf("\n%s\n", jsCode))
	}

	// Append the placeholder error message
	sb.WriteString(fmt.Sprintf("\n%s\n", errPlaceholder))

	return sb.String()
}
