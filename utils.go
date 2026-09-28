package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/antonmedv/fx/internal/engine"
	"github.com/antonmedv/fx/internal/format"
	"github.com/antonmedv/fx/internal/jsonpath"
	"github.com/antonmedv/fx/internal/jsonx"
)

func lookup(names []string, defaultEditor string) string {
	for _, name := range names {
		env, ok := os.LookupEnv(name)
		if ok && env != "" {
			return env
		}
	}
	return defaultEditor
}

func open(filePath string) *os.File {
	f, err := os.Open(filePath)
	if err != nil {
		var pathError *fs.PathError
		if errors.As(err, &pathError) {
			println(err.Error())
			os.Exit(1)
		} else {
			panic(err)
		}
	}
	return f
}

// action is what a flag asks main to do instead of reading input.
type action int

const (
	actionRead action = iota // read the input
	actionHelp
	actionVersion
	actionThemes
	actionExportThemes
	actionGameOfLife
)

// parseFlags reads the command line. It sets the flag variables and returns
// the other arguments and the action asked for.
func parseFlags(argv []string) (args []string, act action, err error) {
	var formatArgs []string // distinct format flags, in order
	for _, arg := range argv {
		if strings.HasPrefix(arg, "--comp") {
			flagComp = true
			continue
		}
		if f := format.ByFlag(arg); f != nil {
			if inputFormat != f {
				formatArgs = append(formatArgs, arg)
			}
			inputFormat = f
			continue
		}
		switch arg {
		case "-h", "--help":
			return nil, actionHelp, nil
		case "-v", "-V", "--version":
			return nil, actionVersion, nil
		case "--themes":
			return nil, actionThemes, nil
		case "--export-themes":
			return nil, actionExportThemes, nil
		case "--game-of-life":
			return nil, actionGameOfLife, nil
		case "--raw", "-r":
			flagRaw = true
		case "--slurp", "-s":
			flagSlurp = true
		case "-rs", "-sr":
			flagRaw = true
			flagSlurp = true
		case "--strict":
			flagStrict = true
		case "--no-inline":
			flagNoInline = true
		default:
			args = append(args, arg)
		}
	}
	if len(formatArgs) > 1 {
		return nil, actionRead, fmt.Errorf("can't use %s and %s flags together", formatArgs[0], formatArgs[1])
	}
	if len(formatArgs) == 1 && flagRaw {
		return nil, actionRead, fmt.Errorf("can't use %s and --raw flags together", formatArgs[0])
	}
	return args, actionRead, nil
}

func regexCase(code string) (string, bool) {
	if strings.HasSuffix(code, "/i") {
		return code[:len(code)-2], true
	} else if strings.HasSuffix(code, "/") {
		return code[:len(code)-1], false
	} else {
		return code, true
	}
}

func flex(width int, a, b string) string {
	return a + strings.Repeat(" ", max(1, width-len(a)-len(b))) + b
}

func safeSlice(s string, start, end int) string {
	length := len(s)
	if start > length {
		start = length
	}
	if end > length {
		end = length
	}
	if start < 0 {
		start = 0
	}
	if end < 0 {
		end = 0
	}
	if start > end {
		start = end
	}
	return s[start:end]
}

func isRefNode(n *jsonx.Node) (string, bool) {
	if n.Kind == jsonx.String && len(n.Key) == 6 && string(n.Key) == `"$ref"` {
		value, err := strconv.Unquote(n.Value)
		if err == nil {
			_, ok := jsonpath.ParseSchemaRef(value)
			if ok {
				return value, true
			}
		}
	}
	return "", false
}

var errEmptyInput = errors.New("Empty input: expected JSON value.")

// nonEmptyParser reports empty input (or input of only whitespace) as an
// error instead of io.EOF, so fx exits non-zero on it.
type nonEmptyParser struct {
	engine.Parser
	parsed bool
}

func (p *nonEmptyParser) Parse() (*jsonx.Node, error) {
	node, err := p.Parser.Parse()
	if err == io.EOF && !p.parsed {
		return nil, errEmptyInput
	}
	if err == nil {
		p.parsed = true
	}
	return node, err
}

type inputSource int

const (
	inputUsage inputSource = iota
	inputFile
	inputStdin
)

var bareIdentifier = regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*$`)

// chooseInput decides where input comes from. The first argument is read as a
// file when it names an existing regular file, whatever stdin is. When stdin is
// a pipe or a regular file, a bare identifier (e.g. `keys`) stays code even if a
// file with that name exists, so pipelines don't change meaning with the cwd.
// Without a file argument, a TTY or char device stdin (e.g. /dev/null) keeps
// file mode, and piped stdin is read as input.
func chooseInput(stdinIsTty, stdinIsInput bool, args []string, stat func(string) (os.FileInfo, error)) inputSource {
	stdinMode := stdinIsInput && !stdinIsTty
	if len(args) > 0 {
		isFile := false
		if info, err := stat(args[0]); err == nil && info.Mode().IsRegular() {
			isFile = true
		}
		if isFile && !(stdinMode && bareIdentifier.MatchString(args[0])) {
			return inputFile
		}
	}
	if stdinMode {
		return inputStdin
	}
	if len(args) == 0 {
		return inputUsage
	}
	return inputFile
}
