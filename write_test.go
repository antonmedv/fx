package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"

	"github.com/antonmedv/fx/internal/engine"
	"github.com/antonmedv/fx/internal/jsonx"
)

// withInputFile makes path the file fx was opened with.
func withInputFile(t *testing.T, path string) {
	t.Helper()
	old := engine.FilePath
	engine.FilePath = path
	t.Cleanup(func() { engine.FilePath = old })
}

// runCommand types :line and presses enter.
func runCommand(m *model, line string) {
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

	runCommand(m, "w "+out)

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

	runCommand(m, "write "+out)

	require.Equal(t, "\"hello\"\n", readFile(t, out))
}

func TestWrite_SeveralDocuments(t *testing.T) {
	m := newQueryModel(t, `{"a": 1}`, `[1]`, `2`)
	out := filepath.Join(t.TempDir(), "out.json")

	runCommand(m, "w "+out)

	require.Equal(t, "{\n  \"a\": 1\n}\n[\n  1\n]\n2\n", readFile(t, out))
}

func TestWrite_NoFileName(t *testing.T) {
	withInputFile(t, "")
	m := newQueryModel(t, `{"a": 1}`)

	runCommand(m, "w")

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
	runCommand(m, "w")

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

	runCommand(m, "w")
	require.NotNil(t, m.confirm)
	require.Equal(t, `Overwrite "`+file+`" with the query result? (y/n)`, m.confirm.prompt)
	require.Contains(t, m.View(), m.confirm.prompt)
	require.Equal(t, m.termHeight-2, m.viewHeight())

	// n keeps the file.
	answer(m, "n")
	require.Nil(t, m.confirm)
	require.Nil(t, m.message)
	require.Equal(t, input, readFile(t, file))

	// y writes the result.
	runCommand(m, "w")
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

	runCommand(m, "w "+filepath.Join(dir, "f.json"))

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

	runCommand(m, "w!")

	require.Nil(t, m.confirm)
	require.Equal(t, "1\n", readFile(t, file))
}

func TestWrite_ExistingFileAsks(t *testing.T) {
	m := newQueryModel(t, `{"a": 1}`)
	out := filepath.Join(t.TempDir(), "out.json")
	require.NoError(t, os.WriteFile(out, []byte("old"), 0o644))

	runCommand(m, "w "+out)
	require.NotNil(t, m.confirm)
	require.Equal(t, `"`+out+`" exists, overwrite? (y/n)`, m.confirm.prompt)
	answer(m, "q") // Any key but y cancels.
	require.Nil(t, m.confirm)
	require.Equal(t, "old", readFile(t, out))

	runCommand(m, "w! "+out)
	require.Nil(t, m.confirm)
	require.Equal(t, "{\n  \"a\": 1\n}\n", readFile(t, out))
}

func TestWrite_ExpandsHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	m := newQueryModel(t, `1`)

	runCommand(m, "w ~/out.json")

	require.Equal(t, "1\n", readFile(t, filepath.Join(home, "out.json")))
}

func TestWrite_DeletedNode(t *testing.T) {
	m := newQueryModel(t, `{"a": 1, "b": [2, 3]}`)
	out := filepath.Join(t.TempDir(), "out.json")
	m.cursor = 2 // "b"
	m.deleteAtCursor()

	runCommand(m, "w "+out)

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

	runCommand(m, "w "+out)

	want := "{\n  \"a\": {\n    \"b\": [\n      1,\n      2\n    ]\n  },\n  \"s\": " + long + "\n}\n"
	require.Equal(t, want, readFile(t, out))
}

func TestWrite_SkipsTextLines(t *testing.T) {
	m := newQueryModel(t, `{"a": 1}`)
	drain(m, m.doQuery(`x => { println("note"); return x.a }`))
	require.Equal(t, []string{"note", "1"}, lines(m))
	out := filepath.Join(t.TempDir(), "out.json")

	runCommand(m, "w "+out)

	require.Equal(t, "1\n", readFile(t, out))
}

func TestWrite_Blocked(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out.json")

	t.Run("loading", func(t *testing.T) {
		m := newQueryModel(t, `{"a": 1}`)
		m.eof = false
		runCommand(m, "w "+out)
		require.Equal(t, "Input is still loading", m.message.text)
	})

	t.Run("query running", func(t *testing.T) {
		m := newQueryModel(t, `{"a": 1}`)
		m.eof = false
		cmd := m.doQuery(".a")
		m.eof = true
		runCommand(m, "w "+out)
		require.Equal(t, "Query is still running", m.message.text)
		m.Update(eofMsg{})
		drain(m, cmd)
	})

	t.Run("query errors", func(t *testing.T) {
		m := newQueryModel(t, `{"a": 1}`)
		drain(m, m.doQuery("x => x.a.b.c"))
		require.NotEmpty(t, m.queryErrors)
		runCommand(m, "w "+out)
		require.Equal(t, "Query result has errors", m.message.text)
	})

	t.Run("empty", func(t *testing.T) {
		m := newQueryModel(t)
		runCommand(m, "w "+out)
		require.Equal(t, "Nothing to write", m.message.text)
	})

	t.Run("yaml input", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "f.yaml")
		require.NoError(t, os.WriteFile(file, []byte("a: 1\n"), 0o644))
		withInputFile(t, file)
		flagYaml = true
		t.Cleanup(func() { flagYaml = false })
		m := newQueryModel(t, `{"a": 1}`)
		runCommand(m, "w")
		require.Equal(t, `Can't write JSON over "`+file+`", write to another file`, m.message.text)
		require.Equal(t, "a: 1\n", readFile(t, file))
	})

	_, err := os.Stat(out)
	require.True(t, os.IsNotExist(err), "nothing was written")
}

func TestWrite_Error(t *testing.T) {
	m := newQueryModel(t, `1`)
	out := filepath.Join(t.TempDir(), "missing", "out.json")

	runCommand(m, "w "+out)

	require.NotNil(t, m.message)
	require.True(t, m.message.isErr)
	require.Equal(t, `Can't write "`+out+`": no such file or directory`, m.message.text)
}
