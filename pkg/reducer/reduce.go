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
	path, ok := SplitSimplePath(args)
	if ok {
		output := GetBySimplePath(input, path)
		Echo(output, theme)
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

	var stdout, stderr bytes.Buffer

	// TODO: Reimplement stringify with io.Reader.
	cmd.Stdin = strings.NewReader(Stringify(input))
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		exitCode := 1
		status, ok := err.(*exec.ExitError)
		if ok {
			exitCode = status.ExitCode()
		}
		fmt.Print(string(stderr.Bytes()))
		return exitCode
	}

	output := stdout.Bytes()

	lastLineIdx := bytes.LastIndexByte(output, byte('\n'))
	customOutput := output[:lastLineIdx]
	objectOutput := output[lastLineIdx+1:]

	fmt.Print(string(customOutput))

	if len(objectOutput) > 0 {
		dec := json.NewDecoder(bytes.NewReader(objectOutput))
		dec.UseNumber()

		object, err := Parse(dec)
		if err != nil {
			fmt.Println(string(objectOutput))
		} else {
			Echo(object, theme)
		}
	}

	return 0
}
