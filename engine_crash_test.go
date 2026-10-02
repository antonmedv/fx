package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// The engine runs next to the UI: no input and no expression may crash fx.

func TestQuery_InvalidStringIsError(t *testing.T) {
	m := newQueryModel(t, `{"a": "\g"}`)
	drain(m, m.doQuery(`.a`))
	require.True(t, m.isQueryError(m.top), "%v", lines(m))
	require.Equal(t, `Invalid JSON string "\g" at .a`, m.top.Value)
}

func TestQuery_OutOfRangeNumberIsInfinity(t *testing.T) {
	m := newQueryModel(t, `{"n": 1e400}`)
	drain(m, m.doQuery(`x => String(x.n)`))
	require.Equal(t, []string{`"Infinity"`}, lines(m))
}

func TestPreview_InvalidStringKeepsView(t *testing.T) {
	m := newQueryModel(t, `{"a": "\g"}`)
	original := m.top
	preview(m, ".a")
	require.Same(t, original, m.top, "%v", lines(m))
}

func TestQuery_RuntimeFailuresAreErrors(t *testing.T) {
	for _, q := range []string{
		"x => (x.self = x, x)",
		"x => { const f = n => f(n + 1); return f(0) }",
	} {
		t.Run(q, func(t *testing.T) {
			m := newQueryModel(t, `{"a": 1}`)
			drain(m, m.doQuery(q))
			require.NotNil(t, m.top)
			require.True(t, m.isQueryError(m.top), "%v", lines(m))
		})
	}
}

func TestPreview_RuntimeFailuresKeepView(t *testing.T) {
	m := newQueryModel(t, `{"a": 1}`)
	preview(m, ".a")
	preview(m, "x => (x.self = x, x)")
	require.Equal(t, []string{"1"}, lines(m))
}
