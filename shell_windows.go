//go:build windows

package main

import (
	"fmt"
	"os/exec"
	"syscall"
)

// shellCommand runs cmd through cmd.exe. The command line is passed as is:
// cmd.exe has its own quoting rules, which the escaping exec.Command does
// for its arguments breaks. /S makes cmd.exe strip only the outer quotes.
func shellCommand(cmd string) *exec.Cmd {
	comspec := lookup([]string{"COMSPEC"}, "cmd")
	c := exec.Command(comspec)
	c.SysProcAttr = &syscall.SysProcAttr{
		CmdLine: fmt.Sprintf(`"%s" /S /C "%s"`, comspec, cmd),
	}
	return c
}
