package reducer

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	. "github.com/antonmedv/fx/pkg/json"
	. "github.com/antonmedv/fx/pkg/theme"
)

func GenerateCode(lang string, args []string, fxrc string) string {
	switch lang {
	case "js":
		return js(args, fxrc)
	case "node":
		return nodejs(args, fxrc)
	case "python", "python3":
		return python(args)
	case "ruby":
		return ruby(args)
	default:
		panic("unknown lang")
	}
}

func Reduce(input interface{}, lang string, args []string, theme Theme, fxrc string) int {
	path, ok := splitPath(args)
	if ok {
		output := getByPath(input, path)
		echo(output, theme)
		return 0
	}
	var cmd *exec.Cmd
	switch lang {
	case "node":
		cmd = CreateNodejs(args, fxrc)
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
	object, err := Parse(dec)
	if err != nil {
		fmt.Print(string(output))
		return 0
	}
	echo(object, theme)
	if dec.InputOffset() < int64(len(output)) {
		fmt.Print(string(output[dec.InputOffset():]))
	}
	return 0
}
