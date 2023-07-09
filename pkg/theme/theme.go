package theme

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mazznoer/colorgrad"
)

type Theme struct {
	Cursor    Color
	Syntax    Color
	Preview   Color
	StatusBar Color
	Search    Color
	Key       func(i, len int) Color
	String    Color
	Null      Color
	Boolean   Color
	Number    Color
	Comment   Color
}
type Color func(s ...string) string

var (
	defaultCursor    = lipgloss.NewStyle().Reverse(true).Render
	defaultPreview   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("8")).Render
	defaultStatusBar = lipgloss.NewStyle().Background(lipgloss.Color("7")).Foreground(lipgloss.Color("0")).Render
	defaultSearch    = lipgloss.NewStyle().Background(lipgloss.Color("11")).Foreground(lipgloss.Color("16")).Render
	defaultNull      = fg("8")
)

var Themes = map[string]Theme{
	"0": {
		Cursor:    defaultCursor,
		Syntax:    noColor,
		Preview:   noColor,
		StatusBar: noColor,
		Search:    defaultSearch,
		Key:       func(_, _ int) Color { return noColor },
		String:    noColor,
		Null:      noColor,
		Boolean:   noColor,
		Number:    noColor,
		Comment:   noColor,
	},
	"1": {
		Cursor:    defaultCursor,
		Syntax:    noColor,
		Preview:   defaultPreview,
		StatusBar: defaultStatusBar,
		Search:    defaultSearch,
		Key:       func(_, _ int) Color { return boldFg("4") },
		String:    boldFg("2"),
		Null:      defaultNull,
		Boolean:   boldFg("3"),
		Number:    boldFg("6"),
		Comment:   boldFg("8"),
	},
	"2": {
		Cursor:    defaultCursor,
		Syntax:    noColor,
		Preview:   defaultPreview,
		StatusBar: defaultStatusBar,
		Search:    defaultSearch,
		Key:       func(_, _ int) Color { return fg("#00F5D4") },
		String:    fg("#00BBF9"),
		Null:      defaultNull,
		Boolean:   fg("#F15BB5"),
		Number:    fg("#9B5DE5"),
		Comment:   fg("#8c8c8c"),
	},
	"3": {
		Cursor:    defaultCursor,
		Syntax:    noColor,
		Preview:   defaultPreview,
		StatusBar: defaultStatusBar,
		Search:    defaultSearch,
		Key:       func(_, _ int) Color { return fg("#faf0ca") },
		String:    fg("#f4d35e"),
		Null:      defaultNull,
		Boolean:   fg("#ee964b"),
		Number:    fg("#ee964b"),
		Comment:   fg("#8c8c8c"),
	},
	"4": {
		Cursor:    defaultCursor,
		Syntax:    noColor,
		Preview:   defaultPreview,
		StatusBar: defaultStatusBar,
		Search:    defaultSearch,
		Key:       func(_, _ int) Color { return fg("#4D96FF") },
		String:    fg("#6BCB77"),
		Null:      defaultNull,
		Boolean:   fg("#FF6B6B"),
		Number:    fg("#FFD93D"),
		Comment:   fg("#8c8c8c"),
	},
	"5": {
		Cursor:    defaultCursor,
		Syntax:    noColor,
		Preview:   defaultPreview,
		StatusBar: defaultStatusBar,
		Search:    defaultSearch,
		Key:       func(_, _ int) Color { return boldFg("42") },
		String:    boldFg("213"),
		Null:      defaultNull,
		Boolean:   boldFg("201"),
		Number:    boldFg("201"),
		Comment:   boldFg("244"),
	},
	"6": {
		Cursor:    defaultCursor,
		Syntax:    noColor,
		Preview:   defaultPreview,
		StatusBar: defaultStatusBar,
		Search:    defaultSearch,
		Key:       func(_, _ int) Color { return gradient("rgb(125,110,221)", "rgb(90%,45%,97%)", "hsl(229,79%,85%)") },
		String:    fg("195"),
		Null:      defaultNull,
		Boolean:   fg("195"),
		Number:    fg("195"),
		Comment:   fg("244"),
	},
	"7": {
		Cursor:    defaultCursor,
		Syntax:    noColor,
		Preview:   defaultPreview,
		StatusBar: defaultStatusBar,
		Search:    defaultSearch,
		Key:       func(_, _ int) Color { return gradient("rgb(123,216,96)", "rgb(255,255,255)") },
		String:    noColor,
		Null:      defaultNull,
		Boolean:   noColor,
		Number:    noColor,
		Comment:   noColor,
	},
	"8": {
		Cursor:    defaultCursor,
		Syntax:    noColor,
		Preview:   defaultPreview,
		StatusBar: defaultStatusBar,
		Search:    defaultSearch,
		Key:       gradientKeys("#ff0000", "#ff8700", "#ffd300", "#deff0a", "#a1ff0a", "#0aff99", "#0aefff", "#147df5", "#580aff", "#be0aff"),
		String:    noColor,
		Null:      defaultNull,
		Boolean:   noColor,
		Number:    noColor,
		Comment:   noColor,
	},
	"9": {
		Cursor:    defaultCursor,
		Syntax:    noColor,
		Preview:   defaultPreview,
		StatusBar: defaultStatusBar,
		Search:    defaultSearch,
		Key:       gradientKeys("rgb(34,126,34)", "rgb(168,251,60)"),
		String:    gradient("rgb(34,126,34)", "rgb(168,251,60)"),
		Null:      defaultNull,
		Boolean:   noColor,
		Number:    noColor,
		Comment:   noColor,
	},
}

func noColor(s ...string) string {
	return s[0]
}

func fg(color string) Color {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render
}

func boldFg(color string) Color {
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(color)).Render
}

func gradient(colors ...string) Color {
	grad, _ := colorgrad.NewGradient().HtmlColors(colors...).Build()
	return func(s ...string) string {
		runes := []rune(s[0])
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
