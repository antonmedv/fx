package engine

import (
	_ "embed"
	"io"
	"reflect"
	"strconv"
	"strings"

	"github.com/dop251/goja"

	"github.com/antonmedv/fx/internal/jsonx"
	"github.com/antonmedv/fx/internal/pretty"
)

//go:embed stdlib.js
var Stdlib string

func init() {
	fxrc, err := readFxrc()
	if err != nil {
		panic(err)
	}
	Stdlib += fxrc
}

type Parser interface {
	Parse() (*jsonx.Node, error)
	Recover() *jsonx.Node
}

func Start(
	parser Parser,
	args []string,
	slurp bool,
	withInline bool,
	writeOut, writeErr func(string),
) int {
	if slurp {
		var ok bool
		parser, ok = Slurp(parser, writeErr)
		if !ok {
			return 1
		}
	}

	isPrettyPrintArg := len(args) == 1 && (args[0] == "." || args[0] == "this" || args[0] == "x")

	// Fast path.
	if isPrettyPrintArg {
		for {
			node, err := parser.Parse()
			if err != nil {
				if err == io.EOF {
					break
				}
				writeErr(err.Error())
				return 1
			}

			if node.Kind == jsonx.String {
				unquoted, err := strconv.Unquote(string(node.Value))
				if err != nil {
					panic(err)
				}
				writeOut(unquoted)
			} else {
				writeOut(pretty.Print(node, withInline))
			}
		}

		return 0
	}

	for i := range args {
		if err := validateSyntax(args, i); err != nil {
			jsCode := transpile(args[i])
			snippet := formatErr(args, i, jsCode)
			message := errorToString(err)
			writeErr(snippet + message)
			return 1
		}
	}

	var code strings.Builder
	code.WriteString(Stdlib)
	code.WriteString("\nfunction __main__(json) {\n")
	for i := range args {
		code.WriteString(Transpile(args, i))
	}
	code.WriteString("  return json\n}\n")

	vm := NewVM(writeOut)
	if _, err := vm.RunString(code.String()); err != nil {
		writeErr(errorToString(err))
		return 1
	}

	skip := vm.Get("skip")
	undefined := vm.Get("undefined")
	main, _ := goja.AssertFunction(vm.Get("__main__"))

	echo := func(output goja.Value) {
		rtype := output.ExportType()
		if output.StrictEquals(undefined) {
			writeErr("undefined")
		} else if rtype != nil && rtype.Kind() == reflect.String {
			writeOut(output.String())
		} else {
			writeOut(Stringify(output, vm, 0))
		}
	}

	for {
		node, err := parser.Parse()
		if err != nil {
			if err == io.EOF {
				break
			}
			writeErr(err.Error())
			return 1
		}

		input := node.ToValue(vm)
		output, err := main(goja.Undefined(), input)
		if err != nil {
			writeErr(errorToString(err))
			return 1
		}

		if output.StrictEquals(skip) {
			continue
		}
		echo(output)
	}

	return 0
}

func validateSyntax(args []string, i int) error {
	var code strings.Builder
	code.WriteString("\nfunction __main__(json) {\n")
	code.WriteString(Transpile(args, i))
	code.WriteString("  return json\n}\n")

	vm := goja.New()
	_, err := vm.RunString(code.String())
	return err
}
