//go:build !windows

package main

import "os/exec"

// shellCommand runs cmd through the user's shell, so quotes and pipes work.
func shellCommand(cmd string) *exec.Cmd {
	return exec.Command(lookup([]string{"SHELL"}, "sh"), "-c", cmd)
}
