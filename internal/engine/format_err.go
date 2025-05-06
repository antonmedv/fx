package engine

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/x/term"
	"github.com/mattn/go-runewidth"
)

func formatErr(args []string, i int, jsCode string) string {
	width, _, err := term.GetSize(os.Stdout.Fd())
	if err != nil || width <= 0 {
		width = 80
	}

	if i < 0 || i >= len(args) {
		return fmt.Sprintf("Invalid argument index: %d", i)
	}
	code := args[i]

	const indentCols = 2 // we print "  " before everything
	const sepCols = 1    // the single space between pre and code
	reserve := indentCols + sepCols + runewidth.StringWidth(code) + sepCols

	available := width - reserve
	if available < 0 {
		available = 0
	}
	maxCtx := available / 2

	pre := strings.Join(args[:i], " ")
	post := strings.Join(args[i+1:], " ")

	pre = trimLeft(pre, maxCtx)
	post = trimRight(post, maxCtx)

	leftSep := 0
	if pre != "" {
		leftSep = sepCols
	}
	spacerCols := indentCols + runewidth.StringWidth(pre) + leftSep
	spacer := strings.Repeat(" ", spacerCols)

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat(" ", indentCols))
	if pre != "" {
		sb.WriteString(pre)
		sb.WriteByte(' ')
	}
	sb.WriteString(code)
	if post != "" {
		sb.WriteByte(' ')
		sb.WriteString(post)
	}
	sb.WriteByte('\n')

	sb.WriteString(spacer)
	sb.WriteString(strings.Repeat("^", runewidth.StringWidth(code)))
	sb.WriteByte('\n')

	if jsCode != "" && jsCode != code {
		snippet := jsCode
		if runewidth.StringWidth(snippet) > width {
			snippet = trimRight(snippet, width)
		}
		sb.WriteByte('\n')
		sb.WriteString(snippet)
		sb.WriteByte('\n')
	}

	sb.WriteString("\n")

	return sb.String()
}

func trimLeft(s string, ctx int) string {
	if runewidth.StringWidth(s) <= ctx {
		return s
	}
	rs := []rune(s)
	widthAccum := 0
	var out []rune
	for i := len(rs) - 1; i >= 0; i-- {
		w := runewidth.RuneWidth(rs[i])
		if widthAccum+w > ctx-1 {
			break
		}
		widthAccum += w
		out = append([]rune{rs[i]}, out...)
	}
	return "…" + string(out)
}

func trimRight(s string, ctx int) string {
	if runewidth.StringWidth(s) <= ctx {
		return s
	}
	rs := []rune(s)
	widthAccum := 0
	var out []rune
	for _, r := range rs {
		w := runewidth.RuneWidth(r)
		if widthAccum+w > ctx-1 {
			break
		}
		out = append(out, r)
		widthAccum += w
	}
	return string(out) + "…"
}
