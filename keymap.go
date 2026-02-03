package main

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Up                  key.Binding `category:"Navigation"`
	Down                key.Binding `category:"Navigation"`
	PageUp              key.Binding `category:"Navigation"`
	PageDown            key.Binding `category:"Navigation"`
	HalfPageUp          key.Binding `category:"Navigation"`
	HalfPageDown        key.Binding `category:"Navigation"`
	GotoTop             key.Binding `category:"Navigation"`
	GotoBottom          key.Binding `category:"Navigation"`
	NextSibling         key.Binding `category:"Navigation"`
	PrevSibling         key.Binding `category:"Navigation"`
	Expand              key.Binding `category:"Expand / Collapse"`
	Collapse            key.Binding `category:"Expand / Collapse"`
	ExpandRecursively   key.Binding `category:"Expand / Collapse"`
	CollapseRecursively key.Binding `category:"Expand / Collapse"`
	ExpandAll           key.Binding `category:"Expand / Collapse"`
	CollapseAll         key.Binding `category:"Expand / Collapse"`
	CollapseLevel       key.Binding `category:"Expand / Collapse"`
	Search              key.Binding `category:"Search"`
	SearchNext          key.Binding `category:"Search"`
	SearchPrev          key.Binding `category:"Search"`
	GotoSymbol          key.Binding `category:"Search"`
	GotoRef             key.Binding `category:"Search"`
	Yank                key.Binding `category:"Actions"`
	Delete              key.Binding `category:"Actions"`
	Preview             key.Binding `category:"Actions"`
	Print               key.Binding `category:"Actions"`
	Open                key.Binding `category:"Actions"`
	DecodeJSON          key.Binding `category:"Actions"`
	EncodeJSON          key.Binding `category:"Actions"`
	ToggleWrap          key.Binding `category:"View"`
	ShowSelector        key.Binding `category:"View"`
	GoBack              key.Binding `category:"Navigation"`
	GoForward           key.Binding `category:"Navigation"`
	Help                key.Binding `category:"Other"`
	CommandLine         key.Binding `category:"Other"`
	Quit                key.Binding `category:"Other"`
	Suspend             key.Binding `category:"Other"`
}

var keyMap KeyMap

func init() {
	keyMap = KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c", "esc"),
			key.WithHelp("", "exit program"),
		),
		Suspend: key.NewBinding(
			key.WithKeys("ctrl+z"),
			key.WithHelp("", "suspend program"),
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
			key.WithKeys("ctrl+u"),
			key.WithHelp("", "half page up"),
		),
		HalfPageDown: key.NewBinding(
			key.WithKeys("ctrl+d"),
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
		ShowSelector: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("", "show sizes/line numbers"),
		),
		Yank: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("", "yank/copy"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("", "delete node"),
		),
		CommandLine: key.NewBinding(
			key.WithKeys(":"),
			key.WithHelp("", "open command line"),
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
			key.WithHelp("", "preview node"),
		),
		Print: key.NewBinding(
			key.WithKeys("P"),
			key.WithHelp("", "print to stdout"),
		),
		Open: key.NewBinding(
			key.WithKeys("v"),
			key.WithHelp("", "open in editor"),
		),
		DecodeJSON: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "decode JSON string"),
		),
		EncodeJSON: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "encode value as string"),
		),
		GoBack: key.NewBinding(
			key.WithKeys("["),
			key.WithHelp("", "go back"),
		),
		GoForward: key.NewBinding(
			key.WithKeys("]"),
			key.WithHelp("", "go forward"),
		),
	}
}

var (
	yankValueY      = key.NewBinding(key.WithKeys("y"))
	yankValueV      = key.NewBinding(key.WithKeys("v"))
	yankKey         = key.NewBinding(key.WithKeys("k"))
	yankPath        = key.NewBinding(key.WithKeys("p"))
	yankKeyValue    = key.NewBinding(key.WithKeys("b"))
	arrowUp         = key.NewBinding(key.WithKeys("up"))
	arrowDown       = key.NewBinding(key.WithKeys("down"))
	showSizes       = key.NewBinding(key.WithKeys("s"))
	showLineNumbers = key.NewBinding(key.WithKeys("l"))
)
