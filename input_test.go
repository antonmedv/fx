package main

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestChooseInput(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "t.json")
	require.NoError(t, os.WriteFile(file, []byte(`{"b":1}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "keys"), []byte(`{}`), 0o644))
	t.Chdir(dir)

	const (
		tty     = "tty"
		devNull = "/dev/null"
		pipe    = "pipe" // empty pipe, pipe with data or open pipe: all look the same here
	)
	tests := []struct {
		name  string
		stdin string
		args  []string
		want  inputSource
	}{
		{"tty no args", tty, nil, inputUsage},
		{"tty file", tty, []string{"t.json", ".b"}, inputFile},
		{"tty missing file", tty, []string{"missing.json"}, inputFile},
		{"tty bare identifier file", tty, []string{"keys"}, inputFile},
		{"dev null no args", devNull, nil, inputUsage},
		{"dev null file", devNull, []string{"t.json", ".b"}, inputFile},
		{"pipe no args", pipe, nil, inputStdin},
		{"pipe file", pipe, []string{"t.json", ".b"}, inputFile},
		{"pipe file only", pipe, []string{"t.json"}, inputFile},
		{"pipe absolute file", pipe, []string{file, ".b"}, inputFile},
		{"pipe code", pipe, []string{".name"}, inputStdin},
		{"pipe nonexistent path", pipe, []string{"missing.json", ".b"}, inputStdin},
		{"pipe directory", pipe, []string{dir}, inputStdin},
		{"pipe dot directory", pipe, []string{"."}, inputStdin},
		{"pipe bare identifier file", pipe, []string{"keys"}, inputStdin},
		{"pipe dotted bare identifier file", pipe, []string{"./keys"}, inputFile},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := chooseInput(tt.stdin == tty, tt.stdin == pipe, tt.args, os.Stat)
			require.Equal(t, tt.want, got)
		})
	}
}

// TestMain lets tests run the fx binary by re-executing the test binary.
func TestMain(m *testing.M) {
	if os.Getenv("FX_TEST_RUN_MAIN") == "1" {
		args := strings.Fields(os.Getenv("FX_TEST_ARGS"))
		if argv, ok := os.LookupEnv("FX_TEST_ARGV"); ok {
			args = strings.Split(argv, argvSep) // Args may contain spaces.
		}
		os.Args = append([]string{"fx"}, args...)
		main()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func runFx(t *testing.T, stdin func(cmd *exec.Cmd), args ...string) (string, error) {
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), "FX_TEST_RUN_MAIN=1", "FX_TEST_ARGS="+strings.Join(args, " "))
	stdin(cmd)
	done := make(chan struct{})
	var out []byte
	var err error
	go func() {
		out, err = cmd.CombinedOutput()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("fx hung")
	}
	return strings.TrimSpace(string(out)), err
}

func TestFileArgWithNonTtyStdin(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "t.json")
	require.NoError(t, os.WriteFile(file, []byte(`{"b":1}`), 0o644))

	tests := []struct {
		name  string
		stdin func(t *testing.T, cmd *exec.Cmd)
	}{
		{"dev null", func(t *testing.T, cmd *exec.Cmd) {
			f, err := os.Open(os.DevNull)
			require.NoError(t, err)
			t.Cleanup(func() { f.Close() })
			cmd.Stdin = f
		}},
		{"empty pipe", func(t *testing.T, cmd *exec.Cmd) {
			cmd.Stdin = strings.NewReader("")
		}},
		{"pipe with data", func(t *testing.T, cmd *exec.Cmd) {
			cmd.Stdin = strings.NewReader(`{"x":1}`)
		}},
		{"pipe left open", func(t *testing.T, cmd *exec.Cmd) {
			r, w, err := os.Pipe()
			require.NoError(t, err)
			t.Cleanup(func() { r.Close(); w.Close() })
			cmd.Stdin = r
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := runFx(t, func(cmd *exec.Cmd) { tt.stdin(t, cmd) }, file, ".b")
			require.NoError(t, err, out)
			require.Equal(t, "1", out)
		})
	}
}

func TestPipedStdinStillWorks(t *testing.T) {
	out, err := runFx(t, func(cmd *exec.Cmd) {
		cmd.Stdin = io.NopCloser(strings.NewReader(`{"name":"fx"}`))
	}, ".name")
	require.NoError(t, err, out)
	require.Equal(t, "fx", out)
}
