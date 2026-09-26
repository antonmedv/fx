package engine_test

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/antonmedv/fx/internal/engine"
	"github.com/antonmedv/fx/internal/jsonx"
	"github.com/antonmedv/fx/internal/pretty"
	"github.com/antonmedv/fx/internal/utils"
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
				unquoted, err := utils.Unquote(node.Value)
				if err != nil {
					panic(err)
				}
				outs = append(outs, unquoted)
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

func TestStart_StringNodeIsQuoted(t *testing.T) {
	parser := jsonx.NewJsonParser(strings.NewReader(`{"a": "x\"y"}`), false)

	out := make(chan *jsonx.Node, 10)
	errCh := make(chan error, 10)
	exitCode := engine.Start(parser, []string{".a"}, out, errCh, make(chan struct{}))
	close(out)
	close(errCh)

	assert.Equal(t, 0, exitCode)
	node := <-out
	assert.Equal(t, jsonx.String, node.Kind)
	assert.Equal(t, `"x\"y"`, node.Value)
}

func TestStart_CancelWhileBlockedOnSend(t *testing.T) {
	for _, args := range [][]string{{"."}, {"x + 1"}, {"x => undefined"}} {
		t.Run(args[0], func(t *testing.T) {
			parser := jsonx.NewJsonParser(strings.NewReader("1 2 3 4 5"), false)

			// Unbuffered and never read: Start blocks on the first send.
			out := make(chan *jsonx.Node)
			errCh := make(chan error)
			cancel := make(chan struct{})

			done := make(chan int)
			go func() {
				done <- engine.Start(parser, args, out, errCh, cancel)
			}()

			close(cancel)

			select {
			case exitCode := <-done:
				assert.Equal(t, 0, exitCode)
			case <-time.After(2 * time.Second):
				t.Fatal("Start did not return after cancel")
			}
		})
	}
}

func TestStart_StringNodeHasLineNumber(t *testing.T) {
	parser := jsonx.NewJsonParser(strings.NewReader(`{"a": "x"}`), false)
	out := make(chan *jsonx.Node, 10)
	errCh := make(chan error, 10)
	engine.Start(parser, []string{".a"}, out, errCh, make(chan struct{}))
	close(out)
	require.Equal(t, 1, (<-out).LineNumber)
}

func TestStart_CancelInterruptsRunningJS(t *testing.T) {
	for _, q := range []string{
		`x => { while (true) {} }`,
		`x => ({get a() { while (true) {} }})`, // Runs during serialization.
	} {
		t.Run(q, func(t *testing.T) {
			parser := jsonx.NewJsonParser(strings.NewReader("1"), false)
			out := make(chan *jsonx.Node)
			errCh := make(chan error)
			cancel := make(chan struct{})

			done := make(chan int)
			go func() {
				done <- engine.Start(parser, []string{q}, out, errCh, cancel)
			}()

			time.Sleep(50 * time.Millisecond)
			close(cancel)

			select {
			case exitCode := <-done:
				assert.Equal(t, 0, exitCode)
			case <-time.After(2 * time.Second):
				t.Fatal("Start did not return after cancel")
			}
		})
	}
}

func TestStart_GetterRunsDuringSerialization(t *testing.T) {
	exitCode, outs, errs := runEngine(jsonx.NewJsonParser(strings.NewReader("1"), false),
		[]string{`x => ({get a() { throw new Error("boom") }})`})
	assert.Equal(t, 1, exitCode)
	assert.Empty(t, outs)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0], "boom")

	exitCode, outs, errs = runEngine(jsonx.NewJsonParser(strings.NewReader("1"), false),
		[]string{`x => ({get a() { exit(3) }})`})
	assert.Equal(t, 3, exitCode)
	assert.Empty(t, outs)
	assert.Empty(t, errs)
}

func TestStart_ExitCodes(t *testing.T) {
	for q, want := range map[string]int{
		`x => exit()`:                        0,
		`x => exit(0)`:                       0,
		`x => exit(2)`:                       2,
		`x => exit(-1)`:                      -1,
		`x => ({get a() { exit(-1) }})`:      -1,
		`x => x == 1 ? x : exit(-1)`:         -1,
		`x => (println("a"), exit(3), x)`:    3,
		`x => ({get a() { return exit() }})`: 0,
	} {
		t.Run(q, func(t *testing.T) {
			parser := jsonx.NewJsonParser(strings.NewReader("1 2"), false)
			exitCode, _, errs := runEngine(parser, []string{q})
			assert.Equal(t, want, exitCode)
			assert.Empty(t, errs)
		})
	}
}

func TestStartPreview_ExitDisabled(t *testing.T) {
	for _, q := range []string{
		`x => exit()`,
		`x => exit(0)`,
		`x => exit(-1)`,
		`x => exit(2)`,
		`x => (__exit__(0), x)`,
		`x => x == 1 ? x : exit(0)`, // After partial output.
		`x => ({get a() { exit(0) }})`,
	} {
		t.Run(q, func(t *testing.T) {
			out := make(chan *jsonx.Node, 10)
			errCh := make(chan error, 10)
			parser := jsonx.NewJsonParser(strings.NewReader("1 2"), false)
			exitCode := engine.StartPreview(parser, []string{q}, out, errCh, make(chan struct{}))
			close(errCh)

			assert.Equal(t, 1, exitCode)
			err := <-errCh
			require.Error(t, err)
			assert.Contains(t, err.Error(), "exit is disabled in preview")
		})
	}
}

func TestStartPreview_SaveDisabled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"a": 1}`), 0644))
	old := engine.FilePath
	engine.FilePath = path
	t.Cleanup(func() { engine.FilePath = old })

	for _, q := range []string{
		`x => save(x)`,
		`x => (__save__("{}"), x)`,
		`x => globalThis["sa" + "ve"](x)`,
	} {
		out := make(chan *jsonx.Node, 10)
		errCh := make(chan error, 10)
		parser := jsonx.NewJsonParser(strings.NewReader(`{"a": 2}`), false)
		exitCode := engine.StartPreview(parser, []string{q}, out, errCh, make(chan struct{}))
		close(errCh)

		assert.Equal(t, 1, exitCode, q)
		err := <-errCh
		require.Error(t, err, q)
		assert.Contains(t, err.Error(), "save is disabled in preview", q)
		data, _ := os.ReadFile(path)
		assert.Equal(t, `{"a": 1}`, string(data), q)
	}

	// Start still saves.
	parser := jsonx.NewJsonParser(strings.NewReader(`{"a": 2}`), false)
	exitCode := engine.Start(parser, []string{`x => (save(x), skip)`}, make(chan *jsonx.Node, 10), make(chan error, 10), make(chan struct{}))
	assert.Equal(t, 0, exitCode)
	data, _ := os.ReadFile(path)
	assert.Contains(t, string(data), `"a": 2`)
}
