package pretty

import (
	"strings"

	"github.com/antonmedv/fx/internal/jsonx"
	"github.com/antonmedv/fx/internal/theme"
)

// Print pretty prints a Node. Node must be the top (head),
// as everything will be printed.
func Print(n *jsonx.Node, withInline bool) string {
	var out strings.Builder

	it := n
	for it != nil {
		if withInline {
			if isTable(it) {
				it = table(&out, it)
				continue
			}
			if isInlineable(it) {
				it = inline(&out, it)
				continue
			}
		}
		printIdent(&out, it)
		printKey(&out, it)
		printValue(&out, it)
		it = next(it)
		if it != nil {
			out.WriteByte('\n')
		}
	}

	return out.String()
}

func table(out *strings.Builder, n *jsonx.Node) *jsonx.Node {
	printIdent(out, n)
	printKey(out, n)
	printValue(out, n)
	out.WriteByte('\n')

	it := next(n)
	end := n.End
	for it != nil && it != end {
		it = inline(out, it)
	}

	printIdent(out, end)
	printValue(out, end)

	it = next(it)
	if it != nil {
		out.WriteByte('\n')
	}
	return it
}

func inline(out *strings.Builder, n *jsonx.Node) *jsonx.Node {
	printIdent(out, n)
	printSpace := false
	it := n
	end := afterEnd(n)
	for it != nil && it != end {
		if printSpace {
			out.WriteString(" ")
		} else {
			printSpace = true
		}
		printKey(out, it)
		printValue(out, it)
		it = next(it)
	}

	out.WriteByte('\n')
	return it
}

func printIdent(out *strings.Builder, n *jsonx.Node) {
	for ident := 0; ident < int(n.Depth); ident++ {
		out.WriteString("  ")
	}
}

func printKey(out *strings.Builder, n *jsonx.Node) {
	if n.Key != "" {
		out.WriteString(theme.CurrentTheme.Key(n.Key))
		out.WriteString(theme.Colon)
	}
}

func printValue(out *strings.Builder, n *jsonx.Node) {
	if n.Value != "" {
		out.WriteString(theme.Value(n.Kind, false)(n.Value))
	}
	if n.Comma {
		out.WriteString(theme.Comma)
	}
}

func next(n *jsonx.Node) *jsonx.Node {
	if n.IsCollapsed() {
		return n.Collapsed
	} else {
		return n.Next
	}
}

func afterEnd(n *jsonx.Node) *jsonx.Node {
	if n.End != nil {
		return n.End.Next
	}
	return n.Next
}
