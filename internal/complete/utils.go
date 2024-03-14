package complete

import (
	"fmt"
	"os"
	"strings"
)

func compReply(reply []string) {
	fmt.Print(strings.Join(reply, "\n"))
}

func filterReply(reply []string, compWord string) []string {
	var filtered []string
	for _, word := range reply {
		if strings.HasPrefix(word, compWord) {
			filtered = append(filtered, word)
		}
	}
	return filtered
}

func isFile(path string) bool {
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
