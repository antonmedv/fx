package main

import (
	"bytes"
	"io"
	"os"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/antonmedv/fx/internal/engine"
	. "github.com/antonmedv/fx/internal/jsonx"
	"github.com/antonmedv/fx/internal/toml"
)

// newParser creates the parser for the input format chosen by the flags.
func newParser(src io.Reader) (engine.Parser, error) {
	switch {
	case flagYaml:
		b, err := io.ReadAll(src)
		if err != nil {
			return nil, err
		}
		jsonBytes, err := parseYAML(b)
		if err != nil {
			return nil, err
		}
		return NewJsonParser(bytes.NewReader(jsonBytes), flagStrict), nil
	case flagToml:
		b, err := io.ReadAll(src)
		if err != nil {
			return nil, err
		}
		jsonBytes, err := toml.ToJSON(b)
		if err != nil {
			return nil, err
		}
		return NewJsonParser(bytes.NewReader(jsonBytes), flagStrict), nil
	case flagRaw:
		return NewLineParser(src), nil
	default:
		return NewJsonParser(src, flagStrict), nil
	}
}

// loader reads the input in its own goroutine and delivers the documents
// to the UI. A reload stops the loader and starts a new generation.
type loader struct {
	gen    uint64
	stop   chan struct{}
	sendFn func(tea.Msg) // delivers messages directly (tea.Program.Send); nil to use msgs
	msgs   chan tea.Msg  // read by wait
}

// read parses the input until EOF or an error. It reports whether EOF was
// reached and delivered.
func (l *loader) read(parser engine.Parser) bool {
	firstOk := false
	for {
		node, err := parser.Parse()
		if err != nil {
			if err == io.EOF {
				return l.send(eofMsg{gen: l.gen})
			}
			if flagStrict {
				l.send(errorMsg{err: err, gen: l.gen})
				return false
			}
			textNode := parser.Recover()
			if !firstOk && !strings.HasPrefix(textNode.Value, "HTTP") {
				l.send(errorMsg{err: err, gen: l.gen})
				return false
			}
			node = textNode
		} else {
			firstOk = true
		}
		if !l.send(nodeMsg{node: node, gen: l.gen}) {
			return false
		}
	}
}

// send delivers msg, unless the loader is stopped.
func (l *loader) send(msg tea.Msg) bool {
	if l.sendFn != nil {
		select {
		case <-l.stop:
			return false
		default:
		}
		l.sendFn(msg)
		return true
	}
	select {
	case l.msgs <- msg:
		return true
	case <-l.stop:
		return false
	}
}

// wait reads the next message of the loader. It is re-issued after every
// document; nil if messages are delivered directly.
func (l *loader) wait() tea.Cmd {
	if l == nil || l.msgs == nil {
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
	if engine.Input != nil {
		_ = engine.Input.Close()
		engine.Input = nil
	}
	m.loadGen++

	m.reloadPos = m.cursorReloadPos()
	m.cancelPreview()
	query := ""
	if m.original != nil {
		// The query result has no place in the new file; run the query again.
		if m.query != nil {
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

	f, err := os.Open(engine.FilePath)
	if err != nil {
		m.showReloadError(err)
		return nil
	}
	engine.Input = f

	l := &loader{gen: m.loadGen, stop: make(chan struct{}), msgs: make(chan tea.Msg)}
	m.loader = l
	go func() {
		parser, err := newParser(f)
		if err != nil {
			l.send(errorMsg{err: err, gen: l.gen})
			return
		}
		l.read(parser)
	}()

	cmds := []tea.Cmd{l.wait(), m.spinner.Tick}
	if query != "" {
		cmds = append(cmds, m.startQuery(query))
	}
	return tea.Batch(cmds...)
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
// and reloaded again.
func (m *model) showReloadError(err error) {
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
