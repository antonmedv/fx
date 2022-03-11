package main

import (
	"encoding/json"
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"strings"
)

func (m *model) print(v interface{}, level, lineNumber, keyEndPos int, path string, dontHighlightCursor bool) []string {
	ident := strings.Repeat("  ", level)
	subident := strings.Repeat("  ", level-1)

	cursorOr := func(style lipgloss.Style) lipgloss.Style {
		if m.cursorPath() == path && !dontHighlightCursor && m.showCursor {
			return colors.cursor
		}
		return style
	}
	searchStyle := colors.search.Render
	if m.resultsCursorPath() == path {
		searchStyle = colors.cursor.Render
	}
	highlight := func(line string, style func(s string) string) string {
		chunks := m.split(line, path)
		return mergeChunks(chunks, style, searchStyle)
	}

	switch v.(type) {
	case nil:
		line := highlight("null", cursorOr(colors.null).Render)
		return []string{line}

	case bool:
		line := highlight(stringify(v), cursorOr(colors.boolean).Render)
		return []string{line}

	case json.Number:
		line := highlight(v.(json.Number).String(), cursorOr(colors.number).Render)
		return []string{line}

	case string:
		stringStyle := cursorOr(colors.string).Render
		line := fmt.Sprintf("%q", v)
		chunks := m.split(line, path)
		if m.wrap && keyEndPos+width(line) > m.width {
			return wrapLines(chunks, keyEndPos, m.width, subident, stringStyle, searchStyle)
		}
		return []string{mergeChunks(chunks, stringStyle, searchStyle)}

	case *dict:
		m.pathToLineNumber.set(path, lineNumber)
		m.canBeExpanded[path] = true
		m.lineNumberToPath[lineNumber] = path
		bracketStyle := cursorOr(colors.bracket).Render
		if len(v.(*dict).keys) == 0 {
			return []string{bracketStyle("{}")}
		}
		if !m.expandedPaths[path] {
			return []string{m.preview(v, path, dontHighlightCursor)}
		}
		output := []string{bracketStyle("{")}
		lineNumber++ // bracket is on separate line
		keys := v.(*dict).keys
		for i, k := range keys {
			subpath := path + "." + k
			m.pathToLineNumber.set(subpath, lineNumber)
			m.lineNumberToPath[lineNumber] = subpath
			keyStyle := colors.key.Render
			if m.cursorPath() == subpath && m.showCursor {
				keyStyle = colors.cursor.Render
			}
			key := fmt.Sprintf("%q", k)
			{
				var indexes [][]int
				if m.searchResults != nil {
					sr, ok := m.searchResults.get(subpath)
					if ok {
						indexes = sr.(searchResult).key
					}
				}
				chunks := explode(key, indexes)
				searchStyle := colors.search.Render
				if m.resultsCursorPath() == subpath && !m.showCursor {
					searchStyle = colors.cursor.Render
				}
				key = mergeChunks(chunks, keyStyle, searchStyle)
			}
			value, _ := v.(*dict).get(k)
			delim := ": "
			keyEndPos := width(ident) + width(key) + width(delim)
			lines := m.print(value, level+1, lineNumber, keyEndPos, subpath, true)
			lines[0] = ident + key + delim + lines[0]
			if i < len(keys)-1 {
				lines[len(lines)-1] += ","
			}
			output = append(output, lines...)
			lineNumber += len(lines)
		}
		output = append(output, subident+colors.bracket.Render("}"))
		return output

	case array:
		m.pathToLineNumber.set(path, lineNumber)
		m.canBeExpanded[path] = true
		m.lineNumberToPath[lineNumber] = path
		bracketStyle := cursorOr(colors.bracket).Render
		if len(v.(array)) == 0 {
			return []string{bracketStyle("[]")}
		}
		if !m.expandedPaths[path] {
			return []string{bracketStyle(m.preview(v, path, dontHighlightCursor))}
		}
		output := []string{bracketStyle("[")}
		lineNumber++ // bracket is on separate line
		slice := v.(array)
		for i, value := range slice {
			subpath := fmt.Sprintf("%v[%v]", path, i)
			m.pathToLineNumber.set(subpath, lineNumber)
			m.lineNumberToPath[lineNumber] = subpath
			lines := m.print(value, level+1, lineNumber, width(ident), subpath, false)
			lines[0] = ident + lines[0]
			if i < len(slice)-1 {
				lines[len(lines)-1] += ","
			}
			lineNumber += len(lines)
			output = append(output, lines...)
		}
		output = append(output, subident+colors.bracket.Render("]"))
		return output

	default:
		return []string{"unknown type"}
	}
}

func (m *model) preview(v interface{}, path string, dontHighlightCursor bool) string {
	cursorOr := func(style lipgloss.Style) lipgloss.Style {
		if m.cursorPath() == path && !dontHighlightCursor {
			return colors.cursor
		}
		return style
	}

	bracketStyle := cursorOr(colors.bracket)
	previewStyle := cursorOr(colors.preview)

	printValue := func(value interface{}) string {
		switch value.(type) {
		case nil, bool, json.Number:
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
		output := bracketStyle.Render("{")
		keys := v.(*dict).keys
		for _, k := range keys {
			key := fmt.Sprintf("%q", k)
			output += previewStyle.Render(key + ": ")
			value, _ := v.(*dict).get(k)
			output += printValue(value)
			break
		}
		if len(keys) == 1 {
			output += bracketStyle.Render("}")
		} else {
			output += bracketStyle.Render(", \u2026}")
		}
		return output

	case array:
		output := bracketStyle.Render("[")
		slice := v.(array)
		for _, value := range slice {
			output += printValue(value)
			break
		}
		if len(slice) == 1 {
			output += bracketStyle.Render("]")
		} else {
			output += bracketStyle.Render(", \u2026]")
		}
		return output
	}
	return "?"
}

func wrapLines(chunks []string, keyEndPos, mWidth int, subident string, stringStyle, searchStyle func(s string) string) []string {
	wrappedLines := make([]string, 0)
	currentLine := ""
	ident := ""      // First line stays on the same line with a "key",
	pos := keyEndPos // so no ident is needed. Start counting from the "key" offset.
	style := stringStyle
	for i, chunk := range chunks {
		if i%2 == 0 {
			style = stringStyle
		} else {
			style = searchStyle
		}
		buffer := ""
		for _, ch := range chunk {
			buffer += string(ch)
			if pos == mWidth-1 {
				wrappedLines = append(wrappedLines, ident+currentLine+style(buffer))
				currentLine = ""
				buffer = ""
				pos = width(subident) // Start counting from ident.
				ident = subident      // After first line, add ident to all.
			} else {
				pos++
			}
		}
		currentLine += style(buffer)
	}
	if width(currentLine) > 0 {
		wrappedLines = append(wrappedLines, subident+currentLine)
	}
	return wrappedLines
}

func (m *model) split(line, path string) []string {
	var indexes [][]int
	if m.searchResults != nil {
		sr, ok := m.searchResults.get(path)
		if ok {
			indexes = sr.(searchResult).value
		}
	}
	return explode(line, indexes)
}

func explode(s string, indexes [][]int) []string {
	out := make([]string, 0)
	pos := 0
	for _, l := range indexes {
		out = append(out, s[pos:l[0]])
		out = append(out, s[l[0]:l[1]])
		pos = l[1]
	}
	out = append(out, s[pos:])
	return out
}

func mergeChunks(chunks []string, stringStyle, searchStyle func(s string) string) string {
	currentLine := ""
	for i, chunk := range chunks {
		if i%2 == 0 {
			currentLine += stringStyle(chunk)
		} else {
			currentLine += searchStyle(chunk)
		}
	}
	return currentLine
}
