package main

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"

	"github.com/antonmedv/fx/internal/engine"
)

// newReloadModel creates a model showing file, loaded as main does.
func newReloadModel(t *testing.T, content string) (*model, string) {
	file := filepath.Join(t.TempDir(), "f.json")
	require.NoError(t, os.WriteFile(file, []byte(content), 0o644))
	oldPath, oldInput := engine.FilePath, engine.Input
	engine.FilePath = file
	t.Cleanup(func() {
		if engine.Input != nil {
			_ = engine.Input.Close()
		}
		engine.FilePath, engine.Input = oldPath, oldInput
	})
	m := newQueryModel(t)
	m.eof = false
	runAll(m, m.reload())
	return m, file
}

// runAll runs cmd, batches included, feeding messages back into Update
// until there are no more commands.
func runAll(m *model, cmd tea.Cmd) {
	queue := []tea.Cmd{cmd}
	for len(queue) > 0 {
		cmd, queue = queue[0], queue[1:]
		if cmd == nil {
			continue
		}
		msg := cmd()
		if batch, ok := msg.(tea.BatchMsg); ok {
			queue = append(batch, queue...)
			continue
		}
		_, next := m.Update(msg)
		queue = append([]tea.Cmd{next}, queue...)
	}
}

func reloadKey(m *model) {
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	runAll(m, cmd)
}

func TestReload_ShowsNewContent(t *testing.T) {
	m, file := newReloadModel(t, `{"a": 1}`)
	require.Equal(t, []string{"{", `"a"1`, "}"}, lines(m))
	require.True(t, m.eof)

	require.NoError(t, os.WriteFile(file, []byte(`{"b": 2}`+"\n"+`[3]`), 0o644))
	reloadKey(m)
	require.Equal(t, []string{"{", `"b"2`, "}", "[", "3", "]"}, lines(m))
	require.True(t, m.eof)
	require.Equal(t, 6, m.totalLines)
}

func TestReload_KeepsCursorPath(t *testing.T) {
	m, file := newReloadModel(t, `{"a": 1, "b": {"c": 2}}`)
	m.selectNode(m.findByPath([]any{"b", "c"}))

	require.NoError(t, os.WriteFile(file, []byte(`{"x": 0, "y": 0, "a": 1, "b": {"c": 3}}`), 0o644))
	reloadKey(m)
	at, ok := m.cursorPointsTo()
	require.True(t, ok)
	require.Equal(t, `"c"`, at.Key)
	require.Equal(t, "3", at.Value)
	require.Equal(t, ".b.c", m.cursorPath())
}

func TestReload_KeepsCursorDocument(t *testing.T) {
	m, file := newReloadModel(t, `{"id": 1}`+"\n"+`{"id": 2}`)
	m.selectNode(m.top.End.Next.Next) // .id of the second document

	require.NoError(t, os.WriteFile(file, []byte(`{"id": 10}`+"\n"+`{"id": 20}`), 0o644))
	reloadKey(m)
	at, ok := m.cursorPointsTo()
	require.True(t, ok)
	require.Equal(t, "20", at.Value)
}

func TestReload_MissingPathSelectsDocument(t *testing.T) {
	m, file := newReloadModel(t, `{"a": {"b": 1}}`)
	m.selectNode(m.findByPath([]any{"a", "b"}))

	require.NoError(t, os.WriteFile(file, []byte(`{"z": 1}`), 0o644))
	reloadKey(m)
	at, ok := m.cursorPointsTo()
	require.True(t, ok)
	require.Same(t, m.top, at)
}

func TestReload_KeyUnquoteFails(t *testing.T) {
	m, file := newReloadModel(t, `{"a\/b": 1, "c": 2}`)
	m.selectNode(m.findByPath([]any{"c"}))

	require.NoError(t, os.WriteFile(file, []byte(`{"a\/b": 1, "c": 3}`), 0o644))
	reloadKey(m)
	at, ok := m.cursorPointsTo()
	require.True(t, ok)
	require.Equal(t, "3", at.Value)
}

func TestReload_RunsQueryAgain(t *testing.T) {
	m, file := newReloadModel(t, `{"a": 1}`)
	runAll(m, m.doQuery(".a"))
	require.Equal(t, []string{"1"}, lines(m))

	require.NoError(t, os.WriteFile(file, []byte(`{"a": 2}`), 0o644))
	reloadKey(m)
	require.Equal(t, []string{"2"}, lines(m))
	require.NotNil(t, m.original)
	require.True(t, m.query.done)

	runAll(m, m.clearQuery())
	require.Equal(t, []string{"{", `"a"2`, "}"}, lines(m))
}

func TestReload_ErrorKeepsRunning(t *testing.T) {
	m, file := newReloadModel(t, `{"a": 1}`)

	require.NoError(t, os.WriteFile(file, []byte(`{"a": `), 0o644))
	reloadKey(m)
	require.Nil(t, m.printErrorOnExit)
	require.True(t, m.eof)
	require.NotEmpty(t, lines(m))
	require.True(t, m.isQueryError(m.top))

	require.NoError(t, os.WriteFile(file, []byte(`{"a": 2}`), 0o644))
	reloadKey(m)
	require.Equal(t, []string{"{", `"a"2`, "}"}, lines(m))
	require.False(t, m.isQueryError(m.top))
}

func TestReload_MissingFile(t *testing.T) {
	m, file := newReloadModel(t, `{"a": 1}`)

	require.NoError(t, os.Remove(file))
	reloadKey(m)
	require.Nil(t, m.printErrorOnExit)
	require.True(t, m.eof)
	require.Len(t, lines(m), 1)
	require.True(t, m.isQueryError(m.top))
}

func TestReload_DropsStaleMessages(t *testing.T) {
	m, _ := newReloadModel(t, `{"a": 1}`)
	stale := m.loadGen - 1

	m.Update(nodeMsg{node: parseDoc(t, `[1]`), gen: stale})
	m.Update(errorMsg{err: os.ErrClosed, gen: stale})
	require.Equal(t, []string{"{", `"a"1`, "}"}, lines(m))
	require.Nil(t, m.printErrorOnExit)
}

func TestReload_StdinIsNoop(t *testing.T) {
	oldPath := engine.FilePath
	engine.FilePath = ""
	t.Cleanup(func() { engine.FilePath = oldPath })

	m := newQueryModel(t, `{"a": 1}`)
	reloadKey(m)
	require.Equal(t, []string{"{", `"a"1`, "}"}, lines(m))
	require.Zero(t, m.loadGen)
}
