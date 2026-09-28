package html

import (
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
		{"document",
			`<!DOCTYPE html><html lang="en"><head><title>T</title></head><body><p class="x">hi</p></body></html>`,
			`{"html":{"@lang":"en","head":{"title":"T"},"body":{"p":{"@class":"x","#text":"hi"}}}}`},
		{"fragment goes under body", `<div>hi</div>`, `{"html":{"head":null,"body":{"div":"hi"}}}`},
		{"names lowercased", `<DIV CLASS="x">hi</DIV>`, `{"html":{"head":null,"body":{"div":{"@class":"x","#text":"hi"}}}}`},
		{"void element", `<p>a<br>b</p>`, `{"html":{"head":null,"body":{"p":{"br":null,"#text":"ab"}}}}`},
		{"unclosed elements", `<ul><li>a<li>b</ul>`, `{"html":{"head":null,"body":{"ul":{"li":["a","b"]}}}}`},
		{"implied tbody", `<table><tr><td>1</td></tr></table>`, `{"html":{"head":null,"body":{"table":{"tbody":{"tr":{"td":"1"}}}}}}`},
		{"unquoted and boolean attributes", `<input disabled type=text>`, `{"html":{"head":null,"body":{"input":{"@disabled":"","@type":"text"}}}}`},
		{"repeated attribute keeps first", `<a x="1" x="2"></a>`, `{"html":{"head":null,"body":{"a":{"@x":"1"}}}}`},
		{"entities", `<p>&lt;&amp;&copy;&#65;&#x42;</p>`, `{"html":{"head":null,"body":{"p":"<&©AB"}}}`},
		{"script keeps raw text", `<body><script>if (a<b && c) {}</script></body>`, `{"html":{"head":null,"body":{"script":"if (a<b && c) {}"}}}`},
		{"title in head", `<title>a &amp; <i>b</i></title>`, `{"html":{"head":{"title":"a & <i>b</i>"},"body":null}}`},
		{"comment dropped", `<p>a<!-- c -->b</p>`, `{"html":{"head":null,"body":{"p":"ab"}}}`},
		{"svg attributes", `<svg viewBox="0 0 1 1"><a xlink:href="u"/></svg>`, `{"html":{"head":null,"body":{"svg":{"@viewBox":"0 0 1 1","a":{"@xlink:href":"u"}}}}}`},
		{"mixed content", `<p>Hello <b>w</b>!</p>`, `{"html":{"head":null,"body":{"p":{"b":"w","#text":"Hello !"}}}}`},
		{"whitespace between elements", "<div>\n  <p>x</p>\n  <p>y</p>\n</div>", `{"html":{"head":null,"body":{"div":{"p":["x","y"]}}}}`},
		{"unicode", `<p>привет 😀</p>`, `{"html":{"head":null,"body":{"p":"привет 😀"}}}`},
		{"utf-8 bom stripped", "\xEF\xBB\xBF<p>x</p>", `{"html":{"head":null,"body":{"p":"x"}}}`},
		{"utf-16le bom", "\xFF\xFE<\x00p\x00>\x00\xe9\x00<\x00/\x00p\x00>\x00", `{"html":{"head":null,"body":{"p":"é"}}}`},
		{"meta charset", `<meta charset="windows-1252"><p>` + "\x93q\x94" + `</p>`, `{"html":{"head":{"meta":{"@charset":"windows-1252"}},"body":{"p":"“q”"}}}`},
		{"meta http-equiv", `<meta http-equiv="Content-Type" content="text/html; charset=ISO-8859-1"><p>` + "\xe9" + `</p>`, `{"html":{"head":{"meta":{"@http-equiv":"Content-Type","@content":"text/html; charset=ISO-8859-1"}},"body":{"p":"é"}}}`},
		{"unknown meta charset read as utf-8", `<meta charset="shift_jis"><p>é</p>`, `{"html":{"head":{"meta":{"@charset":"shift_jis"}},"body":{"p":"é"}}}`},
		{"meta charset beyond 1024 bytes ignored", `<!--` + strings.Repeat("x", 1024) + `--><meta charset="windows-1252"><p>é</p>`, `{"html":{"head":{"meta":{"@charset":"windows-1252"}},"body":{"p":"é"}}}`},
		{"empty", ``, ``},
		{"blank", " \n\t", ``},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToJSON([]byte(tt.in))
			require.NoError(t, err)
			require.Equal(t, tt.want, string(got))
		})
	}
}
