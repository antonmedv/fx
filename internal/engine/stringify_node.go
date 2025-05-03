package engine

import (
	"strings"

	"github.com/antonmedv/fx/internal/jsonx"
	"github.com/antonmedv/fx/internal/theme"
)

// StringifyNode pretty prints a Node. Node must be the top (head),
// as everything will be printed.
func StringifyNode(n *jsonx.Node) string {
	var out strings.Builder

	it := n
	for it != nil {
		for ident := 0; ident < int(it.Depth); ident++ {
			out.WriteString("  ")
		}
		if it.Key != nil {
			out.WriteString(theme.CurrentTheme.Key(string(it.Key)))
			out.WriteString(theme.Colon)
		}
		if it.Value != nil {
			out.WriteString(theme.Value(it.Kind, false)(string(it.Value)))
		}
		if it.Comma {
			out.WriteString(theme.Comma)
		}
		if it.IsCollapsed() {
			it = it.Collapsed
		} else {
			it = it.Next
		}
		if it != nil {
			out.WriteByte('\n')
		}
	}

	return out.String()
}
