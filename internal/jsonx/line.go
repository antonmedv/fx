package jsonx

import (
	"bufio"
	"io"
	"strconv"
	"strings"
)

type LineParser struct {
	buf        *bufio.Reader
	eof        error
	lineNumber int
}

func NewLineParser(in io.Reader) *LineParser {
	p := &LineParser{
		buf:        bufio.NewReader(in),
		lineNumber: 1,
	}
	return p
}

func (p *LineParser) Parse() (*Node, error) {
	if p.eof != nil {
		return nil, p.eof
	}
	b, err := p.buf.ReadBytes('\n')
	if err != nil {
		if err == io.EOF {
			p.eof = err
		} else {
			return nil, err
		}
	}
	if len(b) == 0 {
		return nil, err
	}
	s := strings.TrimRight(string(b), "\r\n")
	quoted := strconv.Quote(s)
	node := &Node{
		Kind:       String,
		Value:      quoted,
		LineNumber: p.lineNumber,
		Depth:      0,
	}
	p.lineNumber++
	return node, nil
}

// More reports whether another line remains, blocking like
// JsonParser.More. A read error is kept for later More and Parse calls.
func (p *LineParser) More() (bool, error) {
	if p.eof != nil {
		if p.eof == io.EOF {
			return false, nil
		}
		return false, p.eof
	}
	if _, err := p.buf.Peek(1); err != nil {
		// Kept: bufio returns a read error only once, and after save()
		// the file is closed.
		p.eof = err
		if err == io.EOF {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (p *LineParser) Recover() *Node {
	return nil
}
