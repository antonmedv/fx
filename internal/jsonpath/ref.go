package jsonpath

import (
	"net/url"
	"strings"
)

func ParseSchemaRef(ref string) ([]any, bool) {
	// Must start with '#'
	if len(ref) == 0 || ref[0] != '#' {
		return nil, false
	}

	// An empty fragment refers to the whole document
	if ref == "#" {
		return []any{}, true
	}

	// Must be a pointer ("#/...")
	if !strings.HasPrefix(ref, "#/") {
		return nil, false
	}

	// Split the pointer without the leading '#/'
	parts := strings.Split(ref[2:], "/")
	out := make([]any, len(parts))
	for i, part := range parts {
		// JSON Pointer unescaping
		s := strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")
		// Percent-unescape
		unescaped, err := url.PathUnescape(s)
		if err != nil {
			return nil, false
		}
		out[i] = unescaped
	}
	return out, true
}
