package jsonx

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// readAll reads input the way the viewer's loader does: values, and the
// text that is not JSON, recovered line by line.
func readAll(t *testing.T, input string) []string {
	t.Helper()
	p := NewJsonParser(strings.NewReader(input), false)
	var out []string
	for range 100 {
		node, err := p.Parse()
		if err == io.EOF {
			return out
		}
		if err != nil {
			node = p.Recover()
			require.NotNil(t, node)
			out = append(out, "text: "+node.Value)
			for it := node.Next; it != nil; it = it.Next {
				out = append(out, "text: "+it.Value)
			}
			continue
		}
		out = append(out, node.String())
	}
	t.Fatal("reading did not end")
	return nil
}

func TestRecover_KeepsTheWholeText(t *testing.T) {
	tests := []struct {
		name, input string
		want        []string
	}{
		{"word like Infinity", "{\"a\":1}\nINFO server started\n{\"b\":2}",
			[]string{`{"a":1}`, "text: INFO server started", `{"b":2}`}},
		{"word like infinity", "{\"a\":1}\ninfinite loop",
			[]string{`{"a":1}`, "text: infinite loop"}},
		{"word like null", "{\"a\":1}\nnothing here",
			[]string{`{"a":1}`, "text: nothing here"}},
		{"word like undefined", "{\"a\":1}\nunderflow",
			[]string{`{"a":1}`, "text: underflow"}},
		{"glued text", `{"a":1}x`,
			[]string{`{"a":1}`, "text: x"}},
		{"no blank line before", "{\"a\":1}\nError: boom",
			[]string{`{"a":1}`, "text: Error: boom"}},
		{"broken value", `{"b": tru}`,
			[]string{`text: {"b": tru}`}},
		{"bad comment", "{\"a\":1}\n/* unclosed",
			[]string{`{"a":1}`, "text: /* unclosed"}},
		{"http headers", "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\n\r\n{\"a\": 1}",
			[]string{"text: HTTP/1.1 200 OK", "text: Content-Type: application/json", "text: ", `{"a":1}`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, readAll(t, tt.input))
		})
	}
}
