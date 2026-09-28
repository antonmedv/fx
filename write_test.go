package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"
	"github.com/stretchr/testify/require"

	"github.com/antonmedv/fx/internal/engine"
	"github.com/antonmedv/fx/internal/ident"
	"github.com/antonmedv/fx/internal/jsonx"
)

// withInputFile makes path the file fx was opened with.
func withInputFile(t *testing.T, path string) {
	t.Helper()
	old := engine.FilePath
	engine.FilePath = path
	t.Cleanup(func() { engine.FilePath = old })
}

// typeCommand types :line and presses enter.
func typeCommand(m *model, line string) {
	typeKeys(m, ":"+line)
	enter(m)
}

func answer(m *model, key string) {
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
	drain(m, cmd)
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(data)
}

func TestWrite_ToNewFile(t *testing.T) {
	m := newQueryModel(t, `{"a": [1, 2], "b": "x", "c": {}, "d": []}`)
	out := filepath.Join(t.TempDir(), "out.json")

	typeCommand(m, "w "+out)

	want := "{\n  \"a\": [\n    1,\n    2\n  ],\n  \"b\": \"x\",\n  \"c\": {},\n  \"d\": []\n}\n"
	require.Equal(t, want, readFile(t, out))
	require.Nil(t, m.confirm)
	require.NotNil(t, m.message)
	require.False(t, m.message.isErr)
	require.Equal(t, fmt.Sprintf(`"%s" 9L, %dB written`, out, len(want)), m.message.text)
}

func TestWrite_FullNameAndScalar(t *testing.T) {
	m := newQueryModel(t, `"hello"`)
	out := filepath.Join(t.TempDir(), "out.json")

	typeCommand(m, "write "+out)

	require.Equal(t, "\"hello\"\n", readFile(t, out))
}

// Several documents are written as JSON Lines, so the format of a JSON
// Lines input is kept.
func TestWrite_SeveralDocuments(t *testing.T) {
	m := newQueryModel(t, `{"a": 1, "b": [2, 3]}`, `[1]`, `2`)
	out := filepath.Join(t.TempDir(), "out.json")

	typeCommand(m, "w "+out)

	require.Equal(t, "{\"a\":1,\"b\":[2,3]}\n[1]\n2\n", readFile(t, out))
	require.Equal(t, fmt.Sprintf(`"%s" 3L, 24B written`, out), m.message.text)
}

func TestWrite_NoFileName(t *testing.T) {
	withInputFile(t, "")
	m := newQueryModel(t, `{"a": 1}`)

	typeCommand(m, "w")

	require.NotNil(t, m.message)
	require.True(t, m.message.isErr)
	require.Equal(t, "No file name", m.message.text)
}

func TestWrite_InputFileWithoutQuery(t *testing.T) {
	file := filepath.Join(t.TempDir(), "f.json")
	require.NoError(t, os.WriteFile(file, []byte(`{"a":1,"b":2}`), 0o600))
	withInputFile(t, file)
	m := newQueryModel(t, `{"a":1,"b":2}`)

	// No query applied: the input file is written without a question.
	typeCommand(m, "w")

	require.Nil(t, m.confirm)
	want := "{\n  \"a\": 1,\n  \"b\": 2\n}\n"
	require.Equal(t, want, readFile(t, file))
	info, err := os.Stat(file)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	require.Equal(t, fmt.Sprintf(`"%s" 4L, %dB written`, file, len(want)), m.message.text)
}

func TestWrite_QueryResultToInputFileAsks(t *testing.T) {
	const input = `{"a": {"b": 1}}`
	file := filepath.Join(t.TempDir(), "f.json")
	require.NoError(t, os.WriteFile(file, []byte(input), 0o644))
	withInputFile(t, file)
	m := newQueryModel(t, input)
	drain(m, m.doQuery(".a"))
	require.NotNil(t, m.original)

	typeCommand(m, "w")
	require.NotNil(t, m.confirm)
	require.Equal(t, `Overwrite "`+file+`" with the query result? (y/n)`, m.confirm.prompt)
	require.Contains(t, m.View(), m.clip(m.confirm.prompt))
	require.Equal(t, m.termHeight-2, m.viewHeight())

	// n keeps the file.
	answer(m, "n")
	require.Nil(t, m.confirm)
	require.Nil(t, m.message)
	require.Equal(t, input, readFile(t, file))

	// y writes the result.
	typeCommand(m, "w")
	require.NotNil(t, m.confirm)
	answer(m, "y")
	require.Nil(t, m.confirm)
	want := "{\n  \"b\": 1\n}\n"
	require.Equal(t, want, readFile(t, file))
	require.Equal(t, fmt.Sprintf(`"%s" 3L, %dB written`, file, len(want)), m.message.text)
}

func TestWrite_QueryResultToInputFileByRelativePath(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	require.NoError(t, os.WriteFile("f.json", []byte(`[1, 2]`), 0o644))
	withInputFile(t, "f.json")
	m := newQueryModel(t, `[1, 2]`)
	drain(m, m.doQuery("len"))

	typeCommand(m, "w "+filepath.Join(dir, "f.json"))

	require.NotNil(t, m.confirm, "the same file by another path still asks")
	answer(m, "y")
	require.Equal(t, "2\n", readFile(t, "f.json"))
}

func TestWrite_BangSkipsConfirmation(t *testing.T) {
	file := filepath.Join(t.TempDir(), "f.json")
	require.NoError(t, os.WriteFile(file, []byte(`{"a": 1}`), 0o644))
	withInputFile(t, file)
	m := newQueryModel(t, `{"a": 1}`)
	drain(m, m.doQuery(".a"))

	typeCommand(m, "w!")

	require.Nil(t, m.confirm)
	require.Equal(t, "1\n", readFile(t, file))
}

func TestWrite_ExistingFileAsks(t *testing.T) {
	m := newQueryModel(t, `{"a": 1}`)
	out := filepath.Join(t.TempDir(), "out.json")
	require.NoError(t, os.WriteFile(out, []byte("old"), 0o644))

	typeCommand(m, "w "+out)
	require.NotNil(t, m.confirm)
	require.Equal(t, `"`+out+`" exists, overwrite? (y/n)`, m.confirm.prompt)
	answer(m, "q") // Any key but y cancels.
	require.Nil(t, m.confirm)
	require.Equal(t, "old", readFile(t, out))

	typeCommand(m, "w! "+out)
	require.Nil(t, m.confirm)
	require.Equal(t, "{\n  \"a\": 1\n}\n", readFile(t, out))
}

func TestWrite_ExpandsHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // Windows
	m := newQueryModel(t, `1`)

	typeCommand(m, "w ~/out.json")

	require.Equal(t, "1\n", readFile(t, filepath.Join(home, "out.json")))
}

func TestWrite_DeletedNode(t *testing.T) {
	m := newQueryModel(t, `{"a": 1, "b": [2, 3]}`)
	out := filepath.Join(t.TempDir(), "out.json")
	m.cursor = 2 // "b"
	m.deleteAtCursor()

	typeCommand(m, "w "+out)

	require.Equal(t, "{\n  \"a\": 1\n}\n", readFile(t, out))
}

func TestWrite_CollapsedAndWrapped(t *testing.T) {
	long := `"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"`
	m := newQueryModel(t, `{"a": {"b": [1, 2]}, "s": `+long+`}`)
	out := filepath.Join(t.TempDir(), "out.json")
	jsonx.Wrap(m.top, 20)
	require.NotNil(t, m.top.Next.Next.Next.Next.Next.Next.Next.ChunkEnd, "the string is wrapped")
	m.top.Next.Collapse() // "a"
	require.True(t, m.top.Next.IsCollapsed())

	typeCommand(m, "w "+out)

	want := "{\n  \"a\": {\n    \"b\": [\n      1,\n      2\n    ]\n  },\n  \"s\": " + long + "\n}\n"
	require.Equal(t, want, readFile(t, out))
}

func TestWrite_SkipsTextLines(t *testing.T) {
	m := newQueryModel(t, `{"a": 1}`)
	drain(m, m.doQuery(`x => { println("note"); return x.a }`))
	require.Equal(t, []string{"note", "1"}, lines(m))
	out := filepath.Join(t.TempDir(), "out.json")

	typeCommand(m, "w "+out)

	require.Equal(t, "1\n", readFile(t, out))
}

func TestWrite_Blocked(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out.json")

	t.Run("loading", func(t *testing.T) {
		m := newQueryModel(t, `{"a": 1}`)
		m.eof = false
		typeCommand(m, "w "+out)
		require.Equal(t, "Input is still loading", m.message.text)
	})

	t.Run("query running", func(t *testing.T) {
		m := newQueryModel(t, `{"a": 1}`)
		m.eof = false
		cmd := m.doQuery(".a")
		m.eof = true
		typeCommand(m, "w "+out)
		require.Equal(t, "Query is still running", m.message.text)
		m.Update(eofMsg{})
		drain(m, cmd)
	})

	t.Run("query errors", func(t *testing.T) {
		m := newQueryModel(t, `{"a": 1}`)
		drain(m, m.doQuery("x => x.a.b.c"))
		require.NotEmpty(t, m.queryErrors)
		typeCommand(m, "w "+out)
		require.Equal(t, "Query result has errors", m.message.text)
	})

	t.Run("empty", func(t *testing.T) {
		m := newQueryModel(t)
		typeCommand(m, "w "+out)
		require.Equal(t, "Nothing to write", m.message.text)
	})

	t.Run("yaml input", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "f.yaml")
		require.NoError(t, os.WriteFile(file, []byte("a: 1\n"), 0o644))
		withInputFile(t, file)
		flagYaml = true
		t.Cleanup(func() { flagYaml = false })
		m := newQueryModel(t, `{"a": 1}`)
		typeCommand(m, "w")
		require.Equal(t, `Can't write JSON over "`+file+`", write to another file`, m.message.text)
		require.Equal(t, "a: 1\n", readFile(t, file))
	})

	_, err := os.Stat(out)
	require.True(t, os.IsNotExist(err), "nothing was written")
}

func TestWrite_Error(t *testing.T) {
	m := newQueryModel(t, `1`)
	out := filepath.Join(t.TempDir(), "missing", "out.json")

	typeCommand(m, "w "+out)

	require.NotNil(t, m.message)
	require.True(t, m.message.isErr)
	require.Equal(t, `Can't write "`+out+`": no such file or directory`, m.message.text)
}

func TestWrite_RawInputFileRefused(t *testing.T) {
	file := filepath.Join(t.TempDir(), "notes.txt")
	require.NoError(t, os.WriteFile(file, []byte("one\ntwo\n"), 0o644))
	withInputFile(t, file)
	flagRaw = true
	t.Cleanup(func() { flagRaw = false })
	m := newQueryModel(t, `"one"`, `"two"`)

	typeCommand(m, "w")

	require.Equal(t, `Can't write JSON over "`+file+`", write to another file`, m.message.text)
	require.Equal(t, "one\ntwo\n", readFile(t, file))

	out := filepath.Join(t.TempDir(), "out.json")
	typeCommand(m, "w "+out)
	require.Equal(t, "\"one\"\n\"two\"\n", readFile(t, out))
}

func TestWrite_RecoveredTextRefused(t *testing.T) {
	m := newQueryModel(t, `{"a": 1}`)
	m.appendNode(&jsonx.Node{Kind: jsonx.Err, Value: "HTTP/1.1 200 OK"})
	m.appendNode(parseDoc(t, `{"a": 2}`))
	out := filepath.Join(t.TempDir(), "out.json")

	typeCommand(m, "w "+out)

	require.Equal(t, "Input contains text that is not JSON", m.message.text)
	require.NoFileExists(t, out)

	// A query result holds no recovered text; it can be written.
	drain(m, m.doQuery(".a"))
	require.Equal(t, []string{"1", "2"}, lines(m))
	typeCommand(m, "w "+out)
	require.Equal(t, "1\n2\n", readFile(t, out))
}

func TestWrite_ReloadErrorRefused(t *testing.T) {
	m := newQueryModel(t, `{"a": 1}`)
	m.loadGen = 1
	m.Update(errorMsg{err: fmt.Errorf("unexpected end of input\n  at line 3"), gen: 1})
	out := filepath.Join(t.TempDir(), "out.json")

	typeCommand(m, "w "+out)

	require.Equal(t, "Input has errors: unexpected end of input", m.message.text)
	require.NoFileExists(t, out)
}

// A reload that fails while a query is shown puts its error in the result
// view only. Clearing the query must not make the read part writable.
func TestWrite_ReloadErrorSurvivesClearingQuery(t *testing.T) {
	file := filepath.Join(t.TempDir(), "f.json")
	require.NoError(t, os.WriteFile(file, []byte(`{"a": 1}`+"\n"+`{"a": 2`), 0o644))
	withInputFile(t, file)
	m := newQueryModel(t, `{"a": 1}`)
	m.loadGen = 1
	drain(m, m.doQuery(".a"))
	m.Update(errorMsg{err: fmt.Errorf("unexpected end of input"), gen: 1})
	require.Equal(t, []string{"1", "unexpected end of input"}, lines(m))

	drain(m, m.clearQuery())
	require.Nil(t, m.original)
	require.Equal(t, []string{"{", `"a"1`, "}"}, lines(m), "the error line is gone with the result view")

	typeCommand(m, "w")

	require.Equal(t, "Input has errors: unexpected end of input", m.message.text)
	require.Equal(t, `{"a": 1}`+"\n"+`{"a": 2`, readFile(t, file), "the file keeps its content")

	// A reload that succeeds makes the input writable again.
	drain(m, m.reload())
	require.Nil(t, m.loadErr)
}

func TestWrite_ResultWithoutDocuments(t *testing.T) {
	m := newQueryModel(t, `{"a": 1}`)
	drain(m, m.doQuery(`x => { println("note"); return skip }`))
	require.Equal(t, []string{"note"}, lines(m))
	out := filepath.Join(t.TempDir(), "out.json")
	require.NoError(t, os.WriteFile(out, []byte("old"), 0o644))

	typeCommand(m, "w! "+out)

	require.Equal(t, "Nothing to write", m.message.text)
	require.Equal(t, "old", readFile(t, out))
}

func TestWrite_HonorsIndent(t *testing.T) {
	old := ident.Ident
	ident.Ident = "\t"
	t.Cleanup(func() { ident.Ident = old })
	m := newQueryModel(t, `{"a": [1]}`)
	out := filepath.Join(t.TempDir(), "out.json")

	typeCommand(m, "w "+out)

	require.Equal(t, "{\n\t\"a\": [\n\t\t1\n\t]\n}\n", readFile(t, out))
}

func TestWrite_DeletedAfterCollapsedAndWrapped(t *testing.T) {
	long := `"` + strings.Repeat("x", 40) + `"`
	m := newQueryModel(t, `{"a": {"x": 1}, "b": 2, "s": `+long+`, "t": 3}`)
	jsonx.Wrap(m.top, 20)
	m.top.Next.Collapse() // "a"
	out := filepath.Join(t.TempDir(), "out.json")

	m.cursor = rowOf(t, m, `"b"`) // after the collapsed "a"
	m.deleteAtCursor()
	m.cursor = rowOf(t, m, `"t"`) // after the wrapped "s"
	m.deleteAtCursor()

	typeCommand(m, "w "+out)

	require.Equal(t, "{\n  \"a\": {\n    \"x\": 1\n  },\n  \"s\": "+long+"\n}\n", readFile(t, out))
}

// rowOf returns the view row showing key.
func rowOf(t *testing.T, m *model, key string) int {
	t.Helper()
	for i := 0; m.at(i) != nil; i++ {
		if m.at(i).Key == key {
			return i
		}
	}
	t.Fatalf("key %s not shown", key)
	return -1
}

func TestWrite_MouseKeepsConfirmation(t *testing.T) {
	m := newQueryModel(t, `{"a": 1}`)
	out := filepath.Join(t.TempDir(), "out.json")
	require.NoError(t, os.WriteFile(out, []byte("old"), 0o644))

	typeCommand(m, "w "+out)
	require.NotNil(t, m.confirm)
	m.Update(tea.MouseMsg{Button: tea.MouseButtonWheelDown})
	require.NotNil(t, m.confirm)
	require.Equal(t, 0, m.cursor)

	answer(m, "y")
	require.Equal(t, "{\n  \"a\": 1\n}\n", readFile(t, out))
}

func TestWrite_PromptClippedToWidth(t *testing.T) {
	m := newQueryModel(t, `1`)
	m.termWidth = 30
	out := filepath.Join(t.TempDir(), strings.Repeat("d", 40), "out.json")

	typeCommand(m, "w "+out)

	require.True(t, m.message.isErr)
	view := m.View()
	last := view[strings.LastIndex(view, "\n")+1:]
	require.Contains(t, last, `Can't write "`)
	require.Contains(t, last, "…")
	require.LessOrEqual(t, runewidth.StringWidth(ansi.Strip(last)), 30)
}

// typeCommandCmd is typeCommand returning the command of enter, for
// commands that quit.
func typeCommandCmd(m *model, line string) tea.Cmd {
	typeKeys(m, ":"+line)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	return cmd
}

func isQuit(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

func TestWriteQuit(t *testing.T) {
	m := newQueryModel(t, `{"a": 1}`)
	out := filepath.Join(t.TempDir(), "out.json")

	cmd := typeCommandCmd(m, "wq "+out)

	require.True(t, isQuit(cmd))
	require.Equal(t, "{\n  \"a\": 1\n}\n", readFile(t, out))
}

func TestWriteQuit_AsksThenQuits(t *testing.T) {
	file := filepath.Join(t.TempDir(), "f.json")
	require.NoError(t, os.WriteFile(file, []byte(`{"a": 1}`), 0o644))
	withInputFile(t, file)
	m := newQueryModel(t, `{"a": 1}`)
	drain(m, m.doQuery(".a"))

	cmd := typeCommandCmd(m, "wq")
	require.Nil(t, cmd)
	require.NotNil(t, m.confirm)

	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	require.True(t, isQuit(cmd))
	require.Equal(t, "1\n", readFile(t, file))

	// ! writes without asking.
	m = newQueryModel(t, `{"a": 2}`)
	drain(m, m.doQuery(".a"))
	require.True(t, isQuit(typeCommandCmd(m, "wq!")))
	require.Equal(t, "2\n", readFile(t, file))
}

func TestWriteQuit_StaysOnError(t *testing.T) {
	m := newQueryModel(t, `{"a": 1}`)
	out := filepath.Join(t.TempDir(), "missing", "out.json")

	cmd := typeCommandCmd(m, "wq "+out)

	require.Nil(t, cmd)
	require.True(t, m.message.isErr)
	require.Contains(t, m.message.text, "Can't write")

	m.eof = false
	require.Nil(t, typeCommandCmd(m, "wq "+out))
	require.Equal(t, "Input is still loading", m.message.text)
}
