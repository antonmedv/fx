package engine

import (
	_ "embed"
	"io"
	"reflect"
	"strconv"
	"strings"

	"github.com/antonmedv/fx/internal/jsonx"
	"github.com/dop251/goja"
)

//go:embed stdlib.js
var Stdlib string

type Parser interface {
	Parse() (*jsonx.Node, error)
	Recover() *jsonx.Node
}

func Start(
	parser Parser,
	args []string,
	slurp bool,
	writeOut, writeErr func(string),
) int {
	isPrettyPrintArg := len(args) == 1 && (args[0] == "." || args[0] == "this" || args[0] == "x")

	// Fast path.
	if isPrettyPrintArg && !slurp {
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
				writeOut(StringifyNode(node))
			}
		}

		return 0
	}

	var code strings.Builder
	code.WriteString(Stdlib)
	code.WriteString("\nfunction __main__(json) {\n")
	for i := range args {
		code.WriteString(Transpile(args, i))
	}
	code.WriteString("  return json\n}\n")
	println(code.String())

	vm := goja.New()
	if err := vm.Set("println", func(s string) any {
		writeOut(s)
		return nil
	}); err != nil {
		panic(err)
	}

	onError := func(err error) {
		if exception, ok := err.(*goja.Exception); ok {
			message := exception.Value().String()
			message = extractErrorMessage(message)
			writeErr(message)
			return
		}
		writeErr(err.Error())
	}

	if _, err := vm.RunString(code.String()); err != nil {
		onError(err)
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

	if slurp {
		var arr []any

		for {
			node, err := parser.Parse()
			if err != nil {
				if err == io.EOF {
					break
				}
				writeErr(err.Error())
				return 1
			}

			arr = append(arr, node.ToValue(vm))
		}

		input := vm.NewArray(arr...)
		output, err := main(goja.Undefined(), input)
		if err != nil {
			writeErr(err.Error())
			return 1
		}

		if output.StrictEquals(skip) {
			return 0
		}
		echo(output)

	} else {
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
				onError(err)
				return 1
			}

			if output.StrictEquals(skip) {
				continue
			}
			echo(output)
		}
	}

	return 0
}
