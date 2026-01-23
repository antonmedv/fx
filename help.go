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
    --game-of-life        play the game of life

  %v
    https://fx.wtf

  %v
    Anton Medvedev <anton@medv.io>
`,
		title.Render("fx "+version),
		title.Render("Usage"),
		title.Render("Flags"),
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

	// Header
	sb.WriteString("\n")
	sb.WriteString(titleStyle.Render("  Key Bindings"))
	sb.WriteString("\n")
	sb.WriteString(dimStyle.Render("  ─────────────────────────────────────────"))
	sb.WriteString("\n")

	for _, cat := range categoryOrder {
		bindings, ok := categories[cat]
		if !ok || len(bindings) == 0 {
			continue
		}

		sb.WriteString(categoryStyle.Render("  " + cat))
		sb.WriteString("\n")

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

			desc := binding.Help().Desc
			keyFormatted := keyStyle.Render(fmt.Sprintf("%-20s", keyStr))
			descFormatted := descStyle.Render(desc)

			sb.WriteString(fmt.Sprintf("    %s  %s\n", keyFormatted, descFormatted))
		}
	}

	sb.WriteString("\n")
	sb.WriteString(dimStyle.Render("  ─────────────────────────────────────────"))
	sb.WriteString("\n")
	sb.WriteString(dimStyle.Render("  Press q or ? to close"))
	sb.WriteString("\n")

	return sb.String()
}

func exit() {
	if showLetter(time.Now()) {
		style := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2)
		_, _ = fmt.Fprintln(os.Stderr, style.Render(`Hello, kind human. :)

This is fx speaking. I know you’re busy, and I won’t take much
of your time.

Every day, I quietly sit in your terminal, helping you explore
and shape your data. No popups, no ads, quiet, helpful work.

But today is different.

Today, I’m asking for something small in return. Just for today.

If fx has saved you time, solved a problem, or simply made your 
life in the terminal a little easier, please consider supporting 
the developer who made me:

    https://github.com/sponsors/antonmedv

He built fx as a passion project, shared it freely with the world, 
and has kept improving it—all without asking much.

Your support helps keep fx alive, maintained, and improving.
Even a small donation means a lot. It shows that you care, that 
this kind of work matters.

This message only appears once, on the first Tuesday of December.
Tomorrow I’ll be silent again.

Thank you for reading. And thank you for using fx.`))
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
