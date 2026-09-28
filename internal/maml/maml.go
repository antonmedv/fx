// Package maml converts MAML (https://maml.dev) to JSON.
package maml

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/maml-dev/go-maml/ast"

	"github.com/antonmedv/fx/internal/engine"
)

// ToJSON converts a MAML document to JSON. Object keys keep their order,
// numbers keep their source text, and raw strings become plain strings.
// Comments are dropped.
func ToJSON(in []byte) ([]byte, error) {
	src := strings.TrimPrefix(string(in), "\xEF\xBB\xBF")
	doc, err := ast.Parse(src)
	if err != nil {
		return nil, fmt.Errorf("maml: %w", err)
	}
	var out bytes.Buffer
	write(&out, doc.Value)
	return out.Bytes(), nil
}

func write(out *bytes.Buffer, node ast.ValueNode) {
	switch n := node.(type) {
	case *ast.ObjectNode:
		out.WriteByte('{')
		for i, p := range n.Properties {
			if i > 0 {
				out.WriteByte(',')
			}
			out.WriteString(engine.Quote(p.Key.KeyValue()))
			out.WriteByte(':')
			write(out, p.Value)
		}
		out.WriteByte('}')
	case *ast.ArrayNode:
		out.WriteByte('[')
		for i, e := range n.Elements {
			if i > 0 {
				out.WriteByte(',')
			}
			write(out, e.Value)
		}
		out.WriteByte(']')
	case *ast.StringNode:
		out.WriteString(engine.Quote(n.Value))
	case *ast.RawStringNode:
		out.WriteString(engine.Quote(n.Value))
	case *ast.IntegerNode:
		out.WriteString(n.Raw)
	case *ast.FloatNode:
		out.WriteString(n.Raw)
	case *ast.BooleanNode:
		if n.Value {
			out.WriteString("true")
		} else {
			out.WriteString("false")
		}
	case *ast.NullNode:
		out.WriteString("null")
	}
}
