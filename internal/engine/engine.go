package engine

import (
	_ "embed"
	"io"
	"reflect"
	"strings"

	"github.com/dop251/goja"

	"github.com/antonmedv/fx/internal/jsonx"
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

type Error struct {
	error string
}

func (e *Error) Error() string {
	return e.error
}

func Start(parser Parser, args []string, out chan *jsonx.Node, errCh chan error, cancel <-chan struct{}) int {
	isPrettyPrintArg := len(args) == 1 && (args[0] == "." || args[0] == "this" || args[0] == "x")

	// Fast path.
	if isPrettyPrintArg {
		for {
			node, err := parser.Parse()

			if err != nil {
				if err == io.EOF {
					break
				}
				errCh <- err
				return 1
			}

			out <- node
		}

		return 0
	}

	for i := range args {
		if err := validateSyntax(args, i); err != nil {
			jsCode := transpile(args[i])
			snippet := formatErr(args, i, jsCode)
			message := gojaErrorToString(err)
			errCh <- &Error{snippet + message}
			return 1
		}
	}

	var code strings.Builder
	code.WriteString(Stdlib)
	code.WriteString(JS(args))

	vm := NewVM(func(s string) {
		out <- &jsonx.Node{Kind: jsonx.Err, Value: s}
	})
	if _, err := vm.RunString(code.String()); err != nil {
		errCh <- &Error{gojaErrorToString(err)}
		return 1
	}

	skip := vm.Get("skip")
	undefined := vm.Get("undefined")
	main, _ := goja.AssertFunction(vm.Get("__main__"))

	echo := func(output goja.Value) {
		rtype := output.ExportType()
		if output.StrictEquals(undefined) {
			errCh <- &Error{"undefined"}
		} else if rtype != nil && rtype.Kind() == reflect.String {
			out <- &jsonx.Node{Kind: jsonx.String, Value: output.String()}
		} else {
			jsonOut := Stringify(output, vm, 0)
			nodeOut, err := jsonx.Parse([]byte(jsonOut))
			if err != nil {
				panic(err)
			}
			out <- nodeOut
		}
	}

	for {
		node, err := parser.Parse()
		if err != nil {
			if err == io.EOF {
				break
			}
			errCh <- err
			return 1
		}

		input := node.ToValue(vm)
		output, exitCode, err := callMain(main, input)
		if exitCode >= 0 {
			return exitCode
		}
		if err != nil {
			errCh <- &Error{gojaErrorToString(err)}
			return 1
		}

		if output.StrictEquals(skip) {
			continue
		}
		echo(output)
	}

	return 0
}

func callMain(main goja.Callable, input goja.Value) (output goja.Value, exitCode int, err error) {
	exitCode = -1
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(ExitError); ok {
				exitCode = e.Code
			} else {
				panic(r)
			}
		}
	}()
	output, err = main(goja.Undefined(), input)
	return
}

func validateSyntax(args []string, i int) error {
	var code strings.Builder
	code.WriteString("\nfunction __main__(json) {\n")
	code.WriteString(Body(args, i))
	code.WriteString("  return json\n}\n")

	vm := goja.New()
	_, err := vm.RunString(code.String())
	return err
}
