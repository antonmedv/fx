package engine_test

import (
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/antonmedv/fx/internal/engine"
	"github.com/antonmedv/fx/internal/jsonx"
	"github.com/antonmedv/fx/internal/pretty"
)

// runEngine runs the engine with the given parser and args, collecting outputs and errors.
// It returns the exit code, collected outputs, and collected errors.
func runEngine(parser engine.Parser, args []string) (exitCode int, outs []string, errs []string) {
	out := make(chan *jsonx.Node)
	errCh := make(chan error)
	cancel := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for node := range out {
			if node.Kind == jsonx.String {
				outs = append(outs, node.Value)
			} else {
				outs = append(outs, pretty.Print(node, false))
			}
		}
	}()

	go func() {
		defer wg.Done()
		for err := range errCh {
			errs = append(errs, err.Error())
		}
	}()

	exitCode = engine.Start(parser, args, out, errCh, cancel)
	close(out)
	close(errCh)
	wg.Wait()

	return exitCode, outs, errs
}

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
			expects:  []string{"\"Hello, world!\""},
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

			exitCode, outs, errs := runEngine(parser, tc.args)

			assert.Equal(t, 0, exitCode)
			assert.Len(t, errs, tc.errCount, "%s: unexpected error count", tc.name)
			assert.Equal(t, tc.expects, outs, "%s: outputs mismatch", tc.name)
		})
	}
}

func TestStart_InvalidJSON(t *testing.T) {
	input := `{"unclosed": 1`
	parser := jsonx.NewJsonParser(strings.NewReader(input), false)

	exitCode, _, errs := runEngine(parser, []string{".unclosed + '!'"})

	assert.Equal(t, 1, exitCode)
	assert.Len(t, errs, 1, "Expected one error message")
}

func TestStart_FastPath_InvalidJSON(t *testing.T) {
	input := `{"unclosed": 1`
	parser := jsonx.NewJsonParser(strings.NewReader(input), false)

	exitCode, _, errs := runEngine(parser, []string{"."})

	assert.Equal(t, 1, exitCode)
	assert.Len(t, errs, 1, "Expected one error message")
}

func TestStart_EscapeSequences(t *testing.T) {
	input := `{"emoji": "\ud83d\ude80"}`
	parser := jsonx.NewJsonParser(strings.NewReader(input), false)

	exitCode, outs, errs := runEngine(parser, []string{".emoji"})

	assert.Equal(t, 0, exitCode)
	assert.Len(t, errs, 0, "Expected no error messages")
	assert.Equal(t, "🚀", outs[0])
}

func TestStart_EscapeSequences_in_key(t *testing.T) {
	input := `{"\ud83d\ude80": "\ud83d\ude80"}`
	parser := jsonx.NewJsonParser(strings.NewReader(input), false)

	exitCode, _, errs := runEngine(parser, []string{"x => x"})

	assert.Equal(t, 0, exitCode)
	assert.Len(t, errs, 0, "Expected no error messages")
}

func TestStart_Cancel(t *testing.T) {
	// Create a parser that would produce multiple values
	input := "1 2 3 4 5"
	parser := jsonx.NewJsonParser(strings.NewReader(input), false)

	out := make(chan *jsonx.Node, 10)
	errCh := make(chan error, 10)
	cancel := make(chan struct{})

	// Close cancel immediately to test cancellation
	close(cancel)

	exitCode := engine.Start(parser, []string{"."}, out, errCh, cancel)
	close(out)
	close(errCh)

	// Should return 0 on cancellation
	assert.Equal(t, 0, exitCode)
}
