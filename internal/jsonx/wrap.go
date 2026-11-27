package jsonx

import (
	"github.com/mattn/go-runewidth"
	"github.com/rivo/uniseg"

	"github.com/antonmedv/fx/internal/ident"
)

func DropWrapAll(n *Node) {
	for n != nil {
		if n.Kind == String || n.Kind == Err {
			n.dropChunks()
		}
		if n.IsCollapsed() {
			n = n.Collapsed
		} else {
			n = n.Next
		}
	}
}

func Wrap(n *Node, termWidth int) {
	if termWidth <= 0 {
		return
	}
	for n != nil {
		if n.Kind == String || n.Kind == Err {
			n.dropChunks()
			lines, count := doWrap(n, termWidth)
			if count > 1 {
				n.Chunk = lines[0]
				for i := 1; i < count; i++ {
					child := &Node{
						Kind:   n.Kind,
						Parent: n,
						Depth:  n.Depth,
						Chunk:  lines[i],
					}
					if n.Comma && i == count-1 {
						child.Comma = true
					}
					n.insertChunk(child)
				}
			}
		}
		if n.IsCollapsed() {
			n = n.Collapsed
		} else {
			n = n.Next
		}
	}
}

func doWrap(n *Node, termWidth int) ([]string, int) {
	lines := make([]string, 0, 1)
	width := int(n.Depth) * ident.IdentWidth

	if n.Key != "" {
		width += runewidth.StringWidth(n.Key)
		width += 2 // for ": "
	}

	linesCount := 0
	start, end := 0, 0
	rest := n.Value
	state := -1

	// Advance a grapheme cluster at a time rather than a rune at a time. A
	// flag, an emoji carrying a skin tone and a family joined by zero-width
	// joiners are each several runes and one glyph: measured rune by rune a
	// family counts as eight columns instead of two, so the line breaks long
	// before it is full, and the break can land between the runes and tear
	// the glyph in half.
	for len(rest) > 0 {
		var cluster string
		cluster, rest, _, state = uniseg.FirstGraphemeClusterInString(rest, state)
		w := runewidth.StringWidth(cluster)
		if width+w > termWidth {
			lines = append(lines, n.Value[start:end])
			start = end
			width = int(n.Depth) * 2
			linesCount++
		}
		width += w
		end += len(cluster)
	}

	if start < end {
		lines = append(lines, n.Value[start:])
		linesCount++
	}

	return lines, linesCount
}
