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

func usage(keyMap KeyMap) string {
	title := lipgloss.NewStyle().Bold(true)
	pad := lipgloss.NewStyle().PaddingLeft(4)
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
    --strict              strict mode

  %v
%v

  %v
    https://fx.wtf

  %v
    Anton Medvedev <anton@medv.io>
`,
		title.Render("fx "+version),
		title.Render("Usage"),
		title.Render("Flags"),
		title.Render("Key Bindings"),
		strings.Join(keyMapInfo(keyMap, pad), "\n"),
		title.Render("More info"),
		title.Render("Author"),
	)
}

func help(keyMap KeyMap) string {
	title := lipgloss.NewStyle().Bold(true)
	pad := lipgloss.NewStyle().PaddingLeft(4)
	return fmt.Sprintf(`
  %v
%v
`,
		title.Render("Key Bindings"),
		strings.Join(keyMapInfo(keyMap, pad), "\n"),
	)
}

func keyMapInfo(keyMap KeyMap, style lipgloss.Style) []string {
	v := reflect.ValueOf(keyMap)
	fields := reflect.VisibleFields(v.Type())

	keys := make([]string, 0)
	for i := range fields {
		k := v.Field(i).Interface().(key.Binding)
		str := k.Help().Key
		if len(str) == 0 {
			if len(k.Keys()) > 5 {
				str = fmt.Sprintf("%v-%v", k.Keys()[0], k.Keys()[len(k.Keys())-1])
			} else {
				str = strings.Join(k.Keys(), ", ")
			}
		}
		keys = append(keys, fmt.Sprintf("%v    ", str))
	}

	desc := make([]string, 0)
	for i := range fields {
		k := v.Field(i).Interface().(key.Binding)
		desc = append(desc, fmt.Sprintf("%v", k.Help().Desc))
	}

	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		strings.Join(keys, "\n"),
		strings.Join(desc, "\n"),
	)

	return strings.Split(style.Render(content), "\n")
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
