package engine

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func Transpile(args []string, i int) string {
	jsCode := transpile(args[i])
	snippet := formatErr(args, i, jsCode)
	return fmt.Sprintf(`  try {
    json = apply((function () {
      const x = this
      return %s
    }).call(json), json)
  } catch (e) {
    throw %s
  }

`, jsCode, strconv.Quote(snippet)+" + e.toString()")
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
