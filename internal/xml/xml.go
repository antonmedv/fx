// Package xml converts XML to JSON.
package xml

import (
	"bufio"
	"bytes"
	goxml "encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/antonmedv/fx/internal/engine"
)

// ToJSON converts XML to JSON, one document per top-level element. The
// mapping follows the xmltodict convention:
//
//	<a/>, <a></a>               "a": null
//	<a>text</a>                 "a": "text"
//	<a k="v">text</a>           "a": {"@k": "v", "#text": "text"}
//	<a><b>1</b><c/></a>         "a": {"b": "1", "c": null}
//	<a><b>1</b><b>2</b></a>     "a": {"b": ["1", "2"]}
//	<a>x<b/>y</a>               "a": {"b": null, "#text": "xy"}
//
// Values stay strings; text is trimmed and whitespace between elements
// is dropped. Names keep their namespace prefix. Comments, processing
// instructions and the DOCTYPE are dropped.
func ToJSON(in []byte) ([]byte, error) {
	in = bytes.TrimPrefix(in, []byte("\xEF\xBB\xBF"))
	d := goxml.NewDecoder(bytes.NewReader(in))
	d.CharsetReader = charsetReader
	errorf := func(format string, args ...any) error {
		line, column := d.InputPos()
		return fmt.Errorf("xml: line %d, column %d: %s", line, column, fmt.Sprintf(format, args...))
	}

	var out bytes.Buffer
	var stack []*Element
	for {
		// RawToken keeps namespace prefixes as written; the stack below
		// does the start/end matching Token would do.
		tok, err := d.RawToken()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case goxml.StartElement:
			el := &Element{Name: name(t.Name)}
			for _, a := range t.Attr {
				el.Attrs = append(el.Attrs, Attr{Name: name(a.Name), Value: a.Value})
			}
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				parent.Children = append(parent.Children, el)
			}
			stack = append(stack, el)
		case goxml.EndElement:
			if len(stack) == 0 {
				return nil, errorf("unexpected end tag </%s>", name(t.Name))
			}
			top := stack[len(stack)-1]
			if top.Name != name(t.Name) {
				return nil, errorf("element <%s> closed by </%s>", top.Name, name(t.Name))
			}
			stack = stack[:len(stack)-1]
			if len(stack) == 0 {
				if out.Len() > 0 {
					out.WriteByte('\n')
				}
				WriteDocument(&out, top)
			}
		case goxml.CharData:
			if len(stack) == 0 {
				if strings.TrimSpace(string(t)) != "" {
					return nil, errorf("text outside of root element")
				}
				continue
			}
			top := stack[len(stack)-1]
			top.Text += string(t)
		}
	}
	if len(stack) > 0 {
		return nil, errorf("unclosed element <%s>", stack[len(stack)-1].Name)
	}
	return out.Bytes(), nil
}

func name(n goxml.Name) string {
	if n.Space == "" {
		return n.Local
	}
	return n.Space + ":" + n.Local
}

// charsetReader decodes the encodings an XML declaration commonly names
// without pulling in the Unicode tables of golang.org/x/text.
func charsetReader(charset string, r io.Reader) (io.Reader, error) {
	switch strings.ToLower(charset) {
	case "utf-8", "utf8", "us-ascii", "ascii":
		return r, nil
	case "iso-8859-1", "iso8859-1", "latin1", "latin-1":
		return &latin1Reader{r: bufio.NewReader(r)}, nil
	}
	return nil, fmt.Errorf("unsupported encoding %q", charset)
}

// latin1Reader re-encodes ISO-8859-1 bytes as UTF-8.
type latin1Reader struct {
	r *bufio.Reader
}

func (l *latin1Reader) Read(p []byte) (int, error) {
	n := 0
	for n+2 <= len(p) {
		b, err := l.r.ReadByte()
		if err != nil {
			if n > 0 {
				return n, nil
			}
			return 0, err
		}
		if b < 0x80 {
			p[n] = b
			n++
		} else {
			p[n] = 0xC0 | b>>6
			p[n+1] = 0x80 | b&0x3F
			n += 2
		}
	}
	return n, nil
}

// Element is a parsed element. The xml and html packages build this
// tree and write it with WriteDocument.
type Element struct {
	Name     string
	Attrs    []Attr
	Children []*Element
	Text     string // character data of the element itself, untrimmed
}

// Attr is an attribute of an Element.
type Attr struct {
	Name, Value string
}

// WriteDocument writes the root element as a JSON object with one key,
// the element's name.
func WriteDocument(b *bytes.Buffer, root *Element) {
	b.WriteByte('{')
	b.WriteString(engine.Quote(root.Name))
	b.WriteByte(':')
	root.WriteJSON(b)
	b.WriteByte('}')
}

// WriteJSON writes the element as a JSON value: null when it is empty, a
// string when it holds only text, and otherwise an object with the
// attributes as "@name", the children by name in order of first
// appearance (a repeated name becomes an array) and the text as "#text".
// A repeated attribute keeps its first value.
func (e *Element) WriteJSON(b *bytes.Buffer) {
	text := strings.TrimSpace(e.Text)
	if len(e.Attrs) == 0 && len(e.Children) == 0 {
		if text == "" {
			b.WriteString("null")
		} else {
			b.WriteString(engine.Quote(text))
		}
		return
	}

	b.WriteByte('{')
	first := true
	key := func(k string) {
		if !first {
			b.WriteByte(',')
		}
		first = false
		b.WriteString(engine.Quote(k))
		b.WriteByte(':')
	}

	seen := map[string]bool{}
	for _, a := range e.Attrs {
		if seen[a.Name] {
			continue
		}
		seen[a.Name] = true
		key("@" + a.Name)
		b.WriteString(engine.Quote(a.Value))
	}

	var names []string
	groups := map[string][]*Element{}
	for _, c := range e.Children {
		if _, ok := groups[c.Name]; !ok {
			names = append(names, c.Name)
		}
		groups[c.Name] = append(groups[c.Name], c)
	}
	for _, n := range names {
		key(n)
		group := groups[n]
		if len(group) == 1 {
			group[0].WriteJSON(b)
			continue
		}
		b.WriteByte('[')
		for i, c := range group {
			if i > 0 {
				b.WriteByte(',')
			}
			c.WriteJSON(b)
		}
		b.WriteByte(']')
	}

	if text != "" {
		key("#text")
		b.WriteString(engine.Quote(text))
	}
	b.WriteByte('}')
}
