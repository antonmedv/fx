package jsonx

import (
	"unicode/utf8"

	"github.com/mattn/go-runewidth"

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
		for _, ch := range n.Key {
			width += runewidth.RuneWidth(ch)
		}
		width += 2 // for ": "
	}

	linesCount := 0
	start, end := 0, 0
	b := []byte(n.Value)

	for len(b) > 0 {
		r, size := utf8.DecodeRune(b)
		w := runewidth.RuneWidth(r)
		if width+w > termWidth {
			lines = append(lines, n.Value[start:end])
			start = end
			width = int(n.Depth) * 2
			linesCount++
		}
		width += w
		end += size
		b = b[size:]
	}

	if start < end {
		lines = append(lines, n.Value[start:])
		linesCount++
	}

	return lines, linesCount
}
