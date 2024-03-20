package utils

func IsHexDigit(ch byte) bool {
	return (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}

func IsDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}
