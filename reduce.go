package main

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path"
)

//go:embed npm/index.js
var src []byte

func reduce(fns []string) {
	script := path.Join(os.TempDir(), fmt.Sprintf("fx-%v.js", version))
	_, err := os.Stat(script)
	if os.IsNotExist(err) {
		err := os.WriteFile(script, src, 0644)
		if err != nil {
			panic(err)
		}
	}

	deno := false
	bin, err := exec.LookPath("node")
	if err != nil {
		if err != nil {
			bin, err = exec.LookPath("deno")
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "Node.js or Deno is required to run fx with reducers.\n")
				os.Exit(1)
			}
			deno = true
		}
	}

	env := os.Environ()
	var args []string

	if deno {
		args = []string{"run", "-A", script}
		env = append(env, "V8_FLAGS=--max-old-space-size=16384")
	} else {
		args = []string{script}
		env = append(env, "NODE_OPTIONS=--max-old-space-size=16384")
	}

	args = append(args, fns...)

	cmd := exec.Command(bin, args...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()

	switch err := err.(type) {
	case nil:
		os.Exit(0)
	case *exec.ExitError:
		os.Exit(err.ExitCode())
	default:
		panic(err)
	}
}
