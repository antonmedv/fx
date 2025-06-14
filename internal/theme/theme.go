package theme

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/antonmedv/fx/internal/jsonx"
)

type Theme struct {
	Cursor     Color
	Syntax     Color
	Preview    Color
	StatusBar  Color
	Search     Color
	Key        Color
	String     Color
	Null       Color
	Boolean    Color
	Number     Color
	Size       Color
	Ref        Color
	LineNumber Color
}

type Color func(s string) string

func Value(kind jsonx.Kind, selected bool) Color {
	if selected {
		return CurrentTheme.Cursor
	} else {
		switch kind {
		case jsonx.String:
			return CurrentTheme.String
		case jsonx.Bool:
			return CurrentTheme.Boolean
		case jsonx.Null:
			return CurrentTheme.Null
		case jsonx.Object, jsonx.Array:
			return CurrentTheme.Syntax
		case jsonx.Number:
			return CurrentTheme.Number
		default:
			return noColor
		}
	}
}

var (
	TermOutput = termenv.NewOutput(os.Stderr)
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

	CurrentTheme, ok = themes[themeId]
	if !ok {
		_, _ = fmt.Fprintf(os.Stderr, "fx: unknown theme %q, available themes: %v\n", themeId, themeNames)
		os.Exit(1)
	}

	if TermOutput.ColorProfile() == termenv.Ascii {
		CurrentTheme = themes["0"]
	}

	Colon = CurrentTheme.Syntax(": ")
	ColonPreview = CurrentTheme.Preview(":")
	Comma = CurrentTheme.Syntax(",")
	CommaPreview = CurrentTheme.Preview(",")
	Empty = CurrentTheme.Preview("~")
	Dot3 = CurrentTheme.Preview("…")
	CloseCurlyBracket = CurrentTheme.Syntax("}")
	CloseSquareBracket = CurrentTheme.Syntax("]")
}

var (
	themeNames []string

	CurrentTheme Theme

	underline         = toColor(lipgloss.NewStyle().Underline(true).Render)
	defaultCursor     = toColor(lipgloss.NewStyle().Reverse(true).Render)
	defaultPreview    = toColor(lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render)
	defaultStatusBar  = toColor(lipgloss.NewStyle().Background(lipgloss.Color("7")).Foreground(lipgloss.Color("0")).Render)
	defaultSearch     = toColor(lipgloss.NewStyle().Background(lipgloss.Color("11")).Foreground(lipgloss.Color("16")).Render)
	defaultNull       = fg("243")
	defaultSize       = toColor(lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render)
	defaultLineNumber = toColor(lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render)
)

var (
	Colon              string
	ColonPreview       string
	Comma              string
	CommaPreview       string
	Empty              string
	Dot3               string
	CloseCurlyBracket  string
	CloseSquareBracket string
)

var NoColor = Theme{
	Cursor:     defaultCursor,
	Syntax:     noColor,
	Preview:    noColor,
	StatusBar:  noColor,
	Search:     defaultSearch,
	Key:        noColor,
	String:     noColor,
	Null:       noColor,
	Boolean:    noColor,
	Number:     noColor,
	Size:       noColor,
	Ref:        noColor,
	LineNumber: defaultLineNumber,
}

var themes = map[string]Theme{
	"0": NoColor,
	"1": {
		Cursor:     defaultCursor,
		Syntax:     noColor,
		Preview:    defaultPreview,
		StatusBar:  defaultStatusBar,
		Search:     defaultSearch,
		Key:        boldFg("4"),
		String:     fg("2"),
		Null:       defaultNull,
		Boolean:    fg("5"),
		Number:     fg("6"),
		Size:       defaultSize,
		Ref:        underlineFg("2"),
		LineNumber: defaultLineNumber,
	},
	"2": {
		Cursor:     defaultCursor,
		Syntax:     noColor,
		Preview:    defaultPreview,
		StatusBar:  defaultStatusBar,
		Search:     defaultSearch,
		Key:        fg("2"),
		String:     fg("4"),
		Null:       defaultNull,
		Boolean:    fg("5"),
		Number:     fg("6"),
		Size:       defaultSize,
		Ref:        underlineFg("4"),
		LineNumber: defaultLineNumber,
	},
	"3": {
		Cursor:     defaultCursor,
		Syntax:     noColor,
		Preview:    defaultPreview,
		StatusBar:  defaultStatusBar,
		Search:     defaultSearch,
		Key:        fg("13"),
		String:     fg("11"),
		Null:       defaultNull,
		Boolean:    fg("1"),
		Number:     fg("14"),
		Size:       defaultSize,
		Ref:        underlineFg("11"),
		LineNumber: defaultLineNumber,
	},
	"4": {
		Cursor:     defaultCursor,
		Syntax:     noColor,
		Preview:    defaultPreview,
		StatusBar:  defaultStatusBar,
		Search:     defaultSearch,
		Key:        fg("#00F5D4"),
		String:     fg("#00BBF9"),
		Null:       defaultNull,
		Boolean:    fg("#F15BB5"),
		Number:     fg("#9B5DE5"),
		Size:       defaultSize,
		Ref:        underlineFg("#00BBF9"),
		LineNumber: defaultLineNumber,
	},
	"5": {
		Cursor:     defaultCursor,
		Syntax:     noColor,
		Preview:    defaultPreview,
		StatusBar:  defaultStatusBar,
		Search:     defaultSearch,
		Key:        fg("#faf0ca"),
		String:     fg("#f4d35e"),
		Null:       defaultNull,
		Boolean:    fg("#ee964b"),
		Number:     fg("#ee964b"),
		Size:       defaultSize,
		Ref:        underlineFg("#f4d35e"),
		LineNumber: defaultLineNumber,
	},
	"6": {
		Cursor:     defaultCursor,
		Syntax:     noColor,
		Preview:    defaultPreview,
		StatusBar:  defaultStatusBar,
		Search:     defaultSearch,
		Key:        fg("#4D96FF"),
		String:     fg("#6BCB77"),
		Null:       defaultNull,
		Boolean:    fg("#FF6B6B"),
		Number:     fg("#FFD93D"),
		Size:       defaultSize,
		Ref:        underlineFg("#6BCB77"),
		LineNumber: defaultLineNumber,
	},
	"7": {
		Cursor:     defaultCursor,
		Syntax:     noColor,
		Preview:    defaultPreview,
		StatusBar:  defaultStatusBar,
		Search:     defaultSearch,
		Key:        boldFg("42"),
		String:     boldFg("213"),
		Null:       defaultNull,
		Boolean:    boldFg("201"),
		Number:     boldFg("201"),
		Size:       defaultSize,
		Ref:        underlineFg("213"),
		LineNumber: defaultLineNumber,
	},
	"8": {
		Cursor:     defaultCursor,
		Syntax:     noColor,
		Preview:    defaultPreview,
		StatusBar:  defaultStatusBar,
		Search:     defaultSearch,
		Key:        boldFg("51"),
		String:     fg("195"),
		Null:       defaultNull,
		Boolean:    fg("50"),
		Number:     fg("123"),
		Size:       defaultSize,
		Ref:        underlineFg("195"),
		LineNumber: defaultLineNumber,
	},
	"🔵": {
		Cursor: toColor(lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("33")).
			Render),
		Syntax:     boldFg("33"),
		Preview:    defaultPreview,
		StatusBar:  defaultStatusBar,
		Search:     defaultSearch,
		Key:        fg("33"),
		String:     noColor,
		Null:       noColor,
		Boolean:    noColor,
		Number:     noColor,
		Size:       defaultSize,
		Ref:        underline,
		LineNumber: defaultLineNumber,
	},
	"🥝": {
		Cursor:     defaultCursor,
		Syntax:     fg("179"),
		Preview:    defaultPreview,
		StatusBar:  defaultStatusBar,
		Search:     defaultSearch,
		Key:        boldFg("154"),
		String:     fg("82"),
		Null:       fg("230"),
		Boolean:    fg("226"),
		Number:     fg("226"),
		Size:       defaultSize,
		Ref:        underlineFg("82"),
		LineNumber: defaultLineNumber,
	},
}

func noColor(s string) string {
	return s
}

func toColor(f func(s ...string) string) Color {
	return func(s string) string {
		return f(s)
	}
}

func fg(color string) Color {
	return toColor(lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render)
}

func underlineFg(color string) Color {
	return toColor(lipgloss.NewStyle().Underline(true).Foreground(lipgloss.Color(color)).Render)
}

func boldFg(color string) Color {
	return toColor(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(color)).Render)
}

func ThemeTester() {
	for _, name := range themeNames {
		t := themes[name]
		comma := t.Syntax(",")
		colon := t.Syntax(":")

		fmt.Println(fmt.Sprintf("export FX_THEME=%q", name))
		fmt.Println(t.Syntax("{"))

		fmt.Printf("  %v%v %v%v\n",
			t.Key("\"string\""),
			colon,
			t.String("\"Fox jumps over the lazy dog\""),
			comma)

		fmt.Printf("  %v%v %v%v\n",
			t.Key("\"number\""),
			colon,
			t.Number("1234567890"),
			comma)

		fmt.Printf("  %v%v %v%v\n",
			t.Key("\"boolean\""),
			colon,
			t.Boolean("true"),
			comma)
		fmt.Printf("  %v%v %v%v\n",
			t.Key("\"null\""),
			colon,
			t.Null("null"),
			comma)
		fmt.Printf("  %v%v %v%v%v\n",
			t.Key("\"collapsed\""),
			colon,
			t.Syntax("{"),
			t.Preview("…"),
			t.Syntax("}"),
		)
		fmt.Println(t.Syntax("}"))
		println()
	}
}

func ExportThemes() {
	lipgloss.SetColorProfile(termenv.ANSI256) // Export in Terminal.app compatible colors
	placeholder := "_"
	extract := func(b string) string {
		matches := regexp.
			MustCompile(`^\x1b\[(.+)m_`).
			FindStringSubmatch(b)
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
