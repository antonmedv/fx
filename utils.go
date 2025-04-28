package main

import (
	"bytes"
	"io"
	"strings"

	"github.com/goccy/go-yaml"
)

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

func safeSlice(b []byte, start, end int) []byte {
	length := len(b)
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
	return b[start:end]
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
