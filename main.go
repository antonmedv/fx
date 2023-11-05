package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"runtime/pprof"

	"github.com/antonmedv/fx/pkg/model"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-isatty"
)

var (
	flagHelp    bool
	flagVersion bool
)

func main() {
	if _, ok := os.LookupEnv("FX_PPROF"); ok {
		f, err := os.Create("cpu.prof")
		if err != nil {
			panic(err)
		}
		err = pprof.StartCPUProfile(f)
		if err != nil {
			panic(err)
		}
		defer pprof.StopCPUProfile()
		memProf, err := os.Create("mem.prof")
		if err != nil {
			panic(err)
		}
		defer pprof.WriteHeapProfile(memProf)
	}

	var args []string
	for _, arg := range os.Args[1:] {
		switch arg {
		case "-h", "--help":
			flagHelp = true
		case "-v", "-V", "--version":
			flagVersion = true
		case "--themes":
			model.ThemeTester()
			return
		case "--export-themes":
			model.ExportThemes()
			return
		default:
			args = append(args, arg)
		}
	}

	if flagHelp {
		fmt.Println(usage(model.GetKeyMap()))
		return
	}

	if flagVersion {
		fmt.Println(version)
		return
	}

	stdinIsTty := isatty.IsTerminal(os.Stdin.Fd())
	var fileName string
	var src io.Reader

	if stdinIsTty && len(args) == 0 {
		fmt.Println(usage(model.GetKeyMap()))
		return
	} else if stdinIsTty && len(args) == 1 {
		filePath := args[0]
		f, err := os.Open(filePath)
		if err != nil {
			var pathError *fs.PathError
			if errors.As(err, &pathError) {
				fmt.Println(err)
				os.Exit(1)
			} else {
				panic(err)
			}
		}
		fileName = path.Base(filePath)
		src = f
	} else if !stdinIsTty && len(args) == 0 {
		src = os.Stdin
	} else {
		reduce(args)
		return
	}

	m, err := model.New(model.Config{
		Source:   src,
		FileName: fileName,
	})
	if err != nil {
		fmt.Print(err.Error())
		os.Exit(1)
		return
	}

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err = p.Run()
	if err != nil {
		panic(err)
	}
}
