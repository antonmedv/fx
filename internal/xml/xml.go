// Package xml converts XML to JSON.
package xml

import (
	"bytes"
	goxml "encoding/xml"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/antonmedv/fx/internal/charset"
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
// instructions and the DOCTYPE are dropped, except that entities the
// DOCTYPE declares inline are expanded. A byte order mark or the
// declaration's encoding selects the charset.
func ToJSON(in []byte) ([]byte, error) {
	enc, in := charset.Sniff(in)
	if enc != nil {
		var err error
		if in, err = charset.Decode(enc, in); err != nil {
			return nil, fmt.Errorf("xml: %w", err)
		}
	}
	d := goxml.NewDecoder(bytes.NewReader(in))
	d.CharsetReader = charset.Reader
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
			var syntaxErr *goxml.SyntaxError
			if errors.As(err, &syntaxErr) {
				return nil, errorf("%s", syntaxErr.Msg)
			}
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
			stack[len(stack)-1].AddText(string(t))
		case goxml.Directive:
			declareEntities(d, t)
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

// reEntity matches an internal general entity declaration in a DOCTYPE:
// <!ENTITY name "value">. Parameter (%) and external (SYSTEM, PUBLIC)
// entities do not match.
var reEntity = regexp.MustCompile(`<!ENTITY\s+([^\s%"'>]+)\s+(?:"([^"]*)"|'([^']*)')`)

// declareEntities registers the entities a DOCTYPE declares inline, so
// the decoder expands references to them in text and attributes.
func declareEntities(d *goxml.Decoder, directive []byte) {
	for _, m := range reEntity.FindAllSubmatch(directive, -1) {
		if d.Entity == nil {
			d.Entity = map[string]string{}
		}
		value := m[2]
		if value == nil {
			value = m[3]
		}
		d.Entity[string(m[1])] = string(value)
	}
}

// Element is a parsed element. The xml and html packages build this
// tree and write it with WriteDocument.
type Element struct {
	Name     string
	Attrs    []Attr
	Children []*Element
	text     strings.Builder
}

// Attr is an attribute of an Element.
type Attr struct {
	Name, Value string
}

// AddText appends character data of the element itself.
func (e *Element) AddText(s string) {
	e.text.WriteString(s)
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
	text := strings.TrimSpace(e.text.String())
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

	for _, g := range e.groups() {
		key(g.name)
		if len(g.elems) == 1 {
			g.elems[0].WriteJSON(b)
			continue
		}
		b.WriteByte('[')
		for i, c := range g.elems {
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

type group struct {
	name  string
	elems []*Element
}

// groups collects the children by name in order of first appearance. A
// map is only worth its allocation for elements with many children.
func (e *Element) groups() []group {
	var groups []group
	var index map[string]int
	if len(e.Children) > 8 {
		index = make(map[string]int, len(e.Children))
	}
	for _, c := range e.Children {
		i := -1
		if index != nil {
			if j, ok := index[c.Name]; ok {
				i = j
			}
		} else {
			for j := range groups {
				if groups[j].name == c.Name {
					i = j
					break
				}
			}
		}
		if i < 0 {
			groups = append(groups, group{name: c.Name})
			i = len(groups) - 1
			if index != nil {
				index[c.Name] = i
			}
		}
		groups[i].elems = append(groups[i].elems, c)
	}
	return groups
}
