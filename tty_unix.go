//go:build !windows

package main

import (
	"os"

	"github.com/charmbracelet/x/term"
)

// reapplyRawMode puts the terminal back into raw mode. The process writing
// to our stdin may restore its own saved terminal state when it exits
// (Node.js does this), which turns off the raw mode bubbletea enabled.
func reapplyRawMode() {
	tty, err := os.Open("/dev/tty")
	if err != nil {
		return
	}
	defer tty.Close()
	_, _ = term.MakeRaw(tty.Fd())
}
