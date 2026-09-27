//go:build windows

package main

import (
	"os"

	"github.com/charmbracelet/x/term"
)

// reapplyRawMode puts the console back into raw mode. See tty_unix.go.
func reapplyRawMode() {
	tty, err := os.OpenFile("CONIN$", os.O_RDWR, 0)
	if err != nil {
		return
	}
	defer tty.Close()
	_, _ = term.MakeRaw(tty.Fd())
}
