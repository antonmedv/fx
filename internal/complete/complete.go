package complete

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dop251/goja"
	"github.com/goccy/go-yaml"

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

var globals = []string{
	"JSON.stringify",
	"JSON.parse",
	"YAML.stringify",
	"YAML.parse",
	"Object.keys",
	"Object.values",
	"Object.entries",
	"Object.fromEntries",
	"Array.isArray",
	"Array.from",
	"len",
	"uniq",
	"sort",
	"map",
	"sortBy",
	"groupBy",
	"chunk",
	"zip",
	"flatten",
	"reverse",
	"keys",
	"values",
}

func Complete() bool {
	compLine, ok := os.LookupEnv("COMP_LINE")

	if ok && len(os.Args) == 3 {
		doComplete(compLine, os.Args[2])
		return true
	}

	compZsh, ok := os.LookupEnv("COMP_ZSH")
	if ok {
		doComplete(compZsh, lastWord(compZsh))
		return true
	}

	return false
}

func doComplete(compLine string, compWord string) {
	if strings.HasPrefix(compWord, "-") {
		compReply(filterReply(flags, compWord))
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
		fileComplete(compWord)
		return
	} else if len(args) == 2 {
		isSecondArgIsFile = isFile(args[1])
		if !isSecondArgIsFile {
			fileComplete(compWord)
			return
		}
	} else {
		isSecondArgIsFile = isFile(args[1])
	}

	if isSecondArgIsFile {
		if globalsComplete(compWord) {
			return
		}

		file := args[1]

		hasYamlExt, _ := regexp.MatchString(`(?i)\.ya?ml$`, file)
		if !flagYaml && hasYamlExt {
			flagYaml = true
		}

		input, err := os.ReadFile(file)
		if err != nil {
			return
		}

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

		codeComplete(string(input), args, compWord)
	}
}

func globalsComplete(compWord string) bool {
	if compWord == "" {
		// We will not complete globals if compWord is empty,
		// as we want to show only object keys.
		return false
	}
	reply := filterReply(globals, compWord)
	if len(reply) > 0 {
		compReply(reply)
		return true
	}
	return false
}

func codeComplete(input string, args []string, compWord string) {
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
	code.WriteString(fmt.Sprintf("let json = %s\n", input))
	for _, arg := range args {
		if arg == "" {
			continue
		}
		code.WriteString(Transform(arg))
	}
	code.WriteString("\n__keys\n")

	out, err := goja.New().RunString(code.String())
	if err != nil {
		return
	}

	if array, ok := out.Export().([]interface{}); ok {
		prefix := dropTail(compWord)
		var reply []string
		for _, key := range array {
			reply = append(reply, prefix+key.(string))
		}
		compReply(filterReply(reply, compWord))
	}
}

func filterArgs(args []string) []string {
	filtered := make([]string, 0, len(args))
	for _, arg := range args {
		found := false
		for _, flag := range flags {
			if arg == flag {
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

func fileComplete(compWord string) {
	var matches []string

	dir, filePrefix := filepath.Split(compWord)
	if dir == "" {
		dir = "."
	}
	pattern := filepath.Join(dir, filePrefix+"*")
	files, err := filepath.Glob(pattern)
	if err != nil {
		fmt.Println("Error reading directory:", err)
		return
	}

	for _, match := range files {
		relativePath, err := filepath.Rel(".", match)
		if err != nil {
			continue
		}
		matches = append(matches, relativePath)
	}

	compReply(matches)
}
