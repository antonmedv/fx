package reducer

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
)

func CreateNodejs(args []string, fxrc string) *exec.Cmd {
	cmd := exec.Command("node", "--input-type=module", "-e", nodejs(args, fxrc))
	nodePath, exist := os.LookupEnv("NODE_PATH")
	if exist {
		cmd.Dir = path.Dir(nodePath)
	}
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "NODE_OPTIONS=--max-old-space-size=8192")
	workingDir, err := os.Getwd()
	if err == nil {
		cmd.Env = append(cmd.Env, "FX_CWD="+workingDir)
	}
	return cmd
}

//go:embed node.js
var templateNode string

func nodejs(args []string, fxrc string) string {
	rs := "\n"
	for i, a := range args {
		rs += "  try {"
		switch {
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
    let f = function () 
      { return %v }
    .call(x)
    x = typeof f === 'function' ? f(x) : f
`, a)
		}
		rs += `
    x = await x
`
		// Generate a beautiful error message.
		rs += "  } catch (e) {\n"
		pre, post, pointer := trace(args, i)
		rs += fmt.Sprintf(
			"    throw `\\n  ${%q} ${%q} ${%q}\\n  %v\\n\\n${e.stack || e}`\n",
			pre, a, post, pointer,
		)
		rs += "  }\n"
	}

	return fmt.Sprintf(templateNode, fxrc, rs)
}
