package jsonx

import (
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/x/term"
	"github.com/mattn/go-runewidth"
)

func (p *JsonParser) errorSnippet(message string) error {
	termWidth, _, err := term.GetSize(os.Stdout.Fd())
	if err != nil {
		termWidth = 80
	}
	maxWidth := min(termWidth, 60)
	maxWidth -= 2
	maxWidth = max(maxWidth, 10)

	// As we already moved end pointer in next(), we need to move it back.
	p.end -= 1

	before, width := p.contextBefore(maxWidth / 2)
	after, _ := p.contextAfter(maxWidth - width)
	snippet := "  " + before + after
	snippet += "\n  " + strings.Repeat(".", width-1) + "^"

	return fmt.Errorf(
		"%s on line %d.\n\n%s\n\n",
		message,
		p.lineNumber,
		snippet,
	)
}

func (p *JsonParser) contextBefore(maxWidth int) (s string, width int) {
	pos := p.end + 1 // +1 to exclude the current character.
	data := p.data[:pos]
	for len(data) > 0 {
		r, size := utf8.DecodeLastRune(data)
		if r == '\n' {
			break
		}
		runeWidth := runewidth.RuneWidth(r)
		if width+runeWidth > maxWidth {
			break
		}
		width += runeWidth
		pos -= size
		data = data[:pos]
	}
	s = string(p.data[pos : p.end+1]) // +1 to include the current character.
	return
}

func (p *JsonParser) contextAfter(maxWidth int) (s string, width int) {
	pos := p.end + 1
	if pos >= len(p.data) {
		return
	}
	data := p.data[pos:]
	for len(data) > 0 {
		r, size := utf8.DecodeRune(data)
		if r == '\n' {
			break
		}
		runeWidth := runewidth.RuneWidth(r)
		if width+runeWidth > maxWidth {
			break
		}
		width += runeWidth
		pos += size
		data = data[size:]
	}
	s = string(p.data[p.end+1 : pos]) // +1 to exclude the current character.
	return
}
