package main

import (
	"regexp"
	"strings"
)

var identifier = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func isHexDigit(ch byte) bool {
	return (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

func max(i, j int) int {
	if i > j {
		return i
	}
	return j
}

func prettyPrint(b []byte, selected bool, isChunk bool) []byte {
	if len(b) == 0 {
		return b
	}
	if selected {
		return currentTheme.Cursor(b)
	} else {
		if isChunk {
			return currentTheme.String(b)
		}
		switch b[0] {
		case '"':
			return currentTheme.String(b)
		case 't', 'f':
			return currentTheme.Boolean(b)
		case 'n':
			return currentTheme.Null(b)
		case '{', '[', '}', ']':
			return currentTheme.Syntax(b)
		default:
			if isDigit(b[0]) || b[0] == '-' {
				return currentTheme.Number(b)
			}
			return noColor(b)
		}
	}
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
