package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormatErrorsGoToStderr(t *testing.T) {
	for _, tt := range []struct{ name, flag, input string }{
		{"yaml", "--yaml", "a: [\n"},
		{"toml", "--toml", "x = [\n"},
		{"xml", "--xml", "<root>\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^$")
			cmd.Env = append(os.Environ(), "FX_TEST_RUN_MAIN=1", "FX_TEST_ARGS="+tt.flag+" .")
			cmd.Stdin = strings.NewReader(tt.input)
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			err := cmd.Run()
			var exitErr *exec.ExitError
			require.ErrorAs(t, err, &exitErr)
			require.Equal(t, 1, exitErr.ExitCode())
			require.Empty(t, stdout.String())
			require.NotEmpty(t, stderr.String())
		})
	}
}
