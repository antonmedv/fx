package engine

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/dop251/goja"
	"github.com/goccy/go-yaml"
)

// FilePath is the file being processed, empty if stdin.
var FilePath string

var (
	inputMu sync.Mutex
	input   io.Closer
)

// SetInput sets the open FilePath. save() closes it before replacing the
// file, as Windows can't rename over an open file. The UI replaces it on
// reload while an engine may run save(), hence the mutex.
func SetInput(c io.Closer) {
	inputMu.Lock()
	defer inputMu.Unlock()
	input = c
}

// CloseInput closes the open FilePath, so it can be replaced on Windows.
func CloseInput() {
	inputMu.Lock()
	defer inputMu.Unlock()
	if input != nil {
		_ = input.Close()
	}
}

// ExitError is used by exit() to signal a specific exit code.
type ExitError struct {
	Code int
}

// NewVM creates a runtime with fx bindings. In preview, save() and exit()
// fail: a preview must neither write the file nor stop with partial output.
func NewVM(writeOut func(string), preview bool) *goja.Runtime {
	return newVM(writeOut, preview, nil)
}

// newVM is NewVM for Start: severalValues reports whether the input holds
// more than one JSON value, in which case save() fails. Without it save()
// always fails.
func newVM(writeOut func(string), preview bool, severalValues func() (bool, error)) *goja.Runtime {
	vm := goja.New()

	if err := vm.Set("println", func(s string) any {
		writeOut(s)
		return nil
	}); err != nil {
		panic(err)
	}

	if err := vm.Set("__save__", func(json string) error {
		if preview {
			return fmt.Errorf("save is disabled in preview, press enter to apply")
		}
		if FilePath == "" {
			return fmt.Errorf("specify a file as the first argument to be able to save: fx file.json ")
		}
		mode, err := saveMode(FilePath)
		if err != nil {
			return err
		}
		// save() replaces the whole file with one value. With several values
		// (JSON Lines, a YAML stream) each would overwrite the file in turn,
		// losing the rest, so refuse before writing anything.
		if severalValues == nil {
			return fmt.Errorf("save is not available here")
		}
		several, err := severalValues()
		if err != nil {
			return err
		}
		if several {
			return fmt.Errorf("save supports a single JSON value, but %s contains several", FilePath)
		}
		// All input is read, so the parser no longer needs the file.
		CloseInput()
		return writeFileAtomic(FilePath, []byte(json), mode)
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

	if err := vm.Set("__exit__", func(code int) error {
		if preview {
			return fmt.Errorf("exit is disabled in preview")
		}
		panic(ExitError{Code: code})
	}); err != nil {
		panic(err)
	}

	return vm
}

// WriteFile replaces the file at path with data, keeping its mode. It is
// what save() does to FilePath, for the UI.
func WriteFile(path string, data []byte) error {
	mode, err := saveMode(path)
	if err != nil {
		return err
	}
	return writeFileAtomic(path, data, mode)
}

// saveMode returns the mode of the file save() may replace. A missing file
// gets 0644.
func saveMode(path string) (os.FileMode, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return 0644, nil
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return 0, fmt.Errorf("cannot save to a symbolic link: %s", path)
	}
	if !info.Mode().IsRegular() {
		return 0, fmt.Errorf("cannot save to %s: not a regular file", path)
	}
	return info.Mode().Perm(), nil
}

// writeFileAtomic replaces path with data via a temp file in the same
// directory and a rename: a crash never leaves a half-written file, and the
// original inode is untouched for anyone still reading it.
func writeFileAtomic(path string, data []byte, mode os.FileMode) (err error) {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tmp.Close()
			_ = os.Remove(tmp.Name())
		}
	}()
	if _, err = tmp.Write(data); err != nil {
		return err
	}
	if err = tmp.Chmod(mode); err != nil {
		return err
	}
	if err = tmp.Sync(); err != nil {
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
