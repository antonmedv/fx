// Package html converts HTML to JSON.
package html

import (
	"bytes"

	nethtml "golang.org/x/net/html"

	"github.com/antonmedv/fx/internal/xml"
)

// ToJSON parses HTML the way a browser does and writes the document with
// the mapping of xml.ToJSON: {"html": {"head": ..., "body": ...}}. The
// parser lowercases names, closes void and unclosed elements and adds
// the elements HTML implies, so a fragment ends up under "body". Blank
// input gives no output.
func ToJSON(in []byte) ([]byte, error) {
	if len(bytes.TrimSpace(in)) == 0 {
		return nil, nil
	}
	doc, err := nethtml.Parse(bytes.NewReader(in))
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	for c := doc.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != nethtml.ElementNode {
			continue
		}
		if out.Len() > 0 {
			out.WriteByte('\n')
		}
		xml.WriteDocument(&out, convert(c))
	}
	return out.Bytes(), nil
}

func convert(n *nethtml.Node) *xml.Element {
	el := &xml.Element{Name: n.Data}
	for _, a := range n.Attr {
		name := a.Key
		if a.Namespace != "" {
			name = a.Namespace + ":" + a.Key
		}
		el.Attrs = append(el.Attrs, xml.Attr{Name: name, Value: a.Val})
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		switch c.Type {
		case nethtml.ElementNode:
			el.Children = append(el.Children, convert(c))
		case nethtml.TextNode:
			el.Text += c.Data
		}
	}
	return el
}
