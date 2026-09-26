package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const argvSep = "\x1f"

// runFxArgs runs fx with args as given and stdin from /dev/null.
func runFxArgs(t *testing.T, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), "FX_TEST_RUN_MAIN=1", "FX_TEST_ARGV="+strings.Join(args, argvSep))
	devNull, err := os.Open(os.DevNull)
	require.NoError(t, err)
	t.Cleanup(func() { devNull.Close() })
	cmd.Stdin = devNull
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	err = cmd.Run()
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	} else {
		require.NoError(t, err)
	}
	return out.String(), errOut.String(), exitCode
}

// save rewrites the file the parser is still reading. The rest of the input
// must be the original file, not the new content from the old offset.
func TestSaveFileMode(t *testing.T) {
	tests := []struct {
		name, input, query, want string
	}{
		{"longer", `{"name":"a"}` + "\n", `x.name = x.name.toUpperCase(), x`, "{\n  \"name\": \"A\"\n}\n"},
		{"longer number", `{"b":1}` + "\n", `x.b = 7, x`, "{\n  \"b\": 7\n}\n"},
		{"spread", `{"b":1}` + "\n", `{...x, b: 8}`, "{\n  \"b\": 8\n}\n"},
		{"shorter", `{"a":1,"b":2,"c":3,"d":4}`, `({a: x.a})`, "{\n  \"a\": 1\n}\n"},
		{"shorter pretty", "{\n    \"a\": 1,\n    \"long\": \"xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx\"\n}\n", `({a: x.a})`, "{\n  \"a\": 1\n}\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "f.json")
			require.NoError(t, os.WriteFile(file, []byte(tt.input), 0o644))

			stdout, stderr, code := runFxArgs(t, file, tt.query, "save")
			require.Equal(t, 0, code, stderr)
			require.Empty(t, stderr)
			require.Equal(t, tt.want, stdout)
			data, err := os.ReadFile(file)
			require.NoError(t, err)
			require.Equal(t, tt.want, string(data))
		})
	}
}

func TestSaveKeepsFileMode(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "f.json")
	require.NoError(t, os.WriteFile(file, []byte(`{"a":1}`), 0o600))
	require.NoError(t, os.Chmod(file, 0o600))

	_, stderr, code := runFxArgs(t, file, `x.a = 2, x`, "save")
	require.Equal(t, 0, code, stderr)
	info, err := os.Stat(file)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1, "temp file left behind")
}

func TestSaveRefusesSeveralValues(t *testing.T) {
	tests := []struct {
		name, file, input string
		args              []string
	}{
		{"json lines", "f.jsonl", "{\"a\":1}\n{\"a\":2}\n", nil},
		{"yaml stream", "f.yaml", "a: 1\n---\na: 2\n", nil},
		{"raw lines", "f.txt", "a\nb\n", []string{"-r"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), tt.file)
			require.NoError(t, os.WriteFile(file, []byte(tt.input), 0o644))

			args := append(tt.args, file, "save")
			stdout, stderr, code := runFxArgs(t, args...)
			require.Equal(t, 1, code)
			require.Empty(t, stdout)
			require.Contains(t, stderr, "save supports a single JSON value")
			data, err := os.ReadFile(file)
			require.NoError(t, err)
			require.Equal(t, tt.input, string(data), "file must be untouched")
		})
	}
}

// Retrying save() after the lookahead failed must not overwrite the input.
func TestSaveRefusesMalformedRest(t *testing.T) {
	const input = `{"a":1} /* unfinished`
	file := filepath.Join(t.TempDir(), "f.json")
	require.NoError(t, os.WriteFile(file, []byte(input), 0o644))

	stdout, stderr, code := runFxArgs(t, file, `x => { try { save(x) } catch (e) {} return save(x) }`)
	require.Equal(t, 1, code)
	require.Empty(t, stdout)
	require.Contains(t, stderr, "Unexpected end of input in comment")
	data, err := os.ReadFile(file)
	require.NoError(t, err)
	require.Equal(t, input, string(data), "file must be untouched")
}

// More() reaching the end must keep it: save() then closes the file.
func TestSaveRawSingleLine(t *testing.T) {
	file := filepath.Join(t.TempDir(), "f.txt")
	require.NoError(t, os.WriteFile(file, []byte("hello\n"), 0o644))

	stdout, stderr, code := runFxArgs(t, "-r", file, "save")
	require.Equal(t, 0, code, stderr)
	require.Empty(t, stderr)
	require.Equal(t, "hello\n", stdout)
	data, err := os.ReadFile(file)
	require.NoError(t, err)
	require.Equal(t, "\"hello\"\n", string(data))
}
