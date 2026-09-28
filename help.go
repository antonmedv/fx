package main

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

func usage() string {
	title := lipgloss.NewStyle().Bold(true)
	return fmt.Sprintf(`
  %v
    Terminal JSON viewer

  %v
    fx data.json
    fx data.json .field
    curl ... | fx

  %v
    -h, --help            print help
    -v, --version         print version
    --themes              print themes
    --comp <shell>        print completion script
    -r, --raw             treat input as a raw string
    -s, --slurp           read all inputs into an array
    --yaml                parse input as YAML
    --toml                parse input as TOML
    --strict              strict mode
    --no-inline           disable inlining in output
    --no-paging            print to stdout without the interactive viewer
    --game-of-life        play the game of life

  %v
    FX_NO_PAGER            enable --no-paging when set

  %v
    https://fx.wtf

  %v
    Anton Medvedev <anton@medv.io>
`,
		title.Render("fx "+version),
		title.Render("Usage"),
		title.Render("Flags"),
		title.Render("Environment"),
		title.Render("More info"),
		title.Render("Author"),
	)
}

var categoryOrder = []string{
	"Navigation",
	"Expand / Collapse",
	"Search",
	"Actions",
	"View",
	"Other",
}

func help(keyMap KeyMap) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true)

	categoryStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		MarginTop(1)

	keyStyle := lipgloss.NewStyle()

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))

	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))

	// Group bindings by category using struct tags
	v := reflect.ValueOf(keyMap)
	t := v.Type()
	categories := make(map[string][]key.Binding)

	for _, field := range reflect.VisibleFields(t) {
		category := field.Tag.Get("category")
		if category == "" {
			continue
		}
		binding := v.FieldByName(field.Name).Interface().(key.Binding)
		categories[category] = append(categories[category], binding)
	}

	var sb strings.Builder

	section := func(title string) {
		sb.WriteString("\n")
		sb.WriteString(titleStyle.Render("  " + title))
		sb.WriteString("\n")
		sb.WriteString(dimStyle.Render("  ─────────────────────────────────────────"))
		sb.WriteString("\n")
	}

	category := func(name string) {
		sb.WriteString(categoryStyle.Render("  " + name))
		sb.WriteString("\n")
	}

	row := func(keyStr, desc string) {
		keyFormatted := keyStyle.Render(fmt.Sprintf("%-20s", keyStr))
		descFormatted := descStyle.Render(desc)
		sb.WriteString(fmt.Sprintf("    %s  %s\n", keyFormatted, descFormatted))
	}

	text := func(lines ...string) {
		sb.WriteString("\n")
		for _, line := range lines {
			sb.WriteString("    " + line + "\n")
		}
	}

	section("Key Bindings")

	for _, cat := range categoryOrder {
		bindings, ok := categories[cat]
		if !ok || len(bindings) == 0 {
			continue
		}

		category(cat)

		for _, binding := range bindings {
			keyStr := binding.Help().Key
			if len(keyStr) == 0 {
				keys := binding.Keys()
				if len(keys) > 5 {
					keyStr = fmt.Sprintf("%v-%v", keys[0], keys[len(keys)-1])
				} else {
					keyStr = strings.Join(keys, ", ")
				}
			}
			row(keyStr, binding.Help().Desc)
		}
	}

	sb.WriteString("\n")
	section("Commands")
	text("Press : to open the command line.")
	sb.WriteString("\n")
	row(":<n>", "goto line n")
	row(":q", "quit")

	sb.WriteString("\n")
	section("Query")
	text(
		"Press . to open the query input,",
		"type a query and press enter.",
	)

	category("Syntax")
	row(".name", "property, .items[0] for index")
	row(".items[].name", "map over array")
	row("@.name", "map with expression")
	row("?.active", "filter with expression")
	row("x => x.a + 1", "any JS expression, x is input")
	row("len keys sort uniq", "built-in functions")
	row(".items len", "space separates piped queries")

	category("Input Keys")
	row("tab, shift+tab", "complete, cycle candidates")
	row("right, end, ctrl+e", "accept suggestion")
	row("up, down", "query history")
	row("enter", "apply query")
	row("esc", "cancel, restore original")

	sb.WriteString("\n\n\n")
	sb.WriteString(dimStyle.Render("  Press q or ? to close"))
	sb.WriteString("\n")

	return sb.String()
}

func exit() {
	if showLetter(time.Now()) {
		style := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2)
		_, _ = fmt.Fprintln(os.Stderr, style.Render(`Hello, kind human. :)

This is fx speaking.

Most days, I quietly sit in your terminal and help you work
with JSON. No ads, no popups, no interruptions.

Today, I have one small request.

If fx has been useful to you, please consider supporting
the developer who keeps me alive and improving:

    https://github.com/sponsors/antonmedv

Even a small contribution helps.

That’s all. Tomorrow I’ll be quiet again.

Thank you for using fx.`))
	}
}

func showLetter(t time.Time) bool {
	if t.Month() != time.December {
		return false
	}
	firstOfDecember := time.Date(t.Year(), time.December, 1, 0, 0, 0, 0, t.Location())
	offset := (int(time.Tuesday) - int(firstOfDecember.Weekday()) + 7) % 7
	firstTuesday := firstOfDecember.AddDate(0, 0, offset)
	return t.Year() == firstTuesday.Year() &&
		t.Month() == firstTuesday.Month() &&
		t.Day() == firstTuesday.Day()
}
