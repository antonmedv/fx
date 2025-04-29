package jsonx

import (
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
)

func DropWrapAll(n *Node) {
	for n != nil {
		if len(n.Value) > 0 && n.Value[0] == '"' {
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
		if len(n.Value) > 0 && n.Value[0] == '"' {
			n.dropChunks()
			lines, count := doWrap(n, termWidth)
			if count > 1 {
				n.Chunk = lines[0]
				for i := 1; i < count; i++ {
					child := &Node{
						directParent: n,
						Depth:        n.Depth,
						Chunk:        lines[i],
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

func doWrap(n *Node, termWidth int) ([][]byte, int) {
	lines := make([][]byte, 0, 1)
	width := int(n.Depth) * 2

	if n.Key != nil {
		for _, ch := range string(n.Key) {
			width += runewidth.RuneWidth(ch)
		}
		width += 2 // for ": "
	}

	linesCount := 0
	start, end := 0, 0
	b := n.Value

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
