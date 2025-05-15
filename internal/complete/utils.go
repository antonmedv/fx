package complete

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func compReply(reply []pair, withDisplay bool) {
	var lines []string
	for _, line := range reply {
		if withDisplay {
			lines = append(lines, fmt.Sprintf("%s\t%s", line.display, line.value))
		} else {
			lines = append(lines, line.value)
		}
	}
	fmt.Print(strings.Join(lines, "\n"))
}

func filterReply(reply []pair, compWord string) []pair {
	var filtered []pair
	for _, word := range reply {
		if strings.HasPrefix(word.value, compWord) {
			filtered = append(filtered, word)
		}
	}
	return filtered
}

func isFile(path string) bool {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(home, path[1:])
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func dropTail(s string) string {
	parts := strings.Split(s, ".")
	return strings.Join(parts[:len(parts)-1], ".")
}

func balanceBrackets(code string) string {
	var stack []rune
	brackets := map[rune]rune{')': '(', '}': '{', ']': '['}
	reverseBrackets := map[rune]rune{'(': ')', '{': '}', '[': ']'}

	for _, char := range code {
		switch char {
		case '(', '{', '[':
			stack = append(stack, char)
		case ')', '}', ']':
			if len(stack) > 0 && brackets[char] == stack[len(stack)-1] {
				stack = stack[:len(stack)-1] // Pop
			}
		}
	}

	for i := len(stack) - 1; i >= 0; i-- {
		code += string(reverseBrackets[stack[i]])
	}

	return code
}

func lastWord(line string) string {
	words := strings.Split(line, " ")
	var s string
	if len(words) > 0 {
		s = words[len(words)-1]
	}
	return s
}

func debug(args ...interface{}) {
	file, err := os.OpenFile("complete.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	_, _ = fmt.Fprintln(file, args...)
	_ = file.Close()
}
