package main

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"strings"
)

func (m *model) print(v interface{}, level, lineNumber, keyEndPos int, path string, selectableValues bool) []string {
	ident := strings.Repeat("  ", level)
	subident := strings.Repeat("  ", level-1)
	connect := func(path string, lineNumber int) {
		m.pathToLineNumber.set(path, lineNumber)
		m.lineNumberToPath[lineNumber] = path
	}
	sri := m.searchResultsIndex[path]

	switch v.(type) {
	case nil:
		return []string{merge(m.explode("null", sri.value, colors.null, path, selectableValues))}

	case bool:
		if v.(bool) {
			return []string{merge(m.explode("true", sri.value, colors.boolean, path, selectableValues))}
		} else {
			return []string{merge(m.explode("false", sri.value, colors.boolean, path, selectableValues))}
		}

	case number:
		return []string{merge(m.explode(v.(number).String(), sri.value, colors.number, path, selectableValues))}

	case string:
		line := fmt.Sprintf("%q", v)
		chunks := m.explode(line, sri.value, colors.string, path, selectableValues)
		if m.wrap && keyEndPos+width(line) > m.width {
			return wrapLines(chunks, keyEndPos, m.width, subident)
		}
		// No wrap
		return []string{merge(chunks)}

	case *dict:
		connect(path, lineNumber)
		if !m.expandedPaths[path] {
			return []string{m.preview(v, path, selectableValues)}
		}
		output := []string{m.printOpenBracket("{", sri, path, selectableValues)}
		lineNumber++ // bracket is on separate line
		keys := v.(*dict).keys
		for i, k := range keys {
			subpath := path + "." + k
			s := m.searchResultsIndex[subpath]
			connect(subpath, lineNumber)
			key := fmt.Sprintf("%q", k)
			key = merge(m.explode(key, s.key, colors.key, subpath, true))
			value, _ := v.(*dict).get(k)
			delim := m.printDelim(": ", s)
			keyEndPos := width(ident) + width(key) + width(delim)
			lines := m.print(value, level+1, lineNumber, keyEndPos, subpath, false)
			lines[0] = ident + key + delim + lines[0]
			if i < len(keys)-1 {
				lines[len(lines)-1] += m.printComma(",", s)
			}
			output = append(output, lines...)
			lineNumber += len(lines)
		}
		output = append(output, subident+m.printCloseBracket("}", sri, path, false))
		return output

	case array:
		connect(path, lineNumber)
		if !m.expandedPaths[path] {
			return []string{m.preview(v, path, selectableValues)}
		}
		output := []string{m.printOpenBracket("[", sri, path, selectableValues)}
		lineNumber++ // bracket is on separate line
		slice := v.(array)
		for i, value := range slice {
			subpath := fmt.Sprintf("%v[%v]", path, i)
			s := m.searchResultsIndex[subpath]
			connect(subpath, lineNumber)
			lines := m.print(value, level+1, lineNumber, width(ident), subpath, true)
			lines[0] = ident + lines[0]
			if i < len(slice)-1 {
				lines[len(lines)-1] += m.printComma(",", s)
			}
			lineNumber += len(lines)
			output = append(output, lines...)
		}
		output = append(output, subident+m.printCloseBracket("]", sri, path, false))
		return output

	default:
		return []string{"unknown type"}
	}
}

func (m *model) preview(v interface{}, path string, selectableValues bool) string {
	searchResult := m.searchResultsIndex[path]
	previewStyle := colors.preview
	if selectableValues && m.cursorPath() == path {
		previewStyle = colors.cursor
	}
	printValue := func(value interface{}) string {
		switch value.(type) {
		case nil, bool, number:
			return previewStyle.Render(fmt.Sprintf("%v", value))
		case string:
			return previewStyle.Render(fmt.Sprintf("%q", value))
		case *dict:
			return previewStyle.Render("{\u2026}")
		case array:
			return previewStyle.Render("[\u2026]")
		}
		return "..."
	}

	switch v.(type) {
	case *dict:
		output := m.printOpenBracket("{", searchResult, path, selectableValues)
		keys := v.(*dict).keys
		for _, k := range keys {
			key := fmt.Sprintf("%q", k)
			output += previewStyle.Render(key + ": ")
			value, _ := v.(*dict).get(k)
			output += printValue(value)
			break
		}
		if len(keys) > 1 {
			output += previewStyle.Render(", \u2026")
		}
		output += m.printCloseBracket("}", searchResult, path, selectableValues)
		return output

	case array:
		output := m.printOpenBracket("[", searchResult, path, selectableValues)
		slice := v.(array)
		for _, value := range slice {
			output += printValue(value)
			break
		}
		if len(slice) > 1 {
			output += previewStyle.Render(", \u2026")
		}
		output += m.printCloseBracket("]", searchResult, path, selectableValues)
		return output
	}
	return "?"
}

func wrapLines(chunks []withStyle, keyEndPos, mWidth int, subident string) []string {
	wrappedLines := make([]string, 0)
	currentLine := ""
	ident := ""      // First line stays on the same line with a "key",
	pos := keyEndPos // so no ident is needed. Start counting from the "key" offset.
	for _, chunk := range chunks {
		buffer := ""
		for _, ch := range chunk.value {
			buffer += string(ch)
			if pos == mWidth-1 {
				wrappedLines = append(wrappedLines, ident+currentLine+chunk.Render(buffer))
				currentLine = ""
				buffer = ""
				pos = width(subident) // Start counting from ident.
				ident = subident      // After first line, add ident to all.
			} else {
				pos++
			}
		}
		currentLine += chunk.Render(buffer)
	}
	if width(currentLine) > 0 {
		wrappedLines = append(wrappedLines, subident+currentLine)
	}
	return wrappedLines
}

func (w withStyle) Render(s string) string {
	return w.style.Render(s)
}

func (m *model) printOpenBracket(line string, s searchResultGroup, path string, selectableValues bool) string {
	if selectableValues && m.cursorPath() == path {
		return colors.cursor.Render(line)
	}
	if s.openBracket != nil {
		if s.openBracket.index == m.searchResultsCursor {
			return colors.cursor.Render(line)
		} else {
			return colors.search.Render(line)
		}
	} else {
		return colors.syntax.Render(line)
	}
}

func (m *model) printCloseBracket(line string, s searchResultGroup, path string, selectableValues bool) string {
	if selectableValues && m.cursorPath() == path {
		return colors.cursor.Render(line)
	}
	if s.closeBracket != nil {
		if s.closeBracket.index == m.searchResultsCursor {
			return colors.cursor.Render(line)
		} else {
			return colors.search.Render(line)
		}
	} else {
		return colors.syntax.Render(line)
	}
}

func (m *model) printDelim(line string, s searchResultGroup) string {
	if s.delim != nil {
		if s.delim.index == m.searchResultsCursor {
			return colors.cursor.Render(line)
		} else {
			return colors.search.Render(line)
		}
	} else {
		return colors.syntax.Render(line)
	}
}

func (m *model) printComma(line string, s searchResultGroup) string {
	if s.comma != nil {
		if s.comma.index == m.searchResultsCursor {
			return colors.cursor.Render(line)
		} else {
			return colors.search.Render(line)
		}
	} else {
		return colors.syntax.Render(line)
	}
}

type withStyle struct {
	value string
	style lipgloss.Style
}

func (m *model) explode(line string, searchResults []*searchResult, defaultStyle lipgloss.Style, path string, selectable bool) []withStyle {
	if selectable && m.cursorPath() == path {
		return []withStyle{{line, colors.cursor}}
	}

	out := make([]withStyle, 0, 1)
	pos := 0
	for _, sr := range searchResults {
		style := colors.search
		if sr.index == m.searchResultsCursor {
			style = colors.cursor
		}
		out = append(out, withStyle{
			value: line[pos:sr.start],
			style: defaultStyle,
		})
		out = append(out, withStyle{
			value: line[sr.start:sr.end],
			style: style,
		})
		pos = sr.end
	}
	out = append(out, withStyle{
		value: line[pos:],
		style: defaultStyle,
	})
	return out
}

func merge(chunks []withStyle) string {
	out := ""
	for _, chunk := range chunks {
		out += chunk.Render(chunk.value)
	}
	return out
}
