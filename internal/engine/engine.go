package engine

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/antonmedv/fx/internal/jsonx"
	"github.com/antonmedv/fx/internal/theme"
	"github.com/dop251/goja"
)

//go:embed stdlib.js
var Stdlib string

func Reduce(parser *jsonx.JsonParser, fns []string) {
	if len(fns) == 1 && (fns[0] == "." || fns[0] == "this" || fns[0] == "x") {
		// Fast path.
		for {
			node, err := parser.Parse()
			if err != nil {
				if err == io.EOF {
					break
				}
				println(err.Error())
				os.Exit(1)
			}

			fmt.Print(theme.PrintFullJson(node))
		}
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
		fmt.Println(s)
		return nil
	}); err != nil {
		panic(err)
	}

	if _, err := vm.RunString(code.String()); err != nil {
		println(err.Error())
		os.Exit(1)
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
			println(err.Error())
			os.Exit(1)
		}

		input := node.ToValue(vm)
		output, err := transform(goja.Undefined(), input)
		if err != nil {
			println(err.Error())
			os.Exit(1)
		}

		rtype := output.ExportType()
		if output.StrictEquals(skip) {
			continue
		} else if output.StrictEquals(undefined) {
			_, _ = fmt.Fprintln(os.Stderr, "undefined")
		} else if rtype != nil && rtype.Kind() == reflect.String {
			fmt.Println(output.ToString())
		} else {
			fmt.Println(Stringify(output, vm, 0))
		}
	}
}
