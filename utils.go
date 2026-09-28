package main

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/goccy/go-yaml"

	"github.com/antonmedv/fx/internal/engine"
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

func open(filePath string, flagYaml, flagToml, flagEdn *bool) *os.File {
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
	fileName := path.Base(filePath)
	hasYamlExt, _ := regexp.MatchString(`(?i)\.ya?ml$`, fileName)
	hasTomlExt, _ := regexp.MatchString(`(?i)\.toml$`, fileName)
	hasEdnExt, _ := regexp.MatchString(`(?i)\.edn$`, fileName)
	if !*flagYaml && hasYamlExt {
		*flagYaml = true
	}
	if !*flagToml && hasTomlExt {
		*flagToml = true
	}
	if !*flagEdn && hasEdnExt {
		*flagEdn = true
	}
	return f
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
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

func parseYAML(b []byte) ([]byte, error) {
	var out []byte
	decoder := yaml.NewDecoder(
		bytes.NewReader(b),
		yaml.UseOrderedMap(),
	)
	for {
		var v any
		if err := decoder.Decode(&v); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		j, err := yaml.MarshalWithOptions(v, yaml.JSON())
		if err != nil {
			return nil, err
		}
		out = append(out, j...)
	}
	return out, nil
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
