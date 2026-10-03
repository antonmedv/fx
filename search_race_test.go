package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"

	. "github.com/antonmedv/fx/internal/jsonx"
)

// startSearch starts a search the way Enter in the search input does, and
// runs it in the background. The result message arrives on the channel.
func startSearch(m *model, query string) <-chan tea.Msg {
	m.searchInput.SetValue(query)
	cmd := m.doSearch(query)
	msgs := make(chan tea.Msg, 1)
	batch, ok := cmd().(tea.BatchMsg)
	if !ok {
		panic("doSearch returns a batch")
	}
	search := batch[1] // The first is the spinner tick.
	go func() { msgs <- search() }()
	return msgs
}

// finishSearch delivers the message of the search to Update.
func finishSearch(t *testing.T, m *model, msgs <-chan tea.Msg) {
	t.Helper()
	select {
	case msg := <-msgs:
		m.Update(msg)
	case <-time.After(20 * time.Second):
		t.Fatal("search did not finish")
	}
}

// requireSearchUsable checks the search is usable after edits: n lands on a
// match in the document, and no wrapped string in the document has
// highlights on chunks re-wrapping dropped. Results deleted since the
// search may remain: n skips them.
func requireSearchUsable(t *testing.T, m *model) {
	t.Helper()
	require.NotEmpty(t, m.search.results)
	for n := range m.search.values {
		if n.IsWrap() && n.Parent.InDocument() {
			require.True(t, n.InDocument(), "highlight on a dropped chunk of %q", n.Parent.Key)
		}
	}
	for range 5 {
		keys(m, "n")
		at, ok := m.cursorPointsTo()
		require.True(t, ok)
		if at.IsWrap() {
			at = at.Parent
		}
		require.True(t, at.InDocument(), "n landed on deleted %q", at.Key+at.Value)
		require.Contains(t, at.Key+at.Value, "needle")
	}
}

func bigInput() string {
	var b strings.Builder
	for i := range 3000 {
		fmt.Fprintf(&b, `{"id": %d, "o": {"text": "%s needle %s"}, "list": [1, 2, "needle"]}`+"\n",
			i, strings.Repeat("word ", 20), strings.Repeat("more ", 20))
	}
	return b.String()
}

// Run with -race: an edit during a search stops it before editing the list,
// which the search reads in the background.
func TestSearch_EditsWhileSearching(t *testing.T) {
	edits := map[string]func(m *model){
		"resize": func(m *model) {
			m.Update(tea.WindowSizeMsg{Width: 30, Height: 10})
			m.Update(tea.WindowSizeMsg{Width: 50, Height: 10})
		},
		"wrap":          func(m *model) { keys(m, "z", "z") },
		"collapse all":  func(m *model) { keys(m, "E", "e") },
		"collapse":      func(m *model) { keys(m, "h", "l") },
		"delete":        func(m *model) { keys(m, "j", "j", "d", "d") },
		"line numbers":  func(m *model) { keys(m, "s", "l") },
		"move and jump": func(m *model) { keys(m, "G", "g", "j", "k") },
		"click":         func(m *model) { m.Update(tea.MouseClickMsg{Button: tea.MouseLeft, Y: 0}) },
	}
	for name, edit := range edits {
		t.Run(name, func(t *testing.T) {
			m := loadModel(t, bigInput(), 40, 10)
			keys(m, "z") // Wrap the long strings.
			msgs := startSearch(m, "needle")
			for range 20 {
				edit(m)
			}
			require.False(t, m.searching, "the edit stopped the search")
			finishSearch(t, m, msgs)
			require.False(t, m.searching)
			require.Empty(t, m.search.results, "the stopped search's result is dropped")
			requireListSound(t, m)

			// Searching again works.
			finishSearch(t, m, startSearch(m, "needle"))
			requireSearchUsable(t, m)
		})
	}
}

// Documents streaming in during a search are attached after the last line
// it reads.
func TestSearch_StreamingWhileSearching(t *testing.T) {
	m := loadModel(t, bigInput(), 40, 10)
	keys(m, "z")
	msgs := startSearch(m, "needle")
	for i := range 200 {
		node, err := Parse([]byte(fmt.Sprintf(`{"late": %d, "s": "needle"}`, i)))
		require.NoError(t, err)
		m.Update(nodeMsg{node: node})
	}
	require.True(t, m.searching, "a new document doesn't stop the search")
	finishSearch(t, m, msgs)
	require.False(t, m.searching)
	requireSearchUsable(t, m)
	requireListSound(t, m)
}

func TestSearch_CancelBeforeStart(t *testing.T) {
	m := loadModel(t, `{"a": "needle"}`, 40, 10)
	cmd := m.doSearch("needle")
	done := make(chan struct{})
	go func() {
		defer close(done)
		m.cancelSearch() // The search command has not run.
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("cancel waited for a search that never started")
	}
	batch := cmd().(tea.BatchMsg)
	msg := batch[1]()
	m.Update(msg)
	require.False(t, m.searching)
	require.Empty(t, m.search.results)
}

func TestSearch_EscDropsFinishedResult(t *testing.T) {
	m := loadModel(t, `{"a": "needle"}`, 40, 10)
	m.Update(press("/"))
	typeKeys(m, "needle")
	m.searchInput.SetValue("needle")
	cmd := m.doSearch("needle")
	msg := cmd().(tea.BatchMsg)[1]() // Finished, not yet delivered.
	m.Update(press("/"))
	m.Update(press("esc"))
	m.Update(msg)
	require.Empty(t, m.search.results, "the search was cancelled before its result arrived")
}

// stop must not return while a started search still reads the list.
func TestSearchRun_StopWaitsForRunningSearch(t *testing.T) {
	run := &searchRun{cancel: make(chan struct{}), done: make(chan struct{})}
	require.True(t, run.begin())
	stopped := make(chan struct{})
	go func() {
		run.stop()
		close(stopped)
	}()
	select {
	case <-run.cancel:
	case <-time.After(5 * time.Second):
		t.Fatal("stop did not cancel")
	}
	select {
	case <-stopped:
		t.Fatal("stop returned while the search still runs")
	case <-time.After(50 * time.Millisecond):
	}
	run.end()
	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("stop did not return once the search ended")
	}
	require.False(t, run.begin(), "a stopped search does not start")
}
