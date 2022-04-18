package main

import (
	"fmt"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	"reflect"
	"strings"
)

func usage(keyMap KeyMap) string {
	title := lipgloss.NewStyle().Bold(true)
	return fmt.Sprintf(`fx - terminal JSON viewer

%v  
  fx data.json
  fx data.json .field
  curl ... | fx

%v
%v

%v
  [https://fx.wtf]
`,
		title.Render("Usage"),
		title.Render("Key Bindings"),
		strings.Join(
			keyMapInfo(
				keyMap,
				lipgloss.NewStyle().PaddingLeft(2),
			),
			"\n",
		),
		title.Render("More info"),
	)
}

func keyMapInfo(keyMap KeyMap, style lipgloss.Style) []string {
	v := reflect.ValueOf(keyMap)
	fields := reflect.VisibleFields(v.Type())

	keys := make([]string, 0)
	for i := range fields {
		k := v.Field(i).Interface().(key.Binding)
		str := k.Help().Key
		if width(str) == 0 {
			str = strings.Join(k.Keys(), ", ")
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
