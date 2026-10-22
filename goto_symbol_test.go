package main

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"
	"github.com/stretchr/testify/require"

	"github.com/antonmedv/fx/internal/fuzzy"
	"github.com/antonmedv/fx/internal/theme"
)

func TestGotoSymbolUnicodeRendering(t *testing.T) {
	saved := theme.CurrentTheme
	t.Cleanup(func() { theme.CurrentTheme = saved })
	theme.CurrentTheme.StatusBar = func(s string) string { return s }
	theme.CurrentTheme.Search = func(s string) string { return "\x1b[31m" + s + "\x1b[0m" }

	for _, tc := range []struct {
		text, pattern string
	}{
		{"name", "m"},
		{"名前", "前"},
		{"éclair", "c"},
		{"🍎apple", "a"},
	} {
		t.Run(tc.text, func(t *testing.T) {
			m := newQueryModel(t, `{"name": 1}`)
			m.gotoSymbolInput = newInput()
			m.gotoSymbolInput.Focus()
			m.fuzzyMatch = fuzzy.Find([]rune(tc.pattern), []string{tc.text})
			require.NotNil(t, m.fuzzyMatch)

			lines := strings.Split(view(m), "\n")
			status := lines[len(lines)-2]
			require.True(t, utf8.ValidString(status))
			require.Equal(t, tc.text, strings.TrimRight(ansi.Strip(status), " "))
			require.Contains(t, status, "\x1b[31m"+tc.pattern+"\x1b[0m")
			require.Equal(t, m.termWidth, runewidth.StringWidth(ansi.Strip(status)))
		})
	}
}

func TestGotoSymbolUnicodeKey(t *testing.T) {
	m := newQueryModel(t, `{"名前": 1}`)
	m.gotoSymbolInput = newInput()
	pressKey(m, press("@"))
	typeKeys(m, "前")
	require.NotNil(t, m.fuzzyMatch)

	lines := strings.Split(view(m), "\n")
	status := ansi.Strip(lines[len(lines)-2])
	require.Contains(t, status, "名前")
	require.Equal(t, m.termWidth, runewidth.StringWidth(status))
}
