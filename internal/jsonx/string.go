package jsonx

import (
	"strings"

	"github.com/antonmedv/fx/internal/theme"
)

func (n *Node) String() string {
	var out strings.Builder

	it := n
	for it != nil {
		if it.Key != nil {
			out.Write(it.Key)
			out.WriteByte(':')
		}
		if it.Value != nil {
			out.Write(it.Value)
		}
		if it.Comma {
			out.WriteByte(',')
		}
		if it.IsCollapsed() {
			it = it.Collapsed
		} else {
			it = it.Next
		}
	}

	return out.String()
}

func (n *Node) PrettyPrint() string {
	var out strings.Builder

	it := n
	for it != nil {
		for ident := 0; ident < int(it.Depth); ident++ {
			out.WriteString("  ")
		}
		if it.Key != nil {
			out.Write(theme.CurrentTheme.Key(it.Key))
			out.Write(theme.Colon)
		}
		if it.Value != nil {
			out.Write(theme.Value(it.Value, false, false)(it.Value))
		}
		if it.Comma {
			out.Write(theme.Comma)
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
