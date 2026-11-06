package jsonx_test

import (
	"strings"
	"testing"

	"github.com/mattn/go-runewidth"
	"github.com/rivo/uniseg"
	"github.com/stretchr/testify/require"

	"github.com/antonmedv/fx/internal/jsonx"
)

const (
	family   = "\U0001F468‍\U0001F469‍\U0001F467‍\U0001F466" // 4 runes joined by ZWJ, one glyph
	flag     = "\U0001F1EF\U0001F1F5"                        // regional indicator pair
	skinTone = "\U0001F44D\U0001F3FD"                        // emoji plus modifier
)

// wrapLines returns the lines Wrap produced for a string node: its own first
// chunk followed by every chunk child. A node that fitted is one line.
func wrapLines(t *testing.T, value string, termWidth int) (*jsonx.Node, []string) {
	t.Helper()
	node, err := jsonx.Parse([]byte(`"` + value + `"`))
	require.NoError(t, err)
	jsonx.Wrap(node, termWidth)
	if node.Chunk == "" {
		return node, []string{node.Value}
	}
	lines := []string{node.Chunk}
	for it := node.Next; it != nil && it.Parent == node; it = it.Next {
		lines = append(lines, it.Chunk)
	}
	return node, lines
}

// clusterBoundaries lists the byte offsets where a grapheme cluster may begin.
func clusterBoundaries(s string) map[int]bool {
	at := map[int]bool{0: true}
	g := uniseg.NewGraphemes(s)
	for g.Next() {
		_, to := g.Positions()
		at[to] = true
	}
	return at
}

// TestWrap_MeasuresGlyphsNotRunes checks that a value is measured the way a
// terminal draws it. Six family emoji occupy twelve columns, so with the
// quotes they fit a twenty column terminal; counted rune by rune they come to
// forty eight and the line breaks three times over.
func TestWrap_MeasuresGlyphsNotRunes(t *testing.T) {
	value := strings.Repeat(family, 6)
	require.Equal(t, 12, runewidth.StringWidth(value))

	node, lines := wrapLines(t, value, 20)
	require.Len(t, lines, 1, "a value narrower than the terminal must not wrap")
	require.Equal(t, node.Value, lines[0])
}

// TestWrap_KeepsGlyphsWhole checks that a break never lands between the runes
// of one glyph, which would leave a dangling zero-width joiner at the end of a
// line and orphan the rest of the glyph at the start of the next.
func TestWrap_KeepsGlyphsWhole(t *testing.T) {
	for _, tc := range []struct {
		name  string
		glyph string
	}{
		{"zwj family", family},
		{"regional indicator flag", flag},
		{"emoji with skin tone", skinTone},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value := strings.Repeat(tc.glyph, 8)
			for _, termWidth := range []int{4, 6, 9, 13} {
				node, lines := wrapLines(t, value, termWidth)
				require.Equal(t, node.Value, strings.Join(lines, ""),
					"the lines must rejoin into the value unchanged")

				at := clusterBoundaries(node.Value)
				off := 0
				for i, line := range lines[:len(lines)-1] {
					off += len(line)
					require.Truef(t, at[off],
						"termWidth %d: the break after line %d (byte %d) falls inside a glyph",
						termWidth, i, off)
				}
			}
		})
	}
}

// TestWrap_LinesFitTheTerminal checks the property wrapping exists to provide:
// no line it produces is wider than the terminal it was given.
func TestWrap_LinesFitTheTerminal(t *testing.T) {
	value := strings.Repeat(family+"ab"+flag+"cd", 4)
	for _, termWidth := range []int{6, 10, 17, 24} {
		_, lines := wrapLines(t, value, termWidth)
		for i, line := range lines {
			require.LessOrEqualf(t, runewidth.StringWidth(line), termWidth,
				"termWidth %d: line %d is wider than the terminal", termWidth, i)
		}
	}
}
