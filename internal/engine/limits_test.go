package engine

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/antonmedv/fx/internal/jsonx"
)

// startFunc is the signature of Start and StartPreview.
type startFunc func(Parser, []string, chan *jsonx.Node, chan error, <-chan struct{}) int

// run starts the engine on input with timeout, returning the exit code, the
// outputs and the errors, and how long it took.
func run(t *testing.T, input, arg string, preview bool, timeout time.Duration) (int, []*jsonx.Node, []string, time.Duration) {
	t.Helper()
	return runWith(t, input, arg, func(p Parser, args []string, out chan *jsonx.Node, errCh chan error, cancel <-chan struct{}) int {
		return start(p, args, out, errCh, cancel, preview, timeout)
	})
}

// runWith is run through an entry point, as Start or StartPreview.
func runWith(t *testing.T, input, arg string, startFn startFunc) (int, []*jsonx.Node, []string, time.Duration) {
	t.Helper()
	out := make(chan *jsonx.Node)
	errCh := make(chan error)
	var outs []*jsonx.Node
	var errs []string
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for n := range out {
			outs = append(outs, n)
		}
	}()
	go func() {
		defer wg.Done()
		for e := range errCh {
			errs = append(errs, e.Error())
		}
	}()
	begin := time.Now()
	code := startFn(jsonx.NewJsonParser(strings.NewReader(input), false), []string{arg}, out, errCh, make(chan struct{}))
	took := time.Since(begin)
	close(out)
	close(errCh)
	wg.Wait()
	return code, outs, errs, took
}

func TestTimeout_StopsNeverEndingQuery(t *testing.T) {
	code, _, errs, took := run(t, `1`, `x => { while (true) {} }`, true, 200*time.Millisecond)
	assert.Equal(t, 1, code)
	require.Len(t, errs, 1)
	assert.Equal(t, "Query stopped: it took longer than 200ms", errs[0])
	assert.Less(t, took, 5*time.Second)
}

func TestTimeout_StopsAllocatingQuery(t *testing.T) {
	code, _, errs, took := run(t, `[1,2,3]`, `chunk(0)`, true, 200*time.Millisecond)
	assert.Equal(t, 1, code)
	require.Len(t, errs, 1)
	assert.Equal(t, "Query stopped: it took longer than 200ms", errs[0])
	assert.Less(t, took, 5*time.Second)
}

func TestTimeout_OnlyPreviews(t *testing.T) {
	assert.Equal(t, 2*time.Second, previewTimeout)

	// A query busy for longer than the preview timeout.
	saved := previewTimeout
	previewTimeout = 50 * time.Millisecond
	defer func() { previewTimeout = saved }()
	slow := `x => { const end = Date.now() + 300; while (Date.now() < end) {} return 1 }`

	code, _, errs, _ := runWith(t, `1`, slow, StartPreview)
	assert.Equal(t, 1, code, "a preview is stopped")
	assert.Equal(t, []string{"Query stopped: it took longer than 50ms"}, errs)

	code, outs, errs, took := runWith(t, `1`, slow, Start)
	assert.Equal(t, 0, code, "a query the user runs is not")
	assert.Empty(t, errs)
	require.Len(t, outs, 1)
	assert.Equal(t, "1", outs[0].Value)
	assert.GreaterOrEqual(t, took, 300*time.Millisecond)
}

func TestTimeout_PreviewAllowsFastQueries(t *testing.T) {
	code, outs, errs, _ := run(t, `[1,2,3]`, `x => x.map(y => y * 2)`, true, previewTimeout)
	assert.Equal(t, 0, code)
	assert.Empty(t, errs)
	require.Len(t, outs, 1)
}

func TestStringify_DeepOutputIsLinear(t *testing.T) {
	code, outs, errs, took := run(t, `1`, `x => { let a = []; for (let i = 0; i < 9000; i++) a = [a]; return a }`, false, 0)
	assert.Equal(t, 0, code)
	assert.Empty(t, errs)
	require.Len(t, outs, 1)
	assert.Less(t, took, 5*time.Second)
}

func TestStringify_AbortStops(t *testing.T) {
	vm := goja.New()
	value, err := vm.RunString(`Array.from({length: 100000}, (_, i) => ({i}))`)
	require.NoError(t, err)
	abort := make(chan struct{})
	close(abort)
	assert.PanicsWithValue(t, errAborted, func() { stringifyCompact(value, vm, abort) })
}

func TestStringify_Formats(t *testing.T) {
	vm := goja.New()
	value, err := vm.RunString(`({a: [1, {b: null}], c: {}, d: [], e: "s"})`)
	require.NoError(t, err)
	assert.Equal(t, `{"a":[1,{"b":null}],"c":{},"d":[],"e":"s"}`, stringifyCompact(value, vm, nil))
	assert.Equal(t, "{\n  \"a\": [\n    1,\n    {\n      \"b\": null\n    }\n  ],\n  \"c\": {},\n  \"d\": [],\n  \"e\": \"s\"\n}", Stringify(value, vm, 0))
}
