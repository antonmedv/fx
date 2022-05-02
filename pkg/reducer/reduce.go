package reducer

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	. "github.com/antonmedv/fx/pkg/dict"
	. "github.com/antonmedv/fx/pkg/json"
	. "github.com/antonmedv/fx/pkg/theme"
)

func GenerateCode(lang string, args []string) string {
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

func Reduce(object interface{}, lang string, args []string, theme Theme) int {
	path, ok := split(args)
	if ok {
		for _, get := range path {
			switch get := get.(type) {
			case string:
				switch o := object.(type) {
				case *Dict:
					object = o.Values[get]
				case string:
					if get == "length" {
						object = Number(strconv.Itoa(len([]rune(o))))
					} else {
						object = nil
					}
				case Array:
					if get == "length" {
						object = Number(strconv.Itoa(len(o)))
					} else {
						object = nil
					}
				default:
					object = nil
				}
			case int:
				switch o := object.(type) {
				case Array:
					object = o[get]
				default:
					object = nil
				}
			}
		}
		echo(object, theme)
		return 0
	}

	var cmd *exec.Cmd
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

	cmd.Stdin = strings.NewReader(Stringify(object))
	output, err := cmd.CombinedOutput()
	if err != nil {
		exitCode := 1
		status, ok := err.(*exec.ExitError)
		if ok {
			exitCode = status.ExitCode()
		} else {
			fmt.Println(err.Error())
		}
		fmt.Print(string(output))
		return exitCode
	}

	dec := json.NewDecoder(bytes.NewReader(output))
	dec.UseNumber()
	jsonObject, err := Parse(dec)
	if err != nil {
		fmt.Print(string(output))
		return 0
	}
	echo(jsonObject, theme)
	if dec.InputOffset() < int64(len(output)) {
		fmt.Print(string(output[dec.InputOffset():]))
	}
	return 0
}

func echo(object interface{}, theme Theme) {
	if s, ok := object.(string); ok {
		fmt.Println(s)
	} else {
		fmt.Println(PrettyPrint(object, 1, theme))
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
