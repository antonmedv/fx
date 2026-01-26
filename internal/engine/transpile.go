package engine

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func JS(args []string) string {
	var code strings.Builder
	code.WriteString("\nfunction __main__(json) {\n")
	for i := range args {
		if args[i] == "" {
			// In autocomplete: after dropTail, we can have empty strings.
			continue
		}
		code.WriteString(Body(args, i))
	}
	code.WriteString("\n  return json\n}\n")
	return code.String()
}

func Body(args []string, i int) string {
	jsCode := transpile(args[i])
	snippet := formatErr(args, i, jsCode)
	return fmt.Sprintf(`
  try {
    json = apply((function () {
      const x = this
      return %s
    }).call(json), json)
  } catch (e) {
    throw %s
  }

  if (json === skip) return skip
`, jsCode, strconv.Quote(snippet)+" + e.toString()")
}

var (
	reBracket      = regexp.MustCompile(`^(\.\w*)+\[]`)
	reBracketStart = regexp.MustCompile(`^\.\[`)
	reDotStart     = regexp.MustCompile(`^\.`)
	reAt           = regexp.MustCompile(`^@`)
	reFilter       = regexp.MustCompile(`^\?`)
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

	if reAt.MatchString(code) {
		jsCode := transpile(code[1:])
		return fmt.Sprintf(`map((x, i) => apply(%s, x, i))`, jsCode)
	}

	if reFilter.MatchString(code) {
		jsCode := transpile(code[1:])
		return fmt.Sprintf(`filter((x, i) => apply(%s, x, i))`, jsCode)
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
