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
	fns []string,
	slurp bool,
	writeOut, writeErr func(string),
) int {
	isPrettyPrintArg := len(fns) == 1 && (fns[0] == "." || fns[0] == "this" || fns[0] == "x")

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
	for _, fn := range fns {
		code.WriteString(Transpile(fn))
	}
	code.WriteString("  return json\n}\n")

	vm := goja.New()
	if err := vm.Set("println", func(s string) any {
		writeOut(s)
		return nil
	}); err != nil {
		panic(err)
	}

	if _, err := vm.RunString(code.String()); err != nil {
		writeErr(err.Error())
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
				writeErr(err.Error())
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
