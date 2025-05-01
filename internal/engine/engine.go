package engine

import (
	_ "embed"
	"fmt"
	"io"
	"os"
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
	code.WriteString("\n  return json\n}\n")

	vm := goja.New()
	if err := vm.Set("println", func(s string) any {
		fmt.Println(s)
		return nil
	}); err != nil {
		panic(err)
	}

	codeStr := code.String()
	println(codeStr)
	if _, err := vm.RunString(codeStr); err != nil {
		println(err.Error())
		os.Exit(1)
	}
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

		println(output.String())
	}
}
