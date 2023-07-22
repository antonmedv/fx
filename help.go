package main

import (
	"fmt"
	"strings"

	"github.com/antonmedv/fx/pkg/model"
	"github.com/charmbracelet/lipgloss"
)

func usage(keyMap model.KeyMap) string {
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
    -h, --help          print help
    -v, --version       print version
    --print-code        print code of the reducer

  %v
%v

  %v
    [https://fx.wtf]
`,
		title.Render("fx "+version),
		title.Render("Usage"),
		title.Render("Flags"),
		title.Render("Key Bindings"),
		strings.Join(model.KeyMapInfo(keyMap, pad), "\n"),
		title.Render("More info"),
	)
}
