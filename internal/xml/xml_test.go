package xml

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestToJSON(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"text", `<a>x</a>`, `{"a":"x"}`},
		{"empty self-closing", `<a/>`, `{"a":null}`},
		{"empty pair", `<a></a>`, `{"a":null}`},
		{"whitespace only", "<a>\n  \n</a>", `{"a":null}`},
		{"text trimmed", "<a>\n  x y \n</a>", `{"a":"x y"}`},
		{"attributes", `<a id="1" k='v'/>`, `{"a":{"@id":"1","@k":"v"}}`},
		{"attributes and text", `<a id="1">x</a>`, `{"a":{"@id":"1","#text":"x"}}`},
		{"repeated attribute keeps first", `<a x="1" x="2"/>`, `{"a":{"@x":"1"}}`},
		{"children keep order", `<r><zebra/><alpha>1</alpha></r>`, `{"r":{"zebra":null,"alpha":"1"}}`},
		{"repeated children", `<r><i>1</i><i>2</i></r>`, `{"r":{"i":["1","2"]}}`},
		{"interleaved children", `<r><i>1</i><j/><i>2</i></r>`, `{"r":{"i":["1","2"],"j":null}}`},
		{"attribute and child with same name", `<a b="1"><b>2</b></a>`, `{"a":{"@b":"1","b":"2"}}`},
		{"mixed content", `<p>Hello <b>w</b>!</p>`, `{"p":{"b":"w","#text":"Hello !"}}`},
		{"nested", `<a><b><c>1</c></b></a>`, `{"a":{"b":{"c":"1"}}}`},
		{"whitespace between elements", "<r>\n  <a>x</a>\n  <b/>\n</r>", `{"r":{"a":"x","b":null}}`},
		{"entities", `<a>&lt;&gt;&amp;&quot;&apos;&#65;&#x42;</a>`, `{"a":"<>&\"'AB"}`},
		{"entities in attribute", `<a x="&lt;&amp;"/>`, `{"a":{"@x":"<&"}}`},
		{"quotes in text", `<a>say "hi"</a>`, `{"a":"say \"hi\""}`},
		{"cdata", `<a><![CDATA[<b>&</b>]]></a>`, `{"a":"<b>&</b>"}`},
		{"declaration", `<?xml version="1.0"?><a/>`, `{"a":null}`},
		{"bom", "\xEF\xBB\xBF<a/>", `{"a":null}`},
		{"comment", `<a><!-- c --><b/></a>`, `{"a":{"b":null}}`},
		{"comment only text", `<a>x<!-- c -->y</a>`, `{"a":"xy"}`},
		{"processing instruction", `<?pi x?><a><?pi y?></a>`, `{"a":null}`},
		{"doctype", `<!DOCTYPE a><a/>`, `{"a":null}`},
		{"doctype entities", `<!DOCTYPE a [<!ENTITY foo "bar"> <!ENTITY q 'x "y"'>]><a k="&foo;">&foo;&q;</a>`, `{"a":{"@k":"bar","#text":"barx \"y\""}}`},
		{"doctype illustrator svg", `<!DOCTYPE svg PUBLIC "-//W3C//DTD SVG 1.1//EN" "http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd" [
	<!ENTITY ns_svg "http://www.w3.org/2000/svg">
]><svg xmlns="&ns_svg;"/>`, `{"svg":{"@xmlns":"http://www.w3.org/2000/svg"}}`},
		{"namespace prefixes kept", `<s:e xmlns:s="u" s:k="1"><s:b/></s:e>`, `{"s:e":{"@xmlns:s":"u","@s:k":"1","s:b":null}}`},
		{"default namespace", `<e xmlns="u"><b/></e>`, `{"e":{"@xmlns":"u","b":null}}`},
		{"unicode", `<a>привет 😀</a>`, `{"a":"привет 😀"}`},
		{"latin1", "<?xml version=\"1.0\" encoding=\"ISO-8859-1\"?><a>\xe9\xe0</a>", `{"a":"éà"}`},
		{"latin1 case", "<?xml version=\"1.0\" encoding=\"latin1\"?><a>\xe9</a>", `{"a":"é"}`},
		{"windows-1252", "<?xml version=\"1.0\" encoding=\"windows-1252\"?><a>\x93q\x94</a>", `{"a":"“q”"}`},
		{"utf-16le bom", "\xFF\xFE<\x00a\x00>\x00\xe9\x00<\x00/\x00a\x00>\x00", `{"a":"é"}`},
		{"utf-16be bom with declaration", "\xFE\xFF\x00<\x00?\x00x\x00m\x00l\x00 \x00v\x00e\x00r\x00s\x00i\x00o\x00n\x00=\x00\"\x001\x00.\x000\x00\"\x00 \x00e\x00n\x00c\x00o\x00d\x00i\x00n\x00g\x00=\x00\"\x00U\x00T\x00F\x00-\x001\x006\x00\"\x00?\x00>\x00<\x00a\x00/\x00>", `{"a":null}`},
		{"utf-16le no bom", "<\x00?\x00x\x00m\x00l\x00 \x00v\x00e\x00r\x00s\x00i\x00o\x00n\x00=\x00\"\x001\x00.\x000\x00\"\x00?\x00>\x00<\x00a\x00/\x00>\x00", `{"a":null}`},
		{"utf-8 declared", `<?xml version="1.0" encoding="UTF-8"?><a>é</a>`, `{"a":"é"}`},
		{"numbers stay strings", `<a>1</a>`, `{"a":"1"}`},
		{"multiple roots", "<a/>\n<b>1</b>", "{\"a\":null}\n{\"b\":\"1\"}"},
		{"empty", ``, ``},
		{"whitespace and comment only", "\n<!-- c -->\n", ``},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToJSON([]byte(tt.in))
			require.NoError(t, err)
			require.Equal(t, tt.want, string(got))
		})
	}
}

func TestToJSON_Invalid(t *testing.T) {
	tests := []struct {
		name string
		in   string
		err  string
	}{
		{"unclosed", `<a>`, "xml: line 1, column 4: unclosed element <a>"},
		{"unclosed root only", `<a><b/>`, "unclosed element <a>"},
		{"unclosed nested", "<a>\n<b>\n</a>", "line 3"},
		{"mismatch", `<a></b>`, "xml: line 1, column 8: element <a> closed by </b>"},
		{"stray end tag", `</a>`, "xml: line 1, column 5: unexpected end tag </a>"},
		{"text outside root", `hi`, "text outside of root element"},
		{"text after root", `<a/>x`, "text outside of root element"},
		{"unknown entity", `<a>&nbsp;</a>`, "xml: line 1, column 10: invalid character entity &nbsp;"},
		{"parameter entity not declared", `<!DOCTYPE a [<!ENTITY % p "v">]><a>&p;</a>`, "invalid character entity &p;"},
		{"external entity not declared", `<!DOCTYPE a [<!ENTITY e SYSTEM "f">]><a>&e;</a>`, "invalid character entity &e;"},
		{"unquoted attribute", `<a x=1/>`, "xml: line 1, column 7: unquoted or missing attribute value in element"},
		{"bare ampersand", `<a>a & b</a>`, "xml: line 1, column 7: invalid character entity & (no semicolon)"},
		{"decoder error on later line", "<a>\n<b>&nbsp;</b></a>", "xml: line 2, column 10: invalid character entity &nbsp;"},
		{"unsupported encoding", `<?xml version="1.0" encoding="shift_jis"?><a/>`, `unsupported encoding "shift_jis"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ToJSON([]byte(tt.in))
			require.ErrorContains(t, err, tt.err)
		})
	}
}

func TestToJSON_LargeMixedContent(t *testing.T) {
	// Text accumulation must stay linear in the number of siblings.
	const n = 100_000
	var in strings.Builder
	in.WriteString("<p>")
	for range n {
		in.WriteString("xxxxxxxxxx<br/>")
	}
	in.WriteString("</p>")
	got, err := ToJSON([]byte(in.String()))
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(string(got), `{"p":{"br":[null,null,`), string(got[:40]))
	require.True(t, strings.HasSuffix(string(got), `],"#text":"`+strings.Repeat("x", 10*n)+`"}}`))
}

func TestGroups(t *testing.T) {
	// Above 8 children a map indexes the groups; both paths must agree.
	for _, n := range []int{3, 8, 9, 50} {
		var in strings.Builder
		in.WriteString("<r>")
		for i := range n {
			fmt.Fprintf(&in, "<k%d>%d</k%d><dup/>", i%3, i, i%3)
		}
		in.WriteString("</r>")
		got, err := ToJSON([]byte(in.String()))
		require.NoError(t, err, "n=%d", n)
		var v map[string]map[string]any
		require.NoError(t, json.Unmarshal(got, &v))
		r := v["r"]
		require.Equal(t, []string{"k0", "dup", "k1", "k2"}[:min(n, 3)+1], keysInOrder(t, got), "n=%d", n)
		require.Len(t, r["dup"], n, "n=%d", n)
	}
}

// keysInOrder returns the keys of the root object in document order.
func keysInOrder(t *testing.T, doc []byte) []string {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(doc))
	var keys []string
	depth := 0
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		switch x := tok.(type) {
		case json.Delim:
			switch x {
			case '{', '[':
				depth++
			case '}', ']':
				depth--
			}
		case string:
			if depth == 2 && !dec.More() {
				continue
			}
			if depth == 2 {
				keys = append(keys, x)
				// skip the value
				var v any
				require.NoError(t, dec.Decode(&v))
			}
		}
	}
	return keys
}
