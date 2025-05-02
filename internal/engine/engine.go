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

func Start(parser *jsonx.JsonParser, fns []string, writeOut, writeErr func(string)) int {
	if len(fns) == 1 && (fns[0] == "." || fns[0] == "this" || fns[0] == "x") {
		// Fast path.
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
	code.WriteString("\nfunction __transform__(json) {\n")
	for _, fn := range fns {
		code.WriteString(Transform(fn))
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
	transform, ok := goja.AssertFunction(vm.Get("__transform__"))
	if !ok {
		panic("__transform__ function not found")
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
		output, err := transform(goja.Undefined(), input)
		if err != nil {
			writeErr(err.Error())
			return 1
		}

		if output.StrictEquals(skip) {
			continue
		}

		rtype := output.ExportType()
		if output.StrictEquals(undefined) {
			writeErr("undefined")
		} else if rtype != nil && rtype.Kind() == reflect.String {
			writeOut(output.String())
		} else {
			writeOut(Stringify(output, vm, 0))
		}
	}

	return 0
}
