package reducer

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

func CreateNodejs(args []string) *exec.Cmd {
	cmd := exec.Command("node", "-e", nodejs(args))
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "NODE_OPTIONS=--max-old-space-size=8192")
	return cmd
}

//go:embed reduce.js
var templateJs string

func nodejs(args []string) string {
	rs := "\n"
	for i, a := range args {
		rs += "  try {"
		switch {
		case a == ".":
			rs += `
    x = function () 
      { return this }
    .call(x)
`

		case flatMapRegex.MatchString(a):
			code := fold(strings.Split(a, "[]"))
			rs += fmt.Sprintf(
				`
    x = (
      %v
    )(x)
`, code)

		case strings.HasPrefix(a, ".["):
			rs += fmt.Sprintf(
				`
    x = function () 
      { return this%v } 
    .call(x)
`, a[1:])

		case strings.HasPrefix(a, "."):
			rs += fmt.Sprintf(
				`
    x = function () 
      { return this%v } 
    .call(x)
`, a)

		default:
			rs += fmt.Sprintf(
				`
    f = function () 
      { return %v }
    .call(x)
    x = typeof f === 'function' ? f(x) : fn
`, a)
		}
		// Generate a beautiful error message.
		rs += "  } catch (e) {\n"
		pre, post, pointer := trace(args, i)
		rs += fmt.Sprintf(
			"    throw `\\n  ${%q} ${%q} ${%q}\\n  %v\\n\\n${e.stack || e}`\n",
			pre, a, post, pointer,
		)
		rs += "  }\n"
	}
	return fmt.Sprintf(templateJs, rs)
}

var flatMapRegex = regexp.MustCompile("^(\\.\\w*)+\\[]")

func fold(s []string) string {
	if len(s) == 1 {
		return "x => x" + s[0]
	}
	obj := s[0]
	if obj == "." {
		obj = "x"
	} else {
		obj = "x" + obj
	}
	return fmt.Sprintf("x => Object.values(%v).flatMap(%v)", obj, fold(s[1:]))
}
