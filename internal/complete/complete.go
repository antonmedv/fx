package complete

import (
	_ "embed"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/antonmedv/fx/internal/format"
	"github.com/antonmedv/fx/internal/jsonx"
	"github.com/antonmedv/fx/internal/shlex"
)

type Reply struct {
	Display string
	Value   string
	Type    string // "file" for files, others optional
}

var Flags []Reply

//go:embed complete.bash
var Bash string

//go:embed complete.zsh
var Zsh string

//go:embed complete.fish
var Fish string

//go:embed prelude.js
var prelude string

func Complete() bool {
	compLine, ok := os.LookupEnv("COMP_LINE")
	if ok && len(os.Args) >= 3 {
		doComplete(compLine, os.Args[2], false)
		return true
	}

	compZsh, ok := os.LookupEnv("COMP_ZSH")
	if ok {
		doComplete(compZsh, lastWord(compZsh), true)
		return true
	}

	compFish, ok := os.LookupEnv("COMP_FISH")
	if ok {
		doComplete(compFish, lastWord(compFish), false)
		return true
	}

	return false
}

func doComplete(compLine string, compWord string, withDisplay bool) {
	if strings.HasPrefix(compWord, "-") {
		compReply(filterReply(Flags, compWord), withDisplay)
		return
	}

	args, err := shlex.Split(compLine)
	if err != nil {
		return
	}

	compWord = shlex.Parse(compWord)

	var inputFormat *format.Format
	for _, arg := range args {
		if f := format.ByFlag(arg); f != nil {
			inputFormat = f
		}
	}

	// Remove Flags from args.
	args = filterArgs(args)

	isSecondArgIsFile := false
	if len(args) == 0 {
		return
	} else if len(args) == 1 {
		reply := fileComplete(compWord)
		compReply(reply, withDisplay)
		return
	} else if len(args) == 2 {
		isSecondArgIsFile = isFile(args[1])
		if !isSecondArgIsFile {
			reply := fileComplete(compWord)
			compReply(reply, withDisplay)
			return
		}
	} else {
		isSecondArgIsFile = isFile(args[1])
	}

	var reply []Reply

	if isSecondArgIsFile {
		file := args[1]

		inputFormat = format.Choose(inputFormat, file)

		if strings.HasPrefix(file, "~") {
			home, err := os.UserHomeDir()
			if err == nil {
				file = filepath.Join(home, file[1:])
			}
		}

		resultCh := make(chan []Reply, 1)

		go func() {
			input, err := os.ReadFile(file)
			if err != nil {
				resultCh <- []Reply{}
				return
			}

			if inputFormat != nil {
				input, err = inputFormat.ToJSON(input)
				if err != nil {
					resultCh <- []Reply{}
					return
				}
			}

			node, err := jsonx.Parse(input)
			if err != nil {
				resultCh <- []Reply{}
				return
			}

			resultCh <- KeysComplete(node, args, compWord)
		}()

		select {
		case result := <-resultCh:
			reply = append(reply, result...)
		case <-time.After(3 * time.Second):
			return
		}
	} else {
		reply = KeysComplete(nil, args, compWord) // Globals only.
	}

	// Replies already match the word; filtering by prefix would drop the
	// rewrite of `.items.` to `.items[].key`.
	compReply(reply, withDisplay)
}

// KeysComplete completes compWord, the last of the command line args
// (fx, the file, then the query), on input.
func KeysComplete(input *jsonx.Node, args []string, compWord string) []Reply {
	args = args[2:] // Drop binary & file from the args.
	if compWord != "" && len(args) > 0 {
		args = args[:len(args)-1]
	}
	return Replies(NewRequest(args, compWord), Docs{First: input})
}

func filterArgs(args []string) []string {
	filtered := make([]string, 0, len(args))
	for _, arg := range args {
		found := false
		for _, flag := range Flags {
			if arg == flag.Value {
				found = true
				break
			}
		}
		if !found {
			filtered = append(filtered, arg)
		}
	}
	return filtered
}

func fileComplete(compWord string) []Reply {
	original := compWord

	// Step 1: Expand ~ to home directory
	if strings.HasPrefix(compWord, "~") {
		if compWord == "~" || strings.HasPrefix(compWord, "~/") {
			home, err := os.UserHomeDir()
			if err == nil {
				compWord = filepath.Join(home, compWord[1:])
			}
		} else {
			// We don't support ~username completion
			return nil
		}
	}

	// Step 2: If compWord ends in "/", treat it as a directory and add a "*" pattern
	info, err := os.Stat(compWord)
	if err == nil && info.IsDir() && !strings.HasSuffix(compWord, "*") {
		compWord = filepath.Join(compWord, "*")
	} else if !strings.HasSuffix(compWord, "*") {
		// Add wildcard if not already present
		compWord = compWord + "*"
	}

	// Step 3: Perform globbing
	files, err := filepath.Glob(compWord)
	if err != nil {
		return nil
	}

	// Step 4: Format matches
	var matches []Reply
	for _, match := range files {
		if match == "." || match == ".." {
			continue
		}

		var suggestion string
		if strings.HasPrefix(original, "~") {
			home, _ := os.UserHomeDir()
			if strings.HasPrefix(match, home) {
				suggestion = "~" + strings.TrimPrefix(match, home)
			} else {
				suggestion = match
			}
		} else if filepath.IsAbs(original) {
			suggestion = match
		} else {
			rel, err := filepath.Rel(".", match)
			if err != nil {
				continue
			}
			suggestion = rel
		}

		dirSuffix := ""
		info, err := os.Stat(match)
		if err == nil {
			if info.IsDir() {
				dirSuffix = "/"
			}
		}

		matches = append(matches, Reply{
			Display: filepath.Base(suggestion) + dirSuffix,
			Value:   suggestion,
			Type:    "file",
		})
	}

	return matches
}
