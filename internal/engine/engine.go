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
	// More reports whether input remains after the values parsed so far,
	// blocking until more input or EOF arrives. An error in the remaining
	// input is returned by every later More and Parse call.
	More() (bool, error)
}

type Error struct {
	error string
}

func (e *Error) Error() string {
	return e.error
}

func Start(parser Parser, args []string, out chan *jsonx.Node, errCh chan error, cancel <-chan struct{}) int {
	return start(parser, args, out, errCh, cancel, false)
}

// StartPreview is Start for live previews: save() and exit() are disabled,
// as a preview runs while the user is still typing the expression.
func StartPreview(parser Parser, args []string, out chan *jsonx.Node, errCh chan error, cancel <-chan struct{}) int {
	return start(parser, args, out, errCh, cancel, true)
}

func start(parser Parser, args []string, out chan *jsonx.Node, errCh chan error, cancel <-chan struct{}, preview bool) int {
	isPrettyPrintArg := len(args) == 1 && (args[0] == "." || args[0] == "this" || args[0] == "x")

	// Fast path.
	if isPrettyPrintArg {
		for {
			select {
			case <-cancel:
				return 0
			default:
			}

			node, err := parser.Parse()

			if err != nil {
				if err == io.EOF {
					break
				}
				sendErr(errCh, err, cancel)
				return 1
			}

			if !send(out, node, cancel) {
				return 0
			}
		}

		return 0
	}

	for i := range args {
		if err := validateSyntax(args, i); err != nil {
			jsCode := transpile(args[i])
			snippet := formatErr(args, i, jsCode)
			message := gojaErrorToString(err)
			sendErr(errCh, &Error{snippet + message}, cancel)
			return 1
		}
	}

	var code strings.Builder
	code.WriteString(Stdlib)
	code.WriteString(JS(args))

	values := 0 // JSON values parsed so far.
	severalValues := func() (bool, error) {
		if values > 1 {
			return true, nil
		}
		return parser.More()
	}

	vm := newVM(func(s string) {
		send(out, &jsonx.Node{Kind: jsonx.Err, Value: s}, cancel)
	}, preview, severalValues)

	// Interrupt running JS on cancel: cancel alone is checked only between
	// documents, so a never-ending expression would never stop.
	finished := make(chan struct{})
	defer close(finished)
	go func() {
		select {
		case <-cancel:
			vm.Interrupt("cancelled")
		case <-finished:
		}
	}()

	if _, err := vm.RunString(code.String()); err != nil {
		if isCancelled(cancel) {
			return 0
		}
		sendErr(errCh, &Error{gojaErrorToString(err)}, cancel)
		return 1
	}

	skip := vm.Get("skip")
	undefined := vm.Get("undefined")
	main, _ := goja.AssertFunction(vm.Get("__main__"))

	// echo returns stop = true and an exit code if Start must return.
	echo := func(output goja.Value) (exitCode int, stop bool) {
		rtype := output.ExportType()
		if output.StrictEquals(undefined) {
			if !sendErr(errCh, &Error{"undefined"}, cancel) {
				return 0, true
			}
		} else if rtype != nil && rtype.Kind() == reflect.String {
			if !send(out, &jsonx.Node{Kind: jsonx.String, Value: Quote(output.String()), LineNumber: 1}, cancel) {
				return 0, true
			}
		} else {
			jsonOut, exit, err := stringify(output, vm)
			if exit != nil {
				return exit.Code, true
			}
			if err != nil {
				if isCancelled(cancel) {
					return 0, true
				}
				sendErr(errCh, &Error{gojaErrorToString(err)}, cancel)
				return 1, true
			}
			nodeOut, err := jsonx.Parse([]byte(jsonOut))
			if err != nil {
				panic(err)
			}
			if !send(out, nodeOut, cancel) {
				return 0, true
			}
		}
		return 0, false
	}

	for {
		select {
		case <-cancel:
			return 0
		default:
		}

		node, err := parser.Parse()
		if err != nil {
			if err == io.EOF {
				break
			}
			sendErr(errCh, err, cancel)
			return 1
		}
		values++

		input := node.ToValue(vm)
		output, exit, err := callMain(main, input)
		if exit != nil {
			return exit.Code
		}
		if err != nil {
			if isCancelled(cancel) {
				return 0
			}
			sendErr(errCh, &Error{gojaErrorToString(err)}, cancel)
			return 1
		}

		if output.StrictEquals(skip) {
			continue
		}
		if exitCode, stop := echo(output); stop {
			return exitCode
		}
	}

	return 0
}

func isCancelled(cancel <-chan struct{}) bool {
	select {
	case <-cancel:
		return true
	default:
		return false
	}
}

// send delivers node to out, returns false if cancelled.
func send(out chan *jsonx.Node, node *jsonx.Node, cancel <-chan struct{}) bool {
	select {
	case out <- node:
		return true
	case <-cancel:
		return false
	}
}

// sendErr delivers err to errCh, returns false if cancelled.
func sendErr(errCh chan error, err error, cancel <-chan struct{}) bool {
	select {
	case errCh <- err:
		return true
	case <-cancel:
		return false
	}
}

// callMain runs main. exit is set if exit() was called, with any code.
func callMain(main goja.Callable, input goja.Value) (output goja.Value, exit *ExitError, err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(ExitError); ok {
				exit = &e
			} else {
				panic(r)
			}
		}
	}()
	output, err = main(goja.Undefined(), input)
	return
}

// stringify serializes output like callMain runs main: getters run JS here,
// which may throw, call exit() or be interrupted on cancel.
func stringify(output goja.Value, vm *goja.Runtime) (json string, exit *ExitError, err error) {
	defer func() {
		if r := recover(); r != nil {
			switch e := r.(type) {
			case ExitError:
				exit = &e
			case *goja.Exception:
				err = e
			case *goja.InterruptedError:
				err = e
			default:
				panic(r)
			}
		}
	}()
	json = Stringify(output, vm, 0)
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
