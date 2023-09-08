package main

func isHexDigit(ch byte) bool {
	return (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

func colorForValue(b []byte) color {
	if len(b) == 0 {
		return noColor
	}

	switch b[0] {
	case '"':
		return currentTheme.String
	case 't', 'f':
		return currentTheme.Boolean
	case 'n':
		return currentTheme.Null
	case '{', '[', '}', ']':
		return currentTheme.Syntax
	default:
		if isDigit(b[0]) || b[0] == '-' {
			return currentTheme.Number
		}
		return noColor
	}
}
