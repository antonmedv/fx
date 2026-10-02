package engine_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/antonmedv/fx/internal/engine"
	"github.com/antonmedv/fx/internal/jsonx"
)

// The engine runs in its own goroutine next to the UI: none of these may
// panic, each is a value or an error.

func TestStart_InvalidStringIsError(t *testing.T) {
	tests := []struct {
		name, input, arg, want string
	}{
		{"unknown escape", `{"a":"\g"}`, "x.a",
			`Invalid JSON string "\g" at .a`},
		{"unknown escape in key", `{"\g":1}`, "x => x",
			`Invalid JSON key "\g"`},
		{"go hex escape", `{"a":{"b c":["ok","\x1b"]}}`, "x => x",
			`Invalid JSON string "\x1b" at .a["b c"][1]`},
		{"raw tab", "\"a\tb\"", "x.length",
			`Invalid JSON string "a\tb"`},
		{"top-level", `"\g"`, "x => x",
			`Invalid JSON string "\g"`},
		{"control char, shortened", "\"\x01" + strings.Repeat("a", 50) + "\"", "x => x",
			`Invalid JSON string "\u0001` + strings.Repeat("a", 38) + "…"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exitCode, outs, errs := runEngine(jsonx.NewJsonParser(strings.NewReader(tt.input), false), []string{tt.arg})
			assert.Equal(t, 1, exitCode)
			assert.Empty(t, outs)
			require.Len(t, errs, 1)
			assert.Equal(t, tt.want, errs[0])
		})
	}
}

func TestStart_InvalidStringInLaterDocument(t *testing.T) {
	exitCode, outs, errs := runEngine(jsonx.NewJsonParser(strings.NewReader("{\"a\":1}\n{\"a\":\"\\g\"}\n{\"a\":3}"), false),
		[]string{"x.a"})
	assert.Equal(t, 1, exitCode)
	assert.Equal(t, []string{"1"}, outs)
	require.Len(t, errs, 1)
	assert.Equal(t, `Invalid JSON string "\g" at .a`, errs[0])
}

func TestStart_OutOfRangeNumberIsInfinity(t *testing.T) {
	exitCode, outs, errs := runEngine(jsonx.NewJsonParser(strings.NewReader(`[1e400, -1e400, 1.5]`), false),
		[]string{"x.map(String)"})
	assert.Equal(t, 0, exitCode)
	assert.Empty(t, errs)
	require.Len(t, outs, 1)
	assert.Contains(t, outs[0], `"Infinity"`)
	assert.Contains(t, outs[0], `"-Infinity"`)
	assert.Contains(t, outs[0], `"1.5"`)
}

func TestStart_CircularStructureIsError(t *testing.T) {
	for _, arg := range []string{
		"x => (x.self = x, x)",
		"x => { const a = [x]; x.a = a; return [a] }",
		"x => (x.self = x, YAML.stringify(x))",
	} {
		t.Run(arg, func(t *testing.T) {
			exitCode, outs, errs := runEngine(jsonx.NewJsonParser(strings.NewReader(`{}`), false), []string{arg})
			assert.Equal(t, 1, exitCode)
			assert.Empty(t, outs)
			require.Len(t, errs, 1)
			assert.Contains(t, errs[0], "Converting circular structure to JSON")
		})
	}
}

func TestStart_SharedObjectIsNotCircular(t *testing.T) {
	exitCode, outs, errs := runEngine(jsonx.NewJsonParser(strings.NewReader(`{"a":1}`), false),
		[]string{"x => [x, x, {b: x}]"})
	assert.Equal(t, 0, exitCode)
	assert.Empty(t, errs)
	require.Len(t, outs, 1)
}

func TestStart_InfiniteRecursionIsRangeError(t *testing.T) {
	for _, arg := range []string{
		"x => { const f = n => f(n + 1); return f(0) }",
		// In a getter run while serializing the output.
		"x => ({ get a() { const f = n => f(n + 1); return f(0) } })",
	} {
		t.Run(arg, func(t *testing.T) {
			exitCode, outs, errs := runEngine(jsonx.NewJsonParser(strings.NewReader(`1`), false), []string{arg})
			assert.Equal(t, 1, exitCode)
			assert.Empty(t, outs)
			require.Len(t, errs, 1)
			assert.Equal(t, "RangeError: Maximum call stack size exceeded", errs[0])
		})
	}
}

// panicParser is an input whose parser panics, as a bug in fx would.
type panicParser struct{}

func (panicParser) Parse() (*jsonx.Node, error) { panic("boom") }
func (panicParser) Recover() *jsonx.Node        { return nil }
func (panicParser) More() (bool, error)         { return false, nil }

func TestStart_PanicIsError(t *testing.T) {
	for _, args := range [][]string{{"."}, {"x => x"}} {
		exitCode, outs, errs := runEngine(panicParser{}, args)
		assert.Equal(t, 1, exitCode)
		assert.Empty(t, outs)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0], "internal error: boom")
	}
}

func TestStartPreview_PanicIsError(t *testing.T) {
	out := make(chan *jsonx.Node)
	errCh := make(chan error, 1)
	exitCode := engine.StartPreview(panicParser{}, []string{"x => x"}, out, errCh, make(chan struct{}))
	assert.Equal(t, 1, exitCode)
	assert.Contains(t, (<-errCh).Error(), "internal error: boom")
}
