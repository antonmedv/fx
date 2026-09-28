package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/antonmedv/fx/internal/engine"
	. "github.com/antonmedv/fx/internal/jsonx"
)

// write runs :w[rite][!] [file]: it writes the displayed JSON, the query
// result if a query is applied, to file, or to the input file without one.
// Overwriting the input file with a query result, or any other existing
// file, asks for confirmation, unless ! is given.
func (m *model) write(c call) tea.Cmd {
	if reason := m.writeBlocked(); reason != "" {
		return m.errorf("%s", reason)
	}
	path := c.arg
	if path == "" {
		if engine.FilePath == "" {
			return m.errorf("No file name")
		}
		path = engine.FilePath
	}
	path = expandHome(path)
	same := engine.FilePath != "" && sameFile(path, engine.FilePath)
	if same && (flagYaml || flagToml) {
		return m.errorf("Can't write JSON over \"%s\", write to another file", engine.FilePath)
	}
	write := func() tea.Cmd { return m.writeFile(path, same) }
	switch {
	case c.bang:
		return write()
	case same && m.original != nil:
		return m.ask(fmt.Sprintf("Overwrite \"%s\" with the query result?", path), write)
	case !same && fileExists(path):
		return m.ask(fmt.Sprintf("\"%s\" exists, overwrite?", path), write)
	}
	return write()
}

// writeBlocked returns why the view can't be written now, or "".
func (m *model) writeBlocked() string {
	switch {
	case !m.eof:
		return "Input is still loading"
	case m.restoring || m.query != nil && !m.query.done:
		return "Query is still running"
	case len(m.queryErrors) > 0:
		return "Query result has errors"
	case m.top == nil:
		return "Nothing to write"
	}
	return ""
}

// writeFile writes the displayed JSON to path. same reports whether path
// is the input file.
func (m *model) writeFile(path string, same bool) tea.Cmd {
	data := m.viewJSON()
	if same {
		// Windows can't rename over the open input file.
		engine.CloseInput()
	}
	if err := engine.WriteFile(path, data); err != nil {
		var pathErr *fs.PathError
		if errors.As(err, &pathErr) {
			return m.errorf("Can't write \"%s\": %v", path, pathErr.Err)
		}
		return m.errorf("%v", err)
	}
	return m.infof("\"%s\" %dL, %dB written", path, bytes.Count(data, []byte{'\n'}), len(data))
}

// viewJSON serializes the displayed documents as indented JSON, one after
// another, like fx prints them to stdout. Text lines (recovered non-JSON
// input, println output) are left out.
func (m *model) viewJSON() []byte {
	var b bytes.Buffer
	for doc := m.top; doc != nil; doc = nextDoc(doc) {
		if doc.Kind == Err {
			continue
		}
		writeIndented(&b, doc)
	}
	return b.Bytes()
}

// writeIndented serializes the top-level document doc with two spaces of
// indentation, the collapsed parts included.
func writeIndented(b *bytes.Buffer, doc *Node) {
	for it := doc; it != nil; {
		if !it.IsWrap() {
			for range it.Depth {
				b.WriteString("  ")
			}
			if it.Key != "" {
				b.WriteString(it.Key)
				b.WriteString(": ")
			}
			b.WriteString(it.Value)
			if it.Comma && it != doc.End {
				b.WriteByte(',')
			}
			b.WriteByte('\n')
		}
		if it == doc.End || doc.End == nil {
			break
		}
		if it.IsCollapsed() {
			it = it.Collapsed
		} else {
			it = it.Next
		}
	}
}

// expandHome replaces a leading ~ with the home directory.
func expandHome(path string) string {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[1:])
}

// sameFile reports whether a and b name the same file. A file that does
// not exist yet is compared by its absolute path.
func sameFile(a, b string) bool {
	ai, errA := os.Stat(a)
	bi, errB := os.Stat(b)
	if errA == nil && errB == nil {
		return os.SameFile(ai, bi)
	}
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)
	return errA == nil && errB == nil && absA == absB
}

func fileExists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}
