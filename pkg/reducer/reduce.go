package reducer

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	. "github.com/antonmedv/fx/pkg/json"
	. "github.com/antonmedv/fx/pkg/theme"
)

func GenerateCode(args []string) string {
	lang, ok := os.LookupEnv("FX_LANG")
	if !ok {
		lang = "node"
	}
	switch lang {
	case "node":
		return nodejs(args)
	case "python", "python3":
		return python(args)
	case "ruby":
		return ruby(args)
	default:
		panic("unknown lang")
	}
}

func Reduce(object interface{}, args []string, theme Theme) {
	var cmd *exec.Cmd
	lang, ok := os.LookupEnv("FX_LANG")
	if !ok {
		lang = "node"
	}
	switch lang {
	case "node":
		cmd = CreateNodejs(args)
	case "python", "python3":
		cmd = CreatePython(lang, args)
	case "ruby":
		cmd = CreateRuby(args)
	default:
		panic("unknown lang")
	}

	var stdout, stderr bytes.Buffer
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

func trace(args []string, i int) (pre, post, pointer string) {
	pre = strings.Join(args[:i], " ")
	if len(pre) > 20 {
		pre = "..." + pre[len(pre)-20:]
	}
	post = strings.Join(args[i+1:], " ")
	if len(post) > 20 {
		post = post[:20] + "..."
	}
	pointer = fmt.Sprintf(
		"%v %v %v",
		strings.Repeat(" ", len(pre)),
		strings.Repeat("^", len(args[i])),
		strings.Repeat(" ", len(post)),
	)
	return
}
