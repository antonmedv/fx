package main

import (
	"fmt"
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
