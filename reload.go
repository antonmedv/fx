package main

import (
	"bytes"
	"io"
	"os"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/antonmedv/fx/internal/engine"
	. "github.com/antonmedv/fx/internal/jsonx"
)

// newParser creates the parser for the input format chosen by the flags.
// A format other than JSON is converted to JSON as a whole first.
func newParser(src io.Reader) (engine.Parser, error) {
	if flagRaw {
		return NewLineParser(src), nil
	}
	if inputFormat != nil {
		b, err := io.ReadAll(src)
		if err != nil {
			return nil, err
		}
		jsonBytes, err := inputFormat.ToJSON(b)
		if err != nil {
			return nil, err
		}
		// The converter validated the source; --strict is about JSON
		// syntax, and the converted JSON may hold Infinity or NaN.
		return NewJsonParser(bytes.NewReader(jsonBytes), false), nil
	}
	return NewJsonParser(src, flagStrict), nil
}

// loader reads the input in its own goroutine and delivers the documents
// to the UI through wait. A reload stops the loader and starts a new
// generation.
type loader struct {
	gen  uint64
	stop chan struct{}
	msgs chan tea.Msg
}

// newLoader starts reading the parser returned by open. The loader owns
// file (nil for stdin) and closes it once done: closing it from outside
// would fail the parser's reads, which panic outside Parse.
func newLoader(gen uint64, file io.Closer, open func() (engine.Parser, error)) *loader {
	l := &loader{gen: gen, stop: make(chan struct{}), msgs: make(chan tea.Msg)}
	go func() {
		if file != nil {
			defer file.Close()
		}
		parser, err := open()
		if err != nil {
			l.send(errorMsg{err: err, gen: gen})
			return
		}
		l.read(parser)
	}()
	return l
}

// read parses the input until EOF or an error.
func (l *loader) read(parser engine.Parser) {
	firstOk := false
	for {
		node, err := parser.Parse()
		if err != nil {
			if err == io.EOF {
				l.send(eofMsg{gen: l.gen})
				return
			}
			if flagStrict {
				l.send(errorMsg{err: err, gen: l.gen})
				return
			}
			textNode := parser.Recover()
			if textNode == nil || !firstOk && !strings.HasPrefix(textNode.Value, "HTTP") {
				// LineParser can't recover, a read error ends it.
				l.send(errorMsg{err: err, gen: l.gen})
				return
			}
			node = textNode
		} else {
			firstOk = true
		}
		if !l.send(nodeMsg{node: node, gen: l.gen}) {
			return
		}
	}
}

// send delivers msg, unless the loader is stopped.
func (l *loader) send(msg tea.Msg) bool {
	select {
	case l.msgs <- msg:
		return true
	case <-l.stop:
		return false
	}
}

// wait reads the next message of the loader. It is re-issued after every
// document.
func (l *loader) wait() tea.Cmd {
	if l == nil {
		return nil
	}
	return func() tea.Msg {
		select {
		case msg := <-l.msgs:
			return msg
		case <-l.stop:
			return nil
		}
	}
}

func (l *loader) stopLoading() {
	if l == nil {
		return
	}
	select {
	case <-l.stop:
	default:
		close(l.stop)
	}
}

// reloadPos is the cursor position to restore once the reloaded
// document arrives.
type reloadPos struct {
	doc  int   // top-level document index
	path []any // path inside the document
	row  int   // cursor row on the screen
}

// reload reads the file again. The cursor stays on the same path; an
// applied query is run again on the new content.
func (m *model) reload() tea.Cmd {
	if engine.FilePath == "" {
		return nil
	}
	m.loader.stopLoading()
	m.loadGen++
	m.loadErr = nil

	m.reloadPos = m.cursorReloadPos()
	m.cancelPreview()
	query := ""
	if m.original != nil {
		// The query result has no place in the new file; run the query again.
		if m.query != nil && !m.restoring {
			query = m.query.query
		}
		m.stopQuery()
		m.original = nil
		m.query = nil
		m.restoring = false
		m.reloadPos = nil
	}
	m.resetView()
	m.eof = false

	var queryCmd tea.Cmd
	if query != "" {
		// Started before opening, so a failed open keeps the query for the
		// next reload.
		queryCmd = m.startQuery(query)
	}

	f, err := os.Open(engine.FilePath)
	if err != nil {
		engine.SetInput(nil)
		m.showReloadError(err)
		return queryCmd
	}
	engine.SetInput(f)
	m.loader = newLoader(m.loadGen, f, func() (engine.Parser, error) { return newParser(f) })
	return tea.Batch(m.loader.wait(), m.spinner.Tick, queryCmd)
}

// reloading reports whether a reloaded file is still being read.
func (m *model) reloading() bool {
	return m.loadGen > 0 && !m.eof
}

// setEOF marks the input as fully read.
func (m *model) setEOF() {
	m.eof = true
	m.reloadPos = nil
	if m.query != nil && m.query.nodes != nil {
		m.query.nodes.setEOF()
	}
}

// showReloadError shows err below what was read, so the file can be fixed
// and reloaded again. The query is stopped, not given EOF: the input is
// incomplete, and save() would overwrite the file with the part read. The
// error is kept in loadErr, as clearing a query drops the line shown.
func (m *model) showReloadError(err error) {
	m.loadErr = err
	m.stopQuery()
	m.setEOF()
	m.appendText(err.Error(), true)
}

// cursorReloadPos returns the cursor position to restore after a reload,
// nil if there is none.
func (m *model) cursorReloadPos() *reloadPos {
	at, ok := m.cursorPointsTo()
	if !ok {
		return nil
	}
	if at.IsWrap() {
		at = at.Parent
	}
	if at.Parent != nil && at.Parent.End == at {
		at = at.Parent // Closing bracket.
	}
	root := at.Root()
	doc := 0
	for it := m.top; it != root; it = nextDoc(it) {
		if it == nil {
			return nil
		}
		doc++
	}
	var path []any
	for it := at; it.Parent != nil; it = it.Parent {
		if it.Key != "" {
			key, err := strconv.Unquote(it.Key)
			if err != nil {
				return nil
			}
			path = append([]any{key}, path...)
		} else {
			path = append([]any{it.Index}, path...)
		}
	}
	return &reloadPos{doc: doc, path: path, row: m.cursor}
}

// restoreReloadPosition moves the cursor to the saved position once the
// document holding it arrives.
func (m *model) restoreReloadPosition(doc *Node) {
	p := m.reloadPos
	if p == nil {
		return
	}
	// p.doc counts down the documents still to arrive before the saved one;
	// doc may hold several (lines of recovered text).
	for ; doc != nil && p.doc > 0; doc = nextDoc(doc) {
		p.doc--
	}
	if doc == nil {
		return
	}
	m.reloadPos = nil
	n := doc.FindByPath(p.path)
	if n == nil {
		n = doc
	}
	for parent := n.Parent; parent != nil; parent = parent.Parent {
		parent.Expand()
	}
	// Keep the cursor on the same screen row.
	head, row := n, 0
	for row < p.row && head.Prev != nil {
		head = head.Prev
		row++
	}
	m.head = head
	m.cursor = row
	m.showCursor = true
	m.scrollIntoView()
	m.recordHistory()
}
