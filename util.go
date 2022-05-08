package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func clamp(v, low, high int) int {
	if high < low {
		low, high = high, low
	}
	return min(high, max(low, v))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func width(s string) int {
	return lipgloss.Width(s)
}

func accessor(path string, to interface{}) string {
	return fmt.Sprintf("%v[%v]", path, to)
}

func toLowerNumber(s string) string {
	var out strings.Builder
	for _, r := range s {
		switch {
		case '0' <= r && r <= '9':
			out.WriteRune('\u2080' + (r - '\u0030'))
		default:
			out.WriteRune(r)
		}
	}
	return out.String()
}
