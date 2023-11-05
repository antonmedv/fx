package model

import (
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
)

func dropWrapAll(n *node) {
	for n != nil {
		if n.value != nil && n.value[0] == '"' {
			n.dropChunks()
		}
		if n.isCollapsed() {
			n = n.collapsed
		} else {
			n = n.next
		}
	}
}

func wrapAll(n *node, termWidth int) {
	if termWidth <= 0 {
		return
	}
	for n != nil {
		if n.value != nil && n.value[0] == '"' {
			n.dropChunks()
			lines, count := doWrap(n, termWidth)
			if count > 1 {
				n.chunk = lines[0]
				for i := 1; i < count; i++ {
					child := &node{
						directParent: n,
						depth:        n.depth,
						chunk:        lines[i],
					}
					if n.comma && i == count-1 {
						child.comma = true
					}
					n.insertChunk(child)
				}
			}
		}
		if n.isCollapsed() {
			n = n.collapsed
		} else {
			n = n.next
		}
	}
}

func doWrap(n *node, termWidth int) ([][]byte, int) {
	lines := make([][]byte, 0, 1)
	width := int(n.depth) * 2

	if n.key != nil {
		for _, ch := range string(n.key) {
			width += runewidth.RuneWidth(ch)
		}
		width += 2 // for ": "
	}

	linesCount := 0
	start, end := 0, 0
	b := n.value

	for len(b) > 0 {
		r, size := utf8.DecodeRune(b)
		w := runewidth.RuneWidth(r)
		if width+w > termWidth {
			lines = append(lines, n.value[start:end])
			start = end
			width = int(n.depth) * 2
			linesCount++
		}
		width += w
		end += size
		b = b[size:]
	}

	if start < end {
		lines = append(lines, n.value[start:])
		linesCount++
	}

	return lines, linesCount
}
