package engine_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/antonmedv/fx/internal/engine"
	"github.com/antonmedv/fx/internal/jsonx"
)

func TestEngine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		args     []string
		expects  []string
		errCount int
	}{
		{
			name:     "fast path: string as raw",
			input:    `"Hello, world!"`,
			args:     []string{"."},
			expects:  []string{"Hello, world!"},
			errCount: 0,
		},
		{
			name:     "string as raw",
			input:    `"Hello, world!"`,
			args:     []string{"x => this"},
			expects:  []string{"Hello, world!"},
			errCount: 0,
		},
		{
			name:     "skip works",
			input:    "1 2 3 4",
			args:     []string{"x % 2 != 0 ? skip : x"},
			expects:  []string{"2", "4"},
			errCount: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			parser := jsonx.NewJsonParser(strings.NewReader(tc.input), false)

			var outs, errs []string
			writeOut := func(s string) { outs = append(outs, s) }
			writeErr := func(s string) { errs = append(errs, s) }

			exitCode := engine.Start(parser, tc.args, false, false, writeOut, writeErr)

			assert.Equal(t, 0, exitCode)
			assert.Len(t, errs, tc.errCount, "%s: unexpected error count", tc.name)
			assert.Equal(t, tc.expects, outs, "%s: outputs mismatch", tc.name)
		})
	}
}

func TestStart_InvalidJSON(t *testing.T) {
	input := `{"unclosed": 1`
	parser := jsonx.NewJsonParser(strings.NewReader(input), false)

	var outs, errs []string
	writeOut := func(s string) { outs = append(outs, s) }
	writeErr := func(s string) { errs = append(errs, s) }

	exitCode := engine.Start(parser, []string{".unclosed + '!'"}, false, false, writeOut, writeErr)

	assert.Equal(t, 1, exitCode)
	assert.Len(t, errs, 1, "Expected one error message")
}

func TestStart_FastPath_InvalidJSON(t *testing.T) {
	input := `{"unclosed": 1`
	parser := jsonx.NewJsonParser(strings.NewReader(input), false)

	var outs, errs []string
	writeOut := func(s string) { outs = append(outs, s) }
	writeErr := func(s string) { errs = append(errs, s) }

	exitCode := engine.Start(parser, []string{"."}, false, false, writeOut, writeErr)

	assert.Equal(t, 1, exitCode)
	assert.Len(t, errs, 1, "Expected one error message")
}

func TestStart_EscapeSequences(t *testing.T) {
	input := `{"emoji": "\ud83d\ude80"}`
	parser := jsonx.NewJsonParser(strings.NewReader(input), false)

	var outs, errs []string
	writeOut := func(s string) { outs = append(outs, s) }
	writeErr := func(s string) { errs = append(errs, s) }

	exitCode := engine.Start(parser, []string{".emoji"}, false, false, writeOut, writeErr)

	assert.Equal(t, 0, exitCode)
	assert.Len(t, errs, 0, "Expected no error messages")
	assert.Equal(t, "🚀", outs[0])
}

func TestStart_EscapeSequences_in_key(t *testing.T) {
	input := `{"\ud83d\ude80": "\ud83d\ude80"}`
	parser := jsonx.NewJsonParser(strings.NewReader(input), false)

	var outs, errs []string
	writeOut := func(s string) { outs = append(outs, s) }
	writeErr := func(s string) { errs = append(errs, s) }

	exitCode := engine.Start(parser, []string{"x => x"}, false, false, writeOut, writeErr)

	assert.Equal(t, 0, exitCode)
	assert.Len(t, errs, 0, "Expected no error messages")
}
