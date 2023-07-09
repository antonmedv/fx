package main

import (
	"encoding/json"
	"fmt"
	"io"

	. "github.com/antonmedv/fx/pkg/json"
	. "github.com/antonmedv/fx/pkg/reducer"
	. "github.com/antonmedv/fx/pkg/theme"
	"github.com/dop251/goja"
)

func stream(dec *json.Decoder, object interface{}, lang string, args []string, theme Theme, fxrc string) int {
	var vm *goja.Runtime
	var fn goja.Callable
	var err error
	if lang == "js" {
		vm, fn, err = CreateJS(args, fxrc)
		if err != nil {
			fmt.Println(err)
			return 1
		}
	}
	for {
		if object != nil {
			if lang == "js" {
				ReduceJS(vm, fn, object, theme)
			} else {
				Reduce(object, lang, args, theme, fxrc)
			}
		}
		// Streaming doesn't support interactive mode (and thus comments), so we pass an empty string ignore them.
		object, _, err = Parse(dec, "")
		if err == io.EOF {
			return 0
		}
		if err != nil {
			fmt.Println("JSON Parse Error:", err.Error())
			return 1
		}
	}
}
