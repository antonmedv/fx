package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCLIProcess(t *testing.T) {
	if os.Getenv("FX_TEST_MAIN") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			os.Args = append([]string{"fx"}, os.Args[i+1:]...)
			main()
			os.Exit(0)
		}
	}
	t.Fatal("missing CLI argument separator")
}

func TestNoPaging(t *testing.T) {
	executable, err := os.Executable()
	require.NoError(t, err)
	for _, tc := range []struct {
		name  string
		args  []string
		env   []string
		input string
		file  bool
		want  string
		fail  bool
	}{
		{"flag stdin", []string{"--no-paging"}, nil, `{"a":1}`, false, "{\n  \"a\": 1\n}\n", false},
		{"environment stdin", nil, []string{"FX_NO_PAGER=1"}, `{"a":1}`, false, "{\n  \"a\": 1\n}\n", false},
		{"flag file", []string{"--no-paging"}, nil, `{"a":1}`, true, "{\n  \"a\": 1\n}\n", false},
		{"environment file", nil, []string{"FX_NO_PAGER=1"}, `{"a":1}`, true, "{\n  \"a\": 1\n}\n", false},
		{"transform", []string{".a"}, []string{"FX_NO_PAGER=1"}, `{"a":1}`, false, "1\n", false},
		{"stream", []string{"--no-paging"}, nil, "1\n2", false, "1\n2\n", false},
		{"raw", []string{"--no-paging", "--raw"}, nil, "hello\nworld", false, "hello\nworld\n", false},
		{"yaml", []string{"--no-paging", "--yaml"}, nil, "a: 1", false, "{\n  \"a\": 1\n}\n", false},
		{"slurp", []string{"--no-paging", "--slurp"}, nil, "1\n2", false, "[\n  1,\n  2\n]\n", false},
		{"invalid input", []string{"--no-paging", "--strict"}, nil, "{", false, "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			args := append([]string{"-test.run=^TestCLIProcess$", "--"}, tc.args...)
			if tc.file {
				path := filepath.Join(t.TempDir(), "input.json")
				require.NoError(t, os.WriteFile(path, []byte(tc.input), 0600))
				args = append(args, path)
			}
			cmd := exec.CommandContext(ctx, executable, args...)
			cmd.Dir = t.TempDir()
			cmd.Env = append([]string{"FX_TEST_MAIN=1", "FX_THEME=0"}, tc.env...)
			if !tc.file {
				cmd.Stdin = strings.NewReader(tc.input)
			}
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			err := cmd.Run()
			require.NoError(t, ctx.Err(), "CLI did not exit")
			if tc.fail {
				require.Error(t, err)
				require.NotEmpty(t, stderr.String())
			} else {
				require.NoError(t, err, stderr.String())
				require.Empty(t, stderr.String())
			}
			require.Equal(t, tc.want, stdout.String())
		})
	}
}
