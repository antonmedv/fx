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
	"github.com/dop251/goja"
)

func GenerateCode(lang string, args []string) string {
	switch lang {
	case "js":
		return js(args)
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

func Reduce(input interface{}, lang string, args []string, theme Theme) int {
	// TODO: Move to separate function.
	path, ok := split(args)
	if ok {
		for _, get := range path {
			switch get := get.(type) {
			case string:
				switch o := input.(type) {
				case *Dict:
					input = o.Values[get]
				case string:
					if get == "length" {
						input = Number(strconv.Itoa(len([]rune(o))))
					} else {
						input = nil
					}
				case Array:
					if get == "length" {
						input = Number(strconv.Itoa(len(o)))
					} else {
						input = nil
					}
				default:
					input = nil
				}
			case int:
				switch o := input.(type) {
				case Array:
					input = o[get]
				default:
					input = nil
				}
			}
		}
		echo(input, theme)
		return 0
	}

	// TODO: Remove switch and this Reduce function.
	var cmd *exec.Cmd
	switch lang {
	case "js":
		vm := goja.New()
		_, err := vm.RunString(js(args))
		if err != nil {
			fmt.Println(err)
			return 1
		}
		// TODO: Do not evaluate reduce function on every message in stream.
		sum, ok := goja.AssertFunction(vm.Get("reduce"))
		if !ok {
			panic("Not a function")
		}
		res, err := sum(goja.Undefined(), vm.ToValue(Stringify(input)))
		if err != nil {
			fmt.Println(err)
			return 1
		}
		output := res.String()
		dec := json.NewDecoder(strings.NewReader(output))
		dec.UseNumber()
		jsonObject, err := Parse(dec)
		if err != nil {
			fmt.Print(output)
			return 0
		}
		echo(jsonObject, theme)
		return 0
	case "node":
		cmd = CreateNodejs(args)
	case "python", "python3":
		cmd = CreatePython(lang, args)
	case "ruby":
		cmd = CreateRuby(args)
	default:
		panic("unknown lang")
	}

	// TODO: Reimplement stringify with io.Reader.
	cmd.Stdin = strings.NewReader(Stringify(input))
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
