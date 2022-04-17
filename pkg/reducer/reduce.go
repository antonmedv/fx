package reducer

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	. "github.com/antonmedv/fx/pkg/json"
	. "github.com/antonmedv/fx/pkg/theme"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

//go:embed reduce.js
var template string

var flatMapRegex = regexp.MustCompile("^(\\.\\w*)+\\[]")

func GenerateCode(args []string) string {
	rs := "\n"
	for i, a := range args {
		rs += "  try {"
		switch {
		case a == ".":
			rs += `
    json = function () 
      { return this }
    .call(json)
`

		case flatMapRegex.MatchString(a):
			code := fold(strings.Split(a, "[]"))
			rs += fmt.Sprintf(
				`
    json = (
      %v
    )(json)
`, code)

		case strings.HasPrefix(a, ".["):
			rs += fmt.Sprintf(
				`
    json = function () 
      { return this%v } 
    .call(json)
`, a[1:])

		case strings.HasPrefix(a, "."):
			rs += fmt.Sprintf(
				`
    json = function () 
      { return this%v } 
    .call(json)
`, a)

		default:
			rs += fmt.Sprintf(
				`
    fn = function () 
      { return %v }
    .call(json)
    json = typeof fn === 'function' ? json = fn(json) : fn
`, a)
		}
		// Generate a beautiful error message.
		rs += "  } catch (e) {\n"
		pre := strings.Join(args[:i], " ")
		if len(pre) > 20 {
			pre = "..." + pre[len(pre)-20:]
		}
		post := strings.Join(args[i+1:], " ")
		if len(post) > 20 {
			post = post[:20] + "..."
		}
		pointer := fmt.Sprintf(
			"%v %v %v",
			strings.Repeat(" ", len(pre)),
			strings.Repeat("^", len(a)),
			strings.Repeat(" ", len(post)),
		)
		rs += fmt.Sprintf(
			"    throw `\\n"+
				"  ${%q} ${%q} ${%q}\\n"+
				"  %v\\n"+
				"\\n${e.stack || e}`\n",
			pre, a, post,
			pointer,
		)
		rs += "  }\n"
	}
	return fmt.Sprintf(template, rs)
}

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

func Reduce(object interface{}, args []string, theme Theme) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("node", "-e", GenerateCode(args))
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "NODE_OPTIONS=--max-old-space-size=8192")
	cmd.Stdin = strings.NewReader(Stringify(object))
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		dec := json.NewDecoder(&stdout)
		dec.UseNumber()
		jsonObject, err := Parse(dec)
		if err == nil {
			if str, ok := jsonObject.(string); ok {
				fmt.Println(str)
			} else {
				fmt.Println(PrettyPrint(jsonObject, 1, theme))
			}
		} else {
			_, _ = fmt.Fprint(os.Stderr, stderr.String())
		}
	} else {
		exitCode := 1
		status, ok := err.(*exec.ExitError)
		if ok {
			exitCode = status.ExitCode()
		}
		_, _ = fmt.Fprint(os.Stderr, stderr.String())
		os.Exit(exitCode)
	}
}
