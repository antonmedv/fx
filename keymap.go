package main

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Quit                key.Binding
	PageDown            key.Binding
	PageUp              key.Binding
	HalfPageUp          key.Binding
	HalfPageDown        key.Binding
	GotoTop             key.Binding
	GotoBottom          key.Binding
	GotoSymbol          key.Binding
	GotoRef             key.Binding
	Down                key.Binding
	Up                  key.Binding
	Help                key.Binding
	Expand              key.Binding
	Collapse            key.Binding
	ExpandRecursively   key.Binding
	CollapseRecursively key.Binding
	ExpandAll           key.Binding
	CollapseAll         key.Binding
	CollapseLevel       key.Binding
	NextSibling         key.Binding
	PrevSibling         key.Binding
	ToggleWrap          key.Binding
	ShowSizes           key.Binding
	Yank                key.Binding
	Search              key.Binding
	SearchNext          key.Binding
	SearchPrev          key.Binding
	Preview             key.Binding
	Print               key.Binding
	GoBack              key.Binding
	GoForward           key.Binding
	Dig                 key.Binding
}

var keyMap KeyMap

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
		GotoSymbol: key.NewBinding(
			key.WithKeys("@"),
			key.WithHelp("", "goto symbol"),
		),
		GotoRef: key.NewBinding(
			key.WithKeys("ctrl+g"),
			key.WithHelp("", "goto ref"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("", "down"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("", "up"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("", "show help"),
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
		CollapseLevel: key.NewBinding(
			key.WithKeys("1", "2", "3", "4", "5", "6", "7", "8", "9"),
			key.WithHelp("", "collapse to nth level"),
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
		ShowSizes: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("", "show array/object sizes"),
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
		Preview: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("", "preview"),
		),
		Print: key.NewBinding(
			key.WithKeys("P"),
			key.WithHelp("", "print"),
		),
		GoBack: key.NewBinding(
			key.WithKeys("["),
			key.WithHelp("", "go back"),
		),
		GoForward: key.NewBinding(
			key.WithKeys("]"),
			key.WithHelp("", "go forward"),
		),
		Dig: key.NewBinding(
			key.WithKeys("."),
			key.WithHelp("", "dig"),
		),
	}
}

var (
	yankValueY = key.NewBinding(key.WithKeys("y"))
	yankValueV = key.NewBinding(key.WithKeys("v"))
	yankKey    = key.NewBinding(key.WithKeys("k"))
	yankPath   = key.NewBinding(key.WithKeys("p"))
	arrowUp    = key.NewBinding(key.WithKeys("up"))
	arrowDown  = key.NewBinding(key.WithKeys("down"))
)
