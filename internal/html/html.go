// Package html converts HTML to JSON.
package html

import (
	"bytes"
	"regexp"

	nethtml "golang.org/x/net/html"

	"github.com/antonmedv/fx/internal/charset"
	"github.com/antonmedv/fx/internal/xml"
)

// ToJSON parses HTML the way a browser does and writes the document with
// the mapping of xml.ToJSON: {"html": {"head": ..., "body": ...}}. The
// parser lowercases names, closes void and unclosed elements and adds
// the elements HTML implies, so a fragment ends up under "body". A byte
// order mark or a <meta charset> in the first 1024 bytes selects the
// charset; an unknown one is read as UTF-8. Blank input gives no output.
func ToJSON(in []byte) ([]byte, error) {
	in, err := decode(in)
	if err != nil {
		return nil, err
	}
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

// reMetaCharset matches <meta charset="x"> and the charset parameter of
// <meta http-equiv="Content-Type" content="text/html; charset=x">.
var reMetaCharset = regexp.MustCompile(`(?i)<meta[^>]*?charset\s*=\s*["']?\s*([a-z0-9_.:-]+)`)

// decode returns in as UTF-8 by its byte order mark or meta charset.
func decode(in []byte) ([]byte, error) {
	enc, in := charset.Sniff(in)
	if enc == nil {
		head := in
		if len(head) > 1024 {
			head = head[:1024]
		}
		m := reMetaCharset.FindSubmatch(head)
		if m == nil {
			return in, nil
		}
		if enc = charset.Lookup(string(m[1])); enc == nil {
			return in, nil
		}
	}
	out, err := charset.Decode(enc, in)
	if err != nil {
		return nil, err
	}
	return out, nil
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
			el.AddText(c.Data)
		}
	}
	return el
}
