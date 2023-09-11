package main

func isHexDigit(ch byte) bool {
	return (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
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
