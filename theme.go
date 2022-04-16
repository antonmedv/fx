package main

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/mazznoer/colorgrad"
	"strings"
)

type Theme struct {
	cursor    Color
	syntax    Color
	preview   Color
	statusBar Color
	search    Color
	key       func(i, len int) Color
	string    Color
	null      Color
	boolean   Color
	number    Color
}
type Color func(s string) string

func fg(color string) Color {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render
}

func boldFg(color string) Color {
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(color)).Render
}

var (
	defaultCursor    = lipgloss.NewStyle().Reverse(true).Render
	defaultPreview   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("8")).Render
	defaultStatusBar = lipgloss.NewStyle().Background(lipgloss.Color("7")).Foreground(lipgloss.Color("0")).Render
	defaultSearch    = lipgloss.NewStyle().Background(lipgloss.Color("11")).Foreground(lipgloss.Color("16")).Render
	defaultNull      = fg("8")
)

var themes = map[string]Theme{
	"0": {
		cursor:    defaultCursor,
		syntax:    noColor,
		preview:   noColor,
		statusBar: noColor,
		search:    defaultSearch,
		key:       func(_, _ int) Color { return noColor },
		string:    noColor,
		null:      noColor,
		boolean:   noColor,
		number:    noColor,
	},
	"1": {
		cursor:    defaultCursor,
		syntax:    noColor,
		preview:   defaultPreview,
		statusBar: defaultStatusBar,
		search:    defaultSearch,
		key:       func(_, _ int) Color { return boldFg("4") },
		string:    boldFg("2"),
		null:      defaultNull,
		boolean:   boldFg("3"),
		number:    boldFg("6"),
	},
	"2": {
		cursor:    defaultCursor,
		syntax:    noColor,
		preview:   defaultPreview,
		statusBar: defaultStatusBar,
		search:    defaultSearch,
		key:       func(_, _ int) Color { return fg("#00F5D4") },
		string:    fg("#00BBF9"),
		null:      defaultNull,
		boolean:   fg("#F15BB5"),
		number:    fg("#9B5DE5"),
	},
	"3": {
		cursor:    defaultCursor,
		syntax:    noColor,
		preview:   defaultPreview,
		statusBar: defaultStatusBar,
		search:    defaultSearch,
		key:       func(_, _ int) Color { return fg("#faf0ca") },
		string:    fg("#f4d35e"),
		null:      defaultNull,
		boolean:   fg("#ee964b"),
		number:    fg("#ee964b"),
	},
	"4": {
		cursor:    defaultCursor,
		syntax:    noColor,
		preview:   defaultPreview,
		statusBar: defaultStatusBar,
		search:    defaultSearch,
		key:       func(_, _ int) Color { return fg("#4D96FF") },
		string:    fg("#6BCB77"),
		null:      defaultNull,
		boolean:   fg("#FF6B6B"),
		number:    fg("#FFD93D"),
	},
	"5": {
		cursor:    defaultCursor,
		syntax:    noColor,
		preview:   defaultPreview,
		statusBar: defaultStatusBar,
		search:    defaultSearch,
		key:       func(_, _ int) Color { return boldFg("42") },
		string:    boldFg("213"),
		null:      defaultNull,
		boolean:   boldFg("201"),
		number:    boldFg("201"),
	},
	"6": {
		cursor:    defaultCursor,
		syntax:    noColor,
		preview:   defaultPreview,
		statusBar: defaultStatusBar,
		search:    defaultSearch,
		key:       func(_, _ int) Color { return gradient("rgb(125,110,221)", "rgb(90%,45%,97%)", "hsl(229,79%,85%)") },
		string:    fg("195"),
		null:      defaultNull,
		boolean:   fg("195"),
		number:    fg("195"),
	},
	"7": {
		cursor:    defaultCursor,
		syntax:    noColor,
		preview:   defaultPreview,
		statusBar: defaultStatusBar,
		search:    defaultSearch,
		key:       func(_, _ int) Color { return gradient("rgb(123,216,96)", "rgb(255,255,255)") },
		string:    noColor,
		null:      defaultNull,
		boolean:   noColor,
		number:    noColor,
	},
	"8": {
		cursor:    defaultCursor,
		syntax:    noColor,
		preview:   defaultPreview,
		statusBar: defaultStatusBar,
		search:    defaultSearch,
		key:       gradientKeys("#ff0000", "#ff8700", "#ffd300", "#deff0a", "#a1ff0a", "#0aff99", "#0aefff", "#147df5", "#580aff", "#be0aff"),
		string:    noColor,
		null:      defaultNull,
		boolean:   noColor,
		number:    noColor,
	},
	"9": {
		cursor:    defaultCursor,
		syntax:    noColor,
		preview:   defaultPreview,
		statusBar: defaultStatusBar,
		search:    defaultSearch,
		key:       gradientKeys("rgb(34,126,34)", "rgb(168,251,60)"),
		string:    gradient("rgb(34,126,34)", "rgb(168,251,60)"),
		null:      defaultNull,
		boolean:   noColor,
		number:    noColor,
	},
}

func noColor(s string) string {
	return s
}

func gradient(colors ...string) Color {
	grad, _ := colorgrad.NewGradient().HtmlColors(colors...).Build()
	return func(s string) string {
		runes := []rune(s)
		colors := grad.ColorfulColors(uint(len(runes)))
		var out strings.Builder
		for i, r := range runes {
			style := lipgloss.NewStyle().Foreground(lipgloss.Color(colors[i].Hex()))
			out.WriteString(style.Render(string(r)))
		}
		return out.String()
	}
}

func gradientKeys(colors ...string) func(i, len int) Color {
	grad, _ := colorgrad.NewGradient().HtmlColors(colors...).Build()
	return func(i, len int) Color {
		return lipgloss.NewStyle().Foreground(lipgloss.Color(grad.At(float64(i) / float64(len)).Hex())).Render
	}
}
