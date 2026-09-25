package main

import (
	"io"
	"sync"

	. "github.com/antonmedv/fx/internal/jsonx"
)

func (m *model) doQuery(query string) {
	//args := []string{query}
	//exitCode := engine.Start(parser, args, opts)
}

// nodesParser feeds already parsed top-level documents to the engine
// without copying them. The UI goroutine appends documents to the list
// and calls publish; the engine goroutine walks the list only up to the
// last published document, so Next pointers are read after the mutex
// establishes happens-before.
type nodesParser struct {
	mu        sync.Mutex
	cond      *sync.Cond
	head      *Node // first document, nil until published
	last      *Node // last published document
	returned  *Node // last document returned by Parse
	eof       bool
	cancelled bool
}

func newNodesParser(head, last *Node, eof bool) *nodesParser {
	p := &nodesParser{head: head, last: last, eof: eof}
	p.cond = sync.NewCond(&p.mu)
	return p
}

func (p *nodesParser) Parse() (*Node, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for {
		if p.cancelled {
			return nil, io.EOF
		}
		if p.returned != p.last {
			if p.returned == nil {
				p.returned = p.head
			} else {
				p.returned = nextDoc(p.returned)
			}
			return p.returned, nil
		}
		if p.eof {
			return nil, io.EOF
		}
		p.cond.Wait()
	}
}

func (p *nodesParser) Recover() *Node {
	return nil
}

// publish makes doc (already linked after the previous last) available.
func (p *nodesParser) publish(doc *Node) {
	p.mu.Lock()
	if p.head == nil {
		p.head = doc
	}
	p.last = doc
	p.mu.Unlock()
	p.cond.Broadcast()
}

func (p *nodesParser) setEOF() {
	p.mu.Lock()
	p.eof = true
	p.mu.Unlock()
	p.cond.Broadcast()
}

// stop unblocks Parse, which then returns io.EOF.
func (p *nodesParser) stop() {
	p.mu.Lock()
	p.cancelled = true
	p.mu.Unlock()
	p.cond.Broadcast()
}

// nextDoc returns the top-level document following doc. Node.Bottom can't
// be used as it walks to the end of the whole list.
func nextDoc(doc *Node) *Node {
	if doc.End != nil {
		// End.Next is kept by Adjacent even if doc is collapsed.
		return doc.End.Next
	}
	if doc.ChunkEnd != nil {
		return doc.ChunkEnd.Next
	}
	return doc.Next
}
