package main

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Quit         key.Binding
	Help         key.Binding
	PageDown     key.Binding
	PageUp       key.Binding
	HalfPageUp   key.Binding
	HalfPageDown key.Binding
	Down         key.Binding
	Up           key.Binding
	Expand       key.Binding
	Collapse     key.Binding
	GotoTop      key.Binding
	GotoBottom   key.Binding
	ToggleWrap   key.Binding
	ExpandAll    key.Binding
	CollapseAll  key.Binding
	Search       key.Binding
	Next         key.Binding
	Prev         key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c", "esc"),
			key.WithHelp("", "exit program"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("", "show help"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", " ", "f"),
			key.WithHelp("pgdown, space, f", "page down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "b"),
			key.WithHelp("pgup, b", "page up"),
		),
		HalfPageUp: key.NewBinding(
			key.WithKeys("u", "ctrl+u"),
			key.WithHelp("", "half page up"),
		),
		HalfPageDown: key.NewBinding(
			key.WithKeys("d", "ctrl+d"),
			key.WithHelp("", "half page down"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("", "down"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("", "up"),
		),
		Expand: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("", "expand"),
		),
		Collapse: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("", "collapse"),
		),
		GotoTop: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("", "goto top"),
		),
		GotoBottom: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("", "goto bottom"),
		),
		ToggleWrap: key.NewBinding(
			key.WithKeys("z"),
			key.WithHelp("", "toggle strings wrap"),
		),
		ExpandAll: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("", "expand all"),
		),
		CollapseAll: key.NewBinding(
			key.WithKeys("E"),
			key.WithHelp("", "collapse all"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("", "search regexp"),
		),
		Next: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("", "next search result"),
		),
		Prev: key.NewBinding(
			key.WithKeys("N"),
			key.WithHelp("", "prev search result"),
		),
	}
}
