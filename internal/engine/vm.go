package engine

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/dop251/goja"
	"github.com/goccy/go-yaml"
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
		return Stringify(x, vm, 0) + "\n"
	}); err != nil {
		panic(err)
	}

	if err := vm.Set("__toBase64__", func(x string) string {
		return base64.StdEncoding.EncodeToString([]byte(x))
	}); err != nil {
		panic(err)
	}

	if err := vm.Set("__fromBase64__", func(x string) (string, error) {
		decoded, err := base64.StdEncoding.DecodeString(x)
		if err != nil {
			return "", err
		}
		return string(decoded), err
	}); err != nil {
		panic(err)
	}

	if err := vm.Set("__yaml_parse__", func(x string) (string, error) {
		b, err := yaml.YAMLToJSON([]byte(x))
		if err != nil {
			return "", err
		}
		return string(b), err
	}); err != nil {
		panic(err)
	}

	if err := vm.Set("__yaml_stringify__", func(x goja.Value) string {
		b, err := yaml.JSONToYAML([]byte(Stringify(x, vm, 0)))
		if err != nil {
			return ""
		}
		return string(b)
	}); err != nil {
		panic(err)
	}

	return vm
}
