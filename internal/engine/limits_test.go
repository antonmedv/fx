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

// run starts the engine on input with timeout, returning the exit code, the
// outputs and the errors, and how long it took.
func run(t *testing.T, input, arg string, preview bool, timeout time.Duration) (int, []*jsonx.Node, []string, time.Duration) {
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
	code := start(jsonx.NewJsonParser(strings.NewReader(input), false), []string{arg}, out, errCh, make(chan struct{}), preview, timeout)
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
