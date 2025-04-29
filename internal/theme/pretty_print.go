package theme

import (
	"strings"

	"github.com/antonmedv/fx/internal/jsonx"
)

func PrettyPrint(n *jsonx.Node) string {
	var out strings.Builder

	it := n
	for it != nil {
		for ident := 0; ident < int(it.Depth); ident++ {
			out.WriteString("  ")
		}
		if it.Key != nil {
			out.Write(CurrentTheme.Key(it.Key))
			out.Write(Colon)
		}
		if it.Value != nil {
			out.Write(Value(it.Kind, false)(it.Value))
		}
		if it.Comma {
			out.Write(Comma)
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
