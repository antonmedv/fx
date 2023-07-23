package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"runtime/pprof"
	"strings"

	. "github.com/antonmedv/fx/pkg/json"
	"github.com/antonmedv/fx/pkg/model"
	. "github.com/antonmedv/fx/pkg/reducer"
	. "github.com/antonmedv/fx/pkg/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-isatty"
	"github.com/muesli/termenv"
)

var (
	flagHelp      bool
	flagVersion   bool
	flagPrintCode bool
)

func main() {
	var args []string
	for _, arg := range os.Args[1:] {
		switch arg {
		case "-h", "--help":
			flagHelp = true
		case "-v", "-V", "--version":
			flagVersion = true
		case "--print-code":
			flagPrintCode = true
		default:
			args = append(args, arg)
		}
	}
	if flagHelp {
		fmt.Println(usage(model.DefaultKeyMap()))
		return
	}
	if flagVersion {
		fmt.Println(version)
		return
	}
	cpuProfile := os.Getenv("CPU_PROFILE")
	if cpuProfile != "" {
		f, err := os.Create(cpuProfile)
		if err != nil {
			panic(err)
		}
		err = pprof.StartCPUProfile(f)
		if err != nil {
			panic(err)
		}
	}
	themeId, ok := os.LookupEnv("FX_THEME")
	if !ok {
		themeId = "1"
	}
	theme, ok := Themes[themeId]
	if !ok {
		theme = Themes["1"]
	}
	if termenv.ColorProfile() == termenv.Ascii {
		theme = Themes["0"]
	}
	var showSize bool
	if s, ok := os.LookupEnv("FX_SHOW_SIZE"); ok {
		if s == "true" {
			showSize = true
		}
	}

	stdinIsTty := isatty.IsTerminal(os.Stdin.Fd())
	stdoutIsTty := isatty.IsTerminal(os.Stdout.Fd())
	filePath := ""
	fileName := ""
	var dec *json.Decoder
	if stdinIsTty {
		// Nothing was piped, maybe file argument?
		if len(args) >= 1 {
			filePath = args[0]
			f, err := os.Open(filePath)
			if err != nil {
				switch err.(type) {
				case *fs.PathError:
					fmt.Println(err)
					os.Exit(1)
				default:
					panic(err)
				}
			}
			fileName = path.Base(filePath)
			dec = json.NewDecoder(f)
			args = args[1:]
		}
	} else {
		dec = json.NewDecoder(os.Stdin)
	}
	if dec == nil {
		fmt.Println(usage(model.DefaultKeyMap()))
		os.Exit(1)
	}
	dec.UseNumber()
	object, err := Parse(dec)
	if err != nil {
		fmt.Println("JSON Parse Error:", err.Error())
		os.Exit(1)
	}
	lang, ok := os.LookupEnv("FX_LANG")
	if !ok {
		lang = "js"
	}
	var fxrc string
	if lang == "js" || lang == "node" {
		home, err := os.UserHomeDir()
		if err == nil {
			b, err := os.ReadFile(path.Join(home, ".fxrc.js"))
			if err == nil {
				fxrc = "\n" + string(b)
				if lang == "js" {
					parts := strings.SplitN(fxrc, "// nodejs:", 2)
					fxrc = parts[0]
				}
			}
		}
	}
	if dec.More() {
		os.Exit(stream(dec, object, lang, args, theme, fxrc))
	}
	if len(args) > 0 || !stdoutIsTty {
		if len(args) > 0 && flagPrintCode {
			fmt.Print(GenerateCode(lang, args, fxrc))
			return
		}
		if lang == "js" {
			simplePath, ok := SplitSimplePath(args)
			if ok {
				output := GetBySimplePath(object, simplePath)
				Echo(output, theme)
				os.Exit(0)
			}
			vm, fn, err := CreateJS(args, fxrc)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			os.Exit(ReduceJS(vm, fn, object, theme))
		} else {
			os.Exit(Reduce(object, lang, args, theme, fxrc))
		}
	}

	// Start interactive mode.
	m := model.New(object, model.Config{
		FileName: fileName,
		Theme:    theme,
		ShowSize: showSize,
	})

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if err := p.Start(); err != nil {
		panic(err)
	}
	if cpuProfile != "" {
		pprof.StopCPUProfile()
	}
	os.Exit(m.ExitCode())
}
