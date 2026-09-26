package engine

import (
	"strings"
	"unicode"
)

// SplitArgs splits an interactive query into arguments, like `fx @.name len`
// on the command line, but without shell quoting. Arguments are separated by
// whitespace outside of JS strings, template literals, regexps and brackets.
// Whitespace around an operator does not separate, so `x => x.a + 1` and
// `.a ?? 0` stay single arguments. Unterminated strings or brackets extend to
// the end of the query.
func SplitArgs(query string) []string {
	var args []string
	start := -1 // start of the current argument
	last := -1  // start of the last token of the current argument
	i := 0
	for i < len(query) {
		c := query[i]
		if isSpace(c) {
			j := i
			for j < len(query) && isSpace(query[j]) {
				j++
			}
			if start >= 0 && (j == len(query) || !continues(query[start:i], last-start, query[j:])) {
				args = append(args, query[start:i])
				start = -1
			}
			i = j
			continue
		}
		if start < 0 {
			start = i
		}
		last = i
		i = skipToken(query, i, query[start:i])
	}
	if start >= 0 {
		args = append(args, query[start:])
	}
	return args
}

// skipToken returns the index after the token starting at s[i]: a string,
// template, regexp or bracketed group as a whole, otherwise one byte.
// before is the code preceding the token, to tell a regexp from a division.
func skipToken(s string, i int, before string) int {
	switch c := s[i]; c {
	case '"', '\'':
		return skipString(s, i)
	case '`':
		return skipTemplate(s, i)
	case '(', '[', '{':
		return skipGroup(s, i+1, closer(c))
	case '/':
		if regexAllowed(before) {
			return skipRegex(s, i)
		}
	}
	return i + 1
}

func closer(c byte) byte {
	switch c {
	case '(':
		return ')'
	case '[':
		return ']'
	}
	return '}'
}

// skipGroup returns the index after the closing bracket end, starting inside
// the group at s[i].
func skipGroup(s string, i int, end byte) int {
	if j := findClose(s, i, end); j >= 0 {
		return j + 1
	}
	return len(s)
}

// findClose returns the index of the closing bracket end, starting inside
// the group at s[i], or -1 if the group is unterminated.
func findClose(s string, i int, end byte) int {
	start := i
	for i < len(s) {
		if s[i] == end {
			return i
		}
		i = skipToken(s, i, s[start:i])
	}
	return -1
}

func skipString(s string, i int) int {
	quote := s[i]
	for i++; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++
		case quote:
			return i + 1
		}
	}
	return i
}

func skipTemplate(s string, i int) int {
	for i++; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++
		case '`':
			return i + 1
		case '$':
			if i+1 < len(s) && s[i+1] == '{' {
				i = skipGroup(s, i+2, '}') - 1
			}
		}
	}
	return i
}

func skipRegex(s string, i int) int {
	inClass := false
	for i++; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++
		case '[':
			inClass = true
		case ']':
			inClass = false
		case '/':
			if !inClass {
				return i + 1
			}
		}
	}
	return i
}

// regexAllowed reports whether a slash after the code before starts a regexp
// rather than being a division.
func regexAllowed(before string) bool {
	before = strings.TrimRightFunc(before, unicode.IsSpace)
	if before == "" {
		return true
	}
	if strings.ContainsRune("(,=:[!&|?{};+-*%<>~^", rune(before[len(before)-1])) {
		return true
	}
	return isKeyword(trailingWord(before))
}

// continues reports whether the code after whitespace continues the argument
// arg, rather than starting a new argument. last is the start of the last
// token of arg.
func continues(arg string, last int, after string) bool {
	token := arg[last:]
	if len(token) == 1 && strings.IndexByte("+-*/%=<>!&|^~?:,", token[0]) >= 0 &&
		!strings.HasSuffix(arg, "++") && !strings.HasSuffix(arg, "--") {
		// A binary or unary operator, not a postfix increment.
		return true
	}
	if isKeyword(trailingWord(arg)) {
		return true
	}
	switch after[0] {
	case '+', '-', '*', '/', '%', '=', '<', '>', '&', '|', '^', ':', ',':
		return true
	case '!':
		return strings.HasPrefix(after, "!=")
	case '?':
		// `a ? b : c` and `a ?? b`, but `?.active` is a filter argument.
		return len(after) == 1 || after[1] == '?' || isSpace(after[1])
	case '{':
		// The body of `function (x) { ... }` or `if (x) { ... }`, but not an
		// object literal after a call like `.map(f) {a: 1}`.
		return token[0] == '(' && isBlockHead(strings.TrimRightFunc(arg[:last], unicode.IsSpace))
	}
	switch leadingWord(after) {
	case "in", "instanceof", "else", "catch", "finally":
		return true
	}
	return false
}

// isBlockHead reports whether code followed by `(...)` starts a block, as in
// `function (x)`, `function f(x)` or `if (x)`.
func isBlockHead(code string) bool {
	word := trailingWord(code)
	switch word {
	case "function", "if", "for", "while", "catch", "switch":
		return true
	case "":
		return false
	}
	return trailingWord(strings.TrimRightFunc(code[:len(code)-len(word)], unicode.IsSpace)) == "function"
}

func isKeyword(word string) bool {
	switch word {
	case "typeof", "new", "void", "delete", "await", "in", "of", "instanceof",
		"return", "yield", "throw", "function", "async", "const", "let", "var",
		"if", "else", "case", "do":
		return true
	}
	return false
}

// trailingWord returns the identifier at the end of code, or "" if it is a
// property name like `.in`.
func trailingWord(code string) string {
	i := len(code)
	for i > 0 && isIdent(code[i-1]) {
		i--
	}
	if i > 0 && code[i-1] == '.' {
		return ""
	}
	return code[i:]
}

func leadingWord(code string) string {
	i := 0
	for i < len(code) && isIdent(code[i]) {
		i++
	}
	return code[:i]
}

func isIdent(c byte) bool {
	return c == '_' || c == '$' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}
