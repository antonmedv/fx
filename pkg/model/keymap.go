package model

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Quit                key.Binding
	PageDown            key.Binding
	PageUp              key.Binding
	HalfPageUp          key.Binding
	HalfPageDown        key.Binding
	GotoTop             key.Binding
	GotoBottom          key.Binding
	Down                key.Binding
	Up                  key.Binding
	Expand              key.Binding
	Collapse            key.Binding
	ExpandRecursively   key.Binding
	CollapseRecursively key.Binding
	ExpandAll           key.Binding
	CollapseAll         key.Binding
	NextSibling         key.Binding
	PrevSibling         key.Binding
	ToggleWrap          key.Binding
	Yank                key.Binding
	Search              key.Binding
	SearchNext          key.Binding
	SearchPrev          key.Binding
	Dig                 key.Binding
}

var keyMap KeyMap

// GetKeyMap is a getter for global keyMap. Do not modify.
func GetKeyMap() KeyMap {
	return keyMap
}

func init() {
	keyMap = KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c", "esc"),
			key.WithHelp("", "exit program"),
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
		GotoTop: key.NewBinding(
			key.WithKeys("g", "home"),
			key.WithHelp("", "goto top"),
		),
		GotoBottom: key.NewBinding(
			key.WithKeys("G", "end"),
			key.WithHelp("", "goto bottom"),
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
			key.WithKeys("right", "l", "enter"),
			key.WithHelp("", "expand"),
		),
		Collapse: key.NewBinding(
			key.WithKeys("left", "h", "backspace"),
			key.WithHelp("", "collapse"),
		),
		ExpandRecursively: key.NewBinding(
			key.WithKeys("L", "shift+right"),
			key.WithHelp("", "expand recursively"),
		),
		CollapseRecursively: key.NewBinding(
			key.WithKeys("H", "shift+left"),
			key.WithHelp("", "collapse recursively"),
		),
		ExpandAll: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("", "expand all"),
		),
		CollapseAll: key.NewBinding(
			key.WithKeys("E"),
			key.WithHelp("", "collapse all"),
		),
		NextSibling: key.NewBinding(
			key.WithKeys("J", "shift+down"),
			key.WithHelp("", "next sibling"),
		),
		PrevSibling: key.NewBinding(
			key.WithKeys("K", "shift+up"),
			key.WithHelp("", "previous sibling"),
		),
		ToggleWrap: key.NewBinding(
			key.WithKeys("z"),
			key.WithHelp("", "toggle strings wrap"),
		),
		Yank: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("", "yank/copy"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("", "search regexp"),
		),
		SearchNext: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("", "next search result"),
		),
		SearchPrev: key.NewBinding(
			key.WithKeys("N"),
			key.WithHelp("", "prev search result"),
		),
		Dig: key.NewBinding(
			key.WithKeys("."),
			key.WithHelp("", "dig json"),
		),
	}
}

var (
	yankValue = key.NewBinding(key.WithKeys("y"))
	yankKey   = key.NewBinding(key.WithKeys("k"))
	yankPath  = key.NewBinding(key.WithKeys("p"))
	arrowUp   = key.NewBinding(key.WithKeys("up"))
	arrowDown = key.NewBinding(key.WithKeys("down"))
)
