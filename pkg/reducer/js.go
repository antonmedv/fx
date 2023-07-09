package reducer

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	. "github.com/antonmedv/fx/pkg/json"
	. "github.com/antonmedv/fx/pkg/theme"
	"github.com/dop251/goja"
)

//go:embed js.js
var templateJs string

func js(args []string, fxrc string) string {
	rs := "\n"
	for i, a := range args {
		rs += "  try {"
		switch {
		case flatMapRegex.MatchString(a):
			code := fold(strings.Split(a, "[]"))
			rs += fmt.Sprintf(
				`
    x = (
      %v
    )(x)
`, code)

		case strings.HasPrefix(a, ".["):
			rs += fmt.Sprintf(
				`
    x = function () 
      { return this%v } 
    .call(x)
`, a[1:])

		case strings.HasPrefix(a, "."):
			rs += fmt.Sprintf(
				`
    x = function () 
      { return this%v } 
    .call(x)
`, a)

		default:
			rs += fmt.Sprintf(
				`
    let f = function () 
      { return %v }
    .call(x)
    x = typeof f === 'function' ? f(x) : f
`, a)
		}
		// Generate a beautiful error message.
		rs += "  } catch (e) {\n"
		pre, post, pointer := trace(args, i)
		rs += fmt.Sprintf(
			"    throw `\\n  ${%q} ${%q} ${%q}\\n  %v\\n\\n${e.stack || e}`\n",
			pre, a, post, pointer,
		)
		rs += "  }\n"
	}

	return fmt.Sprintf(templateJs, fxrc, rs)
}

func CreateJS(args []string, fxrc string) (*goja.Runtime, goja.Callable, error) {
	vm := goja.New()
	_, err := vm.RunString(js(args, fxrc))
	if err != nil {
		return nil, nil, err
	}
	fn, ok := goja.AssertFunction(vm.Get("reduce"))
	if !ok {
		panic("Not a function")
	}
	return vm, fn, nil
}

func ReduceJS(vm *goja.Runtime, reduce goja.Callable, input interface{}, theme Theme) int {
	value, err := reduce(goja.Undefined(), vm.ToValue(Stringify(input)))
	if err != nil {
		fmt.Println(err)
		return 1
	}
	output := value.String()
	dec := json.NewDecoder(strings.NewReader(output))
	dec.UseNumber()
	// We don't need comments when reducing.
	object, _, err := Parse(dec, "")
	if err != nil {
		fmt.Print(output)
		return 0
	}
	Echo(object, theme)
	return 0
}
