package pretty

import (
	"strings"

	"github.com/antonmedv/fx/internal/jsonx"
	"github.com/antonmedv/fx/internal/theme"
)

// Print pretty prints a Node. Node must be the top (head),
// as everything will be printed.
func Print(n *jsonx.Node) string {
	var out strings.Builder

	it := n
	for it != nil {
		if isInlineable(it) {
			out.WriteString(inline(it))
			it = it.End.Next
			continue
		}
		for ident := 0; ident < int(it.Depth); ident++ {
			out.WriteString("  ")
		}
		if it.Key != "" {
			out.WriteString(theme.CurrentTheme.Key(it.Key))
			out.WriteString(theme.Colon)
		}
		if it.Value != "" {
			out.WriteString(theme.Value(it.Kind, false)(it.Value))
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

func inline(n *jsonx.Node) string {
	var out strings.Builder

	for ident := 0; ident < int(n.Depth); ident++ {
		out.WriteString("  ")
	}

	printSpace := false

	it := n
	end := n.End.Next
	for it != nil && it != end {
		if printSpace {
			out.WriteString(" ")
		} else {
			printSpace = true
		}
		if it.Key != "" {
			out.WriteString(theme.CurrentTheme.Key(it.Key))
			out.WriteString(theme.Colon)
		}
		if it.Value != "" {
			out.WriteString(theme.Value(it.Kind, false)(it.Value))
		}
		if it.Comma {
			out.WriteString(theme.Comma)
		}
		if it.IsCollapsed() {
			it = it.Collapsed
		} else {
			it = it.Next
		}
	}

	out.WriteByte('\n')

	return out.String()
}
