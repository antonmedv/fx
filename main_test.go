package main

import (
	"bytes"
	"io"
	"os"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/exp/teatest/v2"
	"github.com/stretchr/testify/require"

	"github.com/antonmedv/fx/internal/jsonx"
)

type options struct {
	showSizes       bool
	showLineNumbers bool
}

func prepare(t *testing.T, opts ...options) *teatest.TestModel {
	file, err := os.Open("testdata/example.json")
	require.NoError(t, err)

	json, err := io.ReadAll(file)
	require.NoError(t, err)

	head, err := jsonx.Parse(json)
	require.NoError(t, err)

	m := &model{
		viewState: viewState{
			top:        head,
			head:       head,
			bottom:     head,
			totalLines: head.Bottom().LineNumber,
			search:     newSearch(),
		},
		eof:          true,
		wrap:         true,
		showCursor:   true,
		searchInput:  newInput(),
		commandInput: newInput(),
	}

	if len(opts) > 0 {
		m.showSizes = opts[0].showSizes
		m.showLineNumbers = opts[0].showLineNumbers
	}

	tm := teatest.NewTestModel(
		t, m,
		teatest.WithInitialTermSize(80, 40),
		teatest.WithProgramOptions(tea.WithColorProfile(colorprofile.ANSI)),
	)
	return tm
}

func read(t *testing.T, tm *teatest.TestModel) []byte {
	var out []byte
	teatest.WaitFor(t,
		tm.Output(),
		func(b []byte) bool {
			out = b
			return bytes.Contains(b, []byte("{"))
		},
		teatest.WithCheckInterval(time.Millisecond*100),
		teatest.WithDuration(time.Second),
	)
	return out
}

func TestOutput(t *testing.T) {
	tm := prepare(t)

	teatest.RequireEqualOutput(t, read(t, tm))

	tm.Send(press("q"))
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestCursorValueDecodesJsonEscapesInWrappedArrayString verifies that preview
// and print decode JSON string escapes even after the viewer wraps an array item.
func TestCursorValueDecodesJsonEscapesInWrappedArrayString(t *testing.T) {
	head, err := jsonx.Parse([]byte(`["before\ud83d\udd55\nsecond line after"]`))
	require.NoError(t, err)

	jsonx.Wrap(head, 12)
	require.NotNil(t, head.Next)
	require.NotNil(t, head.Next.ChunkEnd)

	m := &model{
		viewState: viewState{
			head:   head,
			cursor: 1,
		},
	}
	require.Equal(t, "before\U0001F555\nsecond line after", m.cursorValue())

	m.cursor = 2
	require.Equal(t, "before\U0001F555\nsecond line after", m.cursorValue())
}

func TestNavigation(t *testing.T) {
	tm := prepare(t)

	tm.Send(press("down"))
	tm.Send(press("down"))
	tm.Send(press("down"))
	teatest.RequireEqualOutput(t, read(t, tm))

	tm.Send(press("q"))
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

func TestCollapseRecursive(t *testing.T) {
	tm := prepare(t)

	tm.Send(press("shift+left"))
	teatest.RequireEqualOutput(t, read(t, tm))

	tm.Send(press("q"))
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

func TestCollapseRecursiveWithSizes(t *testing.T) {
	tm := prepare(t, options{showSizes: true})

	tm.Send(press("shift+left"))
	teatest.RequireEqualOutput(t, read(t, tm))

	tm.Send(press("q"))
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}
