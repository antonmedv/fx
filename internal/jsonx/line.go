package jsonx

import (
	"bufio"
	"io"
	"strconv"
)

type LineParser struct {
	buf *bufio.Reader
	eof error
}

func NewLineParser(in io.Reader) *LineParser {
	p := &LineParser{
		buf: bufio.NewReader(in),
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
	if b[len(b)-1] == '\n' {
		b = b[:len(b)-1] // Trim "\n" char at the end.
	}
	quoted := strconv.Quote(string(b))
	node := &Node{Kind: String, Value: []byte(quoted)}
	return node, nil
}

func (p *LineParser) Recover() *Node {
	return nil
}
