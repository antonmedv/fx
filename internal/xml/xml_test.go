package xml

import (
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
		{"namespace prefixes kept", `<s:e xmlns:s="u" s:k="1"><s:b/></s:e>`, `{"s:e":{"@xmlns:s":"u","@s:k":"1","s:b":null}}`},
		{"default namespace", `<e xmlns="u"><b/></e>`, `{"e":{"@xmlns":"u","b":null}}`},
		{"unicode", `<a>привет 😀</a>`, `{"a":"привет 😀"}`},
		{"latin1", "<?xml version=\"1.0\" encoding=\"ISO-8859-1\"?><a>\xe9\xe0</a>", `{"a":"éà"}`},
		{"latin1 case", "<?xml version=\"1.0\" encoding=\"latin1\"?><a>\xe9</a>", `{"a":"é"}`},
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
		{"unclosed nested", "<a>\n<b>\n</a>", "line 3"},
		{"mismatch", `<a></b>`, "xml: line 1, column 8: element <a> closed by </b>"},
		{"stray end tag", `</a>`, "xml: line 1, column 5: unexpected end tag </a>"},
		{"text outside root", `hi`, "text outside of root element"},
		{"text after root", `<a/>x`, "text outside of root element"},
		{"unknown entity", `<a>&nbsp;</a>`, "invalid character entity"},
		{"unquoted attribute", `<a x=1/>`, "unquoted or missing attribute value"},
		{"bare ampersand", `<a>a & b</a>`, "invalid character entity"},
		{"unsupported encoding", `<?xml version="1.0" encoding="windows-1252"?><a/>`, `unsupported encoding "windows-1252"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ToJSON([]byte(tt.in))
			require.ErrorContains(t, err, tt.err)
		})
	}
}

func TestLatin1Reader(t *testing.T) {
	// Multi-byte output must not split across Read calls with a 3-byte buffer.
	in := "<?xml version=\"1.0\" encoding=\"ISO-8859-1\"?><a>" + "\xe9\xe9\xe9\xe9\xe9" + "</a>"
	got, err := ToJSON([]byte(in))
	require.NoError(t, err)
	require.Equal(t, `{"a":"ééééé"}`, string(got))
}
