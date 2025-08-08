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

func (p *LineParser) Recover() *Node {
	return nil
}

func (p *LineParser) SkipWhitespace() {}
