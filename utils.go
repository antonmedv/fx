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

	"github.com/antonmedv/fx/internal/jsonpath"
	"github.com/antonmedv/fx/internal/jsonx"
)

func open(filePath string, flagYaml *bool) *os.File {
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
	if !*flagYaml && hasYamlExt {
		*flagYaml = true
	}
	return f
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
