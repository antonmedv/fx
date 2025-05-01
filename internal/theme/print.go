package theme

import (
	"strings"

	"github.com/antonmedv/fx/internal/jsonx"
)

func PrintFullJson(n *jsonx.Node) string {
	var out strings.Builder

	it := n
	for it != nil {
		for ident := 0; ident < int(it.Depth); ident++ {
			out.WriteString("  ")
		}
		if it.Key != nil {
			out.WriteString(CurrentTheme.Key(string(it.Key)))
			out.WriteString(Colon)
		}
		if it.Value != nil {
			out.WriteString(Value(it.Kind, false)(string(it.Value)))
		}
		if it.Comma {
			out.WriteString(Comma)
		}
		out.WriteByte('\n')
		if it.IsCollapsed() {
			it = it.Collapsed
		} else {
			it = it.Next
		}
	}

	return out.String()
}
