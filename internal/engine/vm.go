package engine

import (
	"fmt"
	"os"

	"github.com/dop251/goja"

	"github.com/antonmedv/fx/internal/theme"
)

// FilePath is the file being processed, empty if stdin.
var FilePath string

func NewVM(writeOut func(string)) *goja.Runtime {
	vm := goja.New()

	if err := vm.Set("println", func(s string) any {
		writeOut(s)
		return nil
	}); err != nil {
		panic(err)
	}

	if err := vm.Set("__save__", func(json string) error {
		if FilePath == "" {
			return fmt.Errorf("specify a file as the first argument to be able to save: fx file.json ")
		}
		if err := os.WriteFile(FilePath, []byte(json), 0644); err != nil {
			return err
		}
		return nil
	}); err != nil {
		panic(err)
	}

	if err := vm.Set("__stringify__", func(x goja.Value) string {
		return Stringify(x, vm, theme.NoColor, 0) + "\n"
	}); err != nil {
		panic(err)
	}

	return vm
}
