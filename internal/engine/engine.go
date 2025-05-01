package engine

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/antonmedv/fx/internal/jsonx"
	"github.com/dop251/goja"
)

//go:embed stdlib.js
var Stdlib string

func Reduce(parser *jsonx.JsonParser, fns []string) {
	var code strings.Builder
	code.WriteString(Stdlib)
	code.WriteString("\nfunction __transform__(json) {\n")
	for _, fn := range fns {
		code.WriteString(Transform(fn))
	}
	code.WriteString("  return json\n}\n")
	code.WriteString("const __undefined__ = undefined\n")

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
	transform, ok := goja.AssertFunction(vm.Get("__transform__"))
	if !ok {
		panic("__transform__ function not found")
	}
	undefined := vm.Get("__undefined__")

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
		if output.StrictEquals(undefined) {
			fmt.Fprintln(os.Stderr, "undefined")
		} else if rtype != nil && rtype.Kind() == reflect.String {
			fmt.Println(output.ToString())
		} else {
			fmt.Println(PrettyPrint(output, vm, 0))
		}
	}
}
