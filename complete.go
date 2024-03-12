package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/antonmedv/fx/internal/shlex"
)

var flags = []string{
	"--help",
	"--raw",
	"--slurp",
	"--themes",
	"--version",
	"--yaml",
	"-h",
	"-r",
	"-s",
	"-v",
}

func complete() {
	compLine, ok := os.LookupEnv("COMP_LINE")

	if !ok || len(os.Args) < 3 {
		return
	}

	// Get the current partial word to be completed
	partial := os.Args[2]

	var reply []string

	if strings.HasPrefix(partial, "-") {
		// Filter the flags that match the partial word
		for _, flag := range flags {
			if strings.HasPrefix(flag, partial) {
				reply = append(reply, flag)
			}
		}
	}

	args, err := shlex.Split(compLine)
	if err != nil {
		return
	}

	if len(args) <= 2 {
		reply = files(partial)
	}

	for _, word := range reply {
		fmt.Println(word)
	}

	os.Exit(0)
}

// log appends the given arguments to the log file.
func log(args ...interface{}) {
	file, err := os.OpenFile("complete.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	fmt.Fprintln(file, args...)
	file.Close()
}
