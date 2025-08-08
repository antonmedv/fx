package engine

import (
	"io"

	"github.com/antonmedv/fx/internal/jsonx"
)

func Slurp(parser Parser, writeErr func(string)) (Parser, bool) {
	arr := &jsonx.Node{
		Kind:       jsonx.Array,
		Value:      "[",
		LineNumber: 1,
	}

	end := arr
	for {
		node, err := parser.Parse()
		if err != nil {
			if err == io.EOF {
				break
			}
			writeErr(err.Error())
			return nil, false
		}

		node.Parent = arr

		it := node
		for it != nil {
			it.Depth++
			it.LineNumber++
			it = it.Next
		}

		end.Next = node
		end = node.Bottom()
		end.Comma = true
	}

	end.Comma = false
	end.Next = &jsonx.Node{
		Kind:       jsonx.Array,
		LineNumber: end.LineNumber + 1,
		Value:      "]",
	}
	arr.End = end.Next

	return &slurpParser{node: arr}, true
}

type slurpParser struct {
	node *jsonx.Node
}

func (p *slurpParser) Parse() (*jsonx.Node, error) {
	if p.node == nil {
		return nil, io.EOF
	}
	node := p.node
	p.node = nil
	return node, nil
}

func (p *slurpParser) Recover() *jsonx.Node {
	return nil
}

func (p *slurpParser) SkipWhitespace() {}
