package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

type theme struct {
	Cursor    color
	Syntax    color
	Preview   color
	StatusBar color
	Search    color
	Key       color
	String    color
	Null      color
	Boolean   color
	Number    color
}

type color func(s []byte) []byte

func valueStyle(b []byte, selected, chunk bool) color {
	if selected {
		return currentTheme.Cursor
	} else if chunk {
		return currentTheme.String
	} else {
		switch b[0] {
		case '"':
			return currentTheme.String
		case 't', 'f':
			return currentTheme.Boolean
		case 'n':
			return currentTheme.Null
		case '{', '[', '}', ']':
			return currentTheme.Syntax
		default:
			if isDigit(b[0]) || b[0] == '-' {
				return currentTheme.Number
			}
			return noColor
		}
	}
}

var (
	termOutput = termenv.NewOutput(os.Stderr)
)

func init() {
	themeNames = make([]string, 0, len(themes))
	for name := range themes {
		themeNames = append(themeNames, name)
	}
	sort.Strings(themeNames)

	themeId, ok := os.LookupEnv("FX_THEME")
	if !ok {
		themeId = "1"
	}

	currentTheme, ok = themes[themeId]
	if !ok {
		_, _ = fmt.Fprintf(os.Stderr, "fx: unknown theme %q, available themes: %v\n", themeId, themeNames)
		os.Exit(1)
	}

	if termOutput.ColorProfile() == termenv.Ascii {
		currentTheme = themes["0"]
	}

	colon = currentTheme.Syntax([]byte{':', ' '})
	colonPreview = currentTheme.Preview([]byte{':'})
	comma = currentTheme.Syntax([]byte{','})
	empty = currentTheme.Preview([]byte{'~'})
	dot3 = currentTheme.Preview([]byte("…"))
	closeCurlyBracket = currentTheme.Syntax([]byte{'}'})
	closeSquareBracket = currentTheme.Syntax([]byte{']'})
}

var (
	themeNames       []string
	currentTheme     theme
	defaultCursor    = toColor(lipgloss.NewStyle().Reverse(true).Render)
	defaultPreview   = toColor(lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render)
	defaultStatusBar = toColor(lipgloss.NewStyle().Background(lipgloss.Color("7")).Foreground(lipgloss.Color("0")).Render)
	defaultSearch    = toColor(lipgloss.NewStyle().Background(lipgloss.Color("11")).Foreground(lipgloss.Color("16")).Render)
	defaultNull      = fg("243")
)

var (
	colon              []byte
	colonPreview       []byte
	comma              []byte
	empty              []byte
	dot3               []byte
	closeCurlyBracket  []byte
	closeSquareBracket []byte
)

var themes = map[string]theme{
	"0": {
		Cursor:    defaultCursor,
		Syntax:    noColor,
		Preview:   noColor,
		StatusBar: noColor,
		Search:    defaultSearch,
		Key:       noColor,
		String:    noColor,
		Null:      noColor,
		Boolean:   noColor,
		Number:    noColor,
	},
	"1": {
		Cursor:    defaultCursor,
		Syntax:    noColor,
		Preview:   defaultPreview,
		StatusBar: defaultStatusBar,
		Search:    defaultSearch,
		Key:       boldFg("4"),
		String:    fg("2"),
		Null:      defaultNull,
		Boolean:   fg("5"),
		Number:    fg("6"),
	},
	"2": {
		Cursor:    defaultCursor,
		Syntax:    noColor,
		Preview:   defaultPreview,
		StatusBar: defaultStatusBar,
		Search:    defaultSearch,
		Key:       fg("2"),
		String:    fg("4"),
		Null:      defaultNull,
		Boolean:   fg("5"),
		Number:    fg("6"),
	},
	"3": {
		Cursor:    defaultCursor,
		Syntax:    noColor,
		Preview:   defaultPreview,
		StatusBar: defaultStatusBar,
		Search:    defaultSearch,
		Key:       fg("13"),
		String:    fg("11"),
		Null:      defaultNull,
		Boolean:   fg("1"),
		Number:    fg("14"),
	},
	"4": {
		Cursor:    defaultCursor,
		Syntax:    noColor,
		Preview:   defaultPreview,
		StatusBar: defaultStatusBar,
		Search:    defaultSearch,
		Key:       fg("#00F5D4"),
		String:    fg("#00BBF9"),
		Null:      defaultNull,
		Boolean:   fg("#F15BB5"),
		Number:    fg("#9B5DE5"),
	},
	"5": {
		Cursor:    defaultCursor,
		Syntax:    noColor,
		Preview:   defaultPreview,
		StatusBar: defaultStatusBar,
		Search:    defaultSearch,
		Key:       fg("#faf0ca"),
		String:    fg("#f4d35e"),
		Null:      defaultNull,
		Boolean:   fg("#ee964b"),
		Number:    fg("#ee964b"),
	},
	"6": {
		Cursor:    defaultCursor,
		Syntax:    noColor,
		Preview:   defaultPreview,
		StatusBar: defaultStatusBar,
		Search:    defaultSearch,
		Key:       fg("#4D96FF"),
		String:    fg("#6BCB77"),
		Null:      defaultNull,
		Boolean:   fg("#FF6B6B"),
		Number:    fg("#FFD93D"),
	},
	"7": {
		Cursor:    defaultCursor,
		Syntax:    noColor,
		Preview:   defaultPreview,
		StatusBar: defaultStatusBar,
		Search:    defaultSearch,
		Key:       boldFg("42"),
		String:    boldFg("213"),
		Null:      defaultNull,
		Boolean:   boldFg("201"),
		Number:    boldFg("201"),
	},
	"8": {
		Cursor:    defaultCursor,
		Syntax:    noColor,
		Preview:   defaultPreview,
		StatusBar: defaultStatusBar,
		Search:    defaultSearch,
		Key:       boldFg("51"),
		String:    fg("195"),
		Null:      defaultNull,
		Boolean:   fg("50"),
		Number:    fg("123"),
	},
	"🔵": {
		Cursor: toColor(lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("33")).
			Render),
		Syntax:    boldFg("33"),
		Preview:   defaultPreview,
		StatusBar: defaultStatusBar,
		Search:    defaultSearch,
		Key:       fg("33"),
		String:    noColor,
		Null:      noColor,
		Boolean:   noColor,
		Number:    noColor,
	},
	"🥝": {
		Cursor:    defaultCursor,
		Syntax:    fg("179"),
		Preview:   defaultPreview,
		StatusBar: defaultStatusBar,
		Search:    defaultSearch,
		Key:       boldFg("154"),
		String:    fg("82"),
		Null:      fg("230"),
		Boolean:   fg("226"),
		Number:    fg("226"),
	},
}

func noColor(s []byte) []byte {
	return s
}

func toColor(f func(s ...string) string) color {
	return func(s []byte) []byte {
		return []byte(f(string(s)))
	}
}

func fg(color string) color {
	return toColor(lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render)
}

func boldFg(color string) color {
	return toColor(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(color)).Render)
}

func themeTester() {
	title := lipgloss.NewStyle().Bold(true)
	for _, name := range themeNames {
		t := themes[name]
		comma := string(t.Syntax([]byte{','}))
		colon := string(t.Syntax([]byte{':'}))

		fmt.Println(title.Render(fmt.Sprintf("Theme %q", name)))
		fmt.Println(string(t.Syntax([]byte("{"))))

		fmt.Printf("  %v%v %v%v\n",
			string(t.Key([]byte("\"string\""))),
			colon,
			string(t.String([]byte("\"Fox jumps over the lazy dog\""))),
			comma)

		fmt.Printf("  %v%v %v%v\n",
			string(t.Key([]byte("\"number\""))),
			colon,
			string(t.Number([]byte("1234567890"))),
			comma)

		fmt.Printf("  %v%v %v%v\n",
			string(t.Key([]byte("\"boolean\""))),
			colon,
			string(t.Boolean([]byte("true"))),
			comma)
		fmt.Printf("  %v%v %v%v\n",
			string(t.Key([]byte("\"null\""))),
			colon,
			string(t.Null([]byte("null"))),
			comma)
		fmt.Println(string(t.Syntax([]byte("}"))))
		println()
	}
}

func exportThemes() {
	lipgloss.SetColorProfile(termenv.ANSI256) // Export in Terminal.app compatible colors
	placeholder := []byte{'_'}
	extract := func(b []byte) string {
		matches := regexp.
			MustCompile(`^\x1b\[(.+)m_`).
			FindStringSubmatch(string(b))
		if len(matches) == 0 {
			return ""
		} else {
			return matches[1]
		}
	}
	var export = map[string][]string{}
	for _, name := range themeNames {
		t := themes[name]
		export[name] = append(export[name], extract(t.Syntax(placeholder)))
		export[name] = append(export[name], extract(t.Key(placeholder)))
		export[name] = append(export[name], extract(t.String(placeholder)))
		export[name] = append(export[name], extract(t.Number(placeholder)))
		export[name] = append(export[name], extract(t.Boolean(placeholder)))
		export[name] = append(export[name], extract(t.Null(placeholder)))
	}
	data, err := json.Marshal(export)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(data))
}
