package complete

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dop251/goja"
	"github.com/goccy/go-yaml"

	"github.com/antonmedv/fx/internal/engine"
	"github.com/antonmedv/fx/internal/shlex"
)

type pair struct {
	display string
	value   string
}

var flags = []pair{
	{"--help", "--help"},
	{"--raw", "--raw"},
	{"--slurp", "--slurp"},
	{"--themes", "--themes"},
	{"--version", "--version"},
	{"--yaml", "--yaml"},
	{"--strict", "--strict"},
}

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
		compReply(filterReply(flags, compWord), withDisplay)
		return
	}

	args, err := shlex.Split(compLine)
	if err != nil {
		return
	}

	compWord = shlex.Parse(compWord)

	var flagYaml bool
	for _, arg := range args {
		if arg == "--yaml" {
			flagYaml = true
			break
		}
	}

	// Remove flags from args.
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

	var reply []pair

	if isSecondArgIsFile {
		file := args[1]

		hasYamlExt, _ := regexp.MatchString(`(?i)\.ya?ml$`, file)
		if !flagYaml && hasYamlExt {
			flagYaml = true
		}

		if strings.HasPrefix(file, "~") {
			home, err := os.UserHomeDir()
			if err == nil {
				file = filepath.Join(home, file[1:])
			}
		}

		input, err := os.ReadFile(file)
		if err != nil {
			return
		}

		input = append(input, '\n')

		// If input is bigger than 100MB, skip completion.
		if len(input) > 100*1024*1024 {
			return
		}

		if flagYaml {
			input, err = yaml.YAMLToJSON(input)
			if err != nil {
				return
			}
		}

		reply = append(reply, keysComplete(input, args, compWord)...)
	}

	reply = filterReply(reply, compWord)
	if len(reply) > 0 {
		compReply(reply, withDisplay)
		return
	}

	if len(compWord) > 0 {
		// Only show globals if compWord is not empty,
		// as we do not want to be very verbose and show all globals.
		compReply(filterReply(globalsComplete(), compWord), withDisplay)
	}
}

func globalsComplete() []pair {
	var code strings.Builder
	code.WriteString(prelude)
	code.WriteString(engine.Stdlib)
	code.WriteString("\n__autocomplete()\n")

	vm := goja.New()
	value, err := vm.RunString(code.String())
	if err != nil {
		return nil
	}

	if array, ok := value.Export().([]any); ok {
		var reply []pair
		for _, key := range array {
			reply = append(reply, pair{
				display: key.(string),
				value:   key.(string),
			})
		}
		return reply
	}
	return nil
}

func keysComplete(input []byte, args []string, compWord string) []pair {
	args = args[2:] // Drop binary & file from the args.

	if compWord == "" {
		args = append(args, ".__keys()")
	} else {
		if len(args) > 0 {
			last := args[len(args)-1]
			last = dropTail(args[len(args)-1])
			last = last + ".__keys()"
			last = balanceBrackets(last)
			args[len(args)-1] = last
		}
	}

	var code strings.Builder
	code.WriteString(prelude)
	code.WriteString(engine.Stdlib)
	code.WriteString("let json = ")
	code.Write(input)
	for i, arg := range args {
		if arg == "" { // After dropTail, we can have empty strings.
			continue
		}
		code.WriteString(engine.Transpile(args, i))
	}
	code.WriteString("\n__keys\n")

	vm := goja.New()
	value, err := vm.RunString(code.String())
	if err != nil {
		return nil
	}

	if array, ok := value.Export().([]interface{}); ok {
		prefix := dropTail(compWord)
		var reply []pair
		for _, key := range array {
			reply = append(reply, pair{
				display: "." + key.(string),
				value:   join(prefix, key.(string)),
			})
		}
		return reply
	}
	return nil
}

var alphaRe = regexp.MustCompile(`^[\w$]+$`)

func join(prefix, key string) string {
	if alphaRe.MatchString(key) {
		return prefix + "." + key
	} else {
		if prefix == "" {
			return fmt.Sprintf(".[%q]", key)
		}
		return fmt.Sprintf("%s[%q]", prefix, key)
	}
}

func filterArgs(args []string) []string {
	filtered := make([]string, 0, len(args))
	for _, arg := range args {
		found := false
		for _, flag := range flags {
			if arg == flag.value {
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

func fileComplete(compWord string) []pair {
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
	var matches []pair
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

		matches = append(matches, pair{
			display: filepath.Base(suggestion) + dirSuffix,
			value:   suggestion,
		})
	}

	return matches
}
