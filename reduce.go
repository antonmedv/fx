package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

//go:embed reduce.js
var template string

var flatMapRegex = regexp.MustCompile("^(\\.\\w*)+\\[]")

func generateCode(args []string) string {
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
			strings.Repeat(" ", width(pre)),
			strings.Repeat("^", width(a)),
			strings.Repeat(" ", width(post)),
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

func reduce(object interface{}, args []string) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("node", "-e", generateCode(args))
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "NODE_OPTIONS=--max-old-space-size=8192")
	cmd.Stdin = strings.NewReader(stringify(object))
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		dec := json.NewDecoder(&stdout)
		dec.UseNumber()
		jsonObject, err := parse(dec)
		if err == nil {
			if str, ok := jsonObject.(string); ok {
				fmt.Println(str)
			} else {
				fmt.Println(prettyPrint(jsonObject, 1))
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

func prettyPrint(v interface{}, level int) string {
	ident := strings.Repeat("  ", level)
	subident := strings.Repeat("  ", level-1)
	switch v.(type) {
	case nil:
		return colors.null.Render("null")

	case bool:
		if v.(bool) {
			return colors.boolean.Render("true")
		} else {
			return colors.boolean.Render("false")
		}

	case number:
		return colors.number.Render(v.(number).String())

	case string:
		return colors.string.Render(fmt.Sprintf("%q", v))

	case *dict:
		keys := v.(*dict).keys
		if len(keys) == 0 {
			return colors.syntax.Render("{}")
		}
		output := colors.syntax.Render("{\n")
		for i, k := range keys {
			key := colors.key.Render(fmt.Sprintf("%q", k))
			value, _ := v.(*dict).get(k)
			delim := ": "
			line := ident + key + delim + prettyPrint(value, level+1)
			if i < len(keys)-1 {
				line += ",\n"
			} else {
				line += "\n"
			}
			output += line
		}
		return output + subident + colors.syntax.Render("}")

	case array:
		slice := v.(array)
		if len(slice) == 0 {
			return colors.syntax.Render("[]")
		}
		output := colors.syntax.Render("[\n")
		for i, value := range v.(array) {
			line := ident + prettyPrint(value, level+1)
			if i < len(slice)-1 {
				line += ",\n"
			} else {
				line += "\n"
			}
			output += line
		}
		return output + subident + colors.syntax.Render("]")

	default:
		return "unknown type"
	}
}
