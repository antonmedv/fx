package model

import (
	"fmt"
	"strings"

	. "github.com/antonmedv/fx/pkg/dict"
	. "github.com/antonmedv/fx/pkg/json"
	"github.com/antonmedv/fx/pkg/theme"
)

func (m *model) connect(path string, lineNumber int) {
	if _, exist := m.pathToLineNumber[path]; exist {
		return
	}
	m.paths = append(m.paths, path)
	m.pathToIndex[path] = len(m.paths) - 1
	m.pathToLineNumber[path] = lineNumber
	m.lineNumberToPath[lineNumber] = path
}

func (m *model) print(v interface{}, level, lineNumber, keyEndPos int, path string, selectableValues bool) []string {
	m.connect(path, lineNumber)
	ident := strings.Repeat("  ", level)
	subident := strings.Repeat("  ", level-1)
	highlight := m.highlightIndex[path]
	var searchValue []*foundRange
	if highlight != nil {
		searchValue = highlight.value
	}

	switch v.(type) {
	case nil:
		return []string{merge(m.explode("null", searchValue, m.theme.Null, path, selectableValues))}

	case bool:
		if v.(bool) {
			return []string{merge(m.explode("true", searchValue, m.theme.Boolean, path, selectableValues))}
		} else {
			return []string{merge(m.explode("false", searchValue, m.theme.Boolean, path, selectableValues))}
		}

	case Number:
		return []string{merge(m.explode(v.(Number).String(), searchValue, m.theme.Number, path, selectableValues))}

	case string:
		line := fmt.Sprintf("%q", v)
		chunks := m.explode(line, searchValue, m.theme.String, path, selectableValues)
		if m.wrap && keyEndPos+width(line) > m.width {
			return wrapLines(chunks, keyEndPos, m.width, subident)
		}
		// No wrap
		return []string{merge(chunks)}

	case *Dict:
		if !m.expandedPaths[path] {
			return []string{m.preview(v, path, selectableValues)}
		}
		output := []string{m.printOpenBracket("{", highlight, path, selectableValues)}
		lineNumber++ // bracket is on separate line
		keys := v.(*Dict).Keys
		for i, k := range keys {
			subpath := path + "." + k
			highlight := m.highlightIndex[subpath]
			var keyRanges, delimRanges []*foundRange
			if highlight != nil {
				keyRanges = highlight.key
				delimRanges = highlight.delim
			}
			m.connect(subpath, lineNumber)
			key := fmt.Sprintf("%q", k)
			keyTheme := m.theme.Key(i, len(keys))
			key = merge(m.explode(key, keyRanges, keyTheme, subpath, true))
			value, _ := v.(*Dict).Get(k)
			delim := merge(m.explode(": ", delimRanges, m.theme.Syntax, subpath, false))
			keyEndPos := width(ident) + width(key) + width(delim)
			lines := m.print(value, level+1, lineNumber, keyEndPos, subpath, false)
			lines[0] = ident + key + delim + lines[0]
			if i < len(keys)-1 {
				lines[len(lines)-1] += m.printComma(",", highlight)
			}
			output = append(output, lines...)
			lineNumber += len(lines)
		}
		output = append(output, subident+m.printCloseBracket("}", highlight, path, false))
		return output

	case Array:
		if !m.expandedPaths[path] {
			return []string{m.preview(v, path, selectableValues)}
		}
		output := []string{m.printOpenBracket("[", highlight, path, selectableValues)}
		lineNumber++ // bracket is on separate line
		slice := v.(Array)
		for i, value := range slice {
			subpath := fmt.Sprintf("%v[%v]", path, i)
			s := m.highlightIndex[subpath]
			m.connect(subpath, lineNumber)
			lines := m.print(value, level+1, lineNumber, width(ident), subpath, true)
			lines[0] = ident + lines[0]
			if i < len(slice)-1 {
				lines[len(lines)-1] += m.printComma(",", s)
			}
			lineNumber += len(lines)
			output = append(output, lines...)
		}
		output = append(output, subident+m.printCloseBracket("]", highlight, path, false))
		return output

	default:
		return []string{"unknown type"}
	}
}

func (m *model) preview(v interface{}, path string, selectableValues bool) string {
	searchResult := m.highlightIndex[path]
	previewStyle := m.theme.Preview
	if selectableValues && m.cursorPath() == path {
		previewStyle = m.theme.Cursor
	}
	printValue := func(v interface{}) string {
		switch v := v.(type) {
		case nil, bool, Number:
			return previewStyle(fmt.Sprintf("%v", v))
		case string:
			return previewStyle(fmt.Sprintf("%q", v))
		case *Dict:
			if m.showSize {
				return previewStyle(toLowerNumber(fmt.Sprintf("{\u2026%v\u2026}", len(v.Keys))))
			} else {
				return previewStyle("{\u2026}")
			}
		case Array:
			if m.showSize {
				return previewStyle(toLowerNumber(fmt.Sprintf("[\u2026%v\u2026]", len(v))))
			} else {
				return previewStyle("[\u2026]")
			}
		}
		return "..."
	}

	switch v := v.(type) {
	case *Dict:
		output := m.printOpenBracket("{", searchResult, path, selectableValues)
		keys := v.Keys
		for _, k := range keys {
			key := fmt.Sprintf("%q", k)
			output += previewStyle(key + ": ")
			value, _ := v.Get(k)
			output += printValue(value)
			break
		}
		if len(keys) > 1 {
			if m.showSize {
				output += previewStyle(toLowerNumber(fmt.Sprintf(", \u2026%v\u2026", len(v.Keys)-1)))
			} else {
				output += previewStyle(", \u2026")
			}
		}
		output += m.printCloseBracket("}", searchResult, path, selectableValues)
		return output

	case Array:
		output := m.printOpenBracket("[", searchResult, path, selectableValues)
		for _, value := range v {
			output += printValue(value)
			break
		}
		if len(v) > 1 {
			if m.showSize {
				output += previewStyle(toLowerNumber(fmt.Sprintf(", \u2026%v\u2026", len(v)-1)))
			} else {
				output += previewStyle(", \u2026")
			}
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
	return w.style(s)
}

func (m *model) printOpenBracket(line string, s *rangeGroup, path string, selectableValues bool) string {
	if selectableValues && m.cursorPath() == path {
		return m.theme.Cursor(line)
	}
	if s != nil && s.openBracket != nil {
		if s.openBracket.parent.index == m.searchResultsCursor {
			return m.theme.Cursor(line)
		} else {
			return m.theme.Search(line)
		}
	} else {
		return m.theme.Syntax(line)
	}
}

func (m *model) printCloseBracket(line string, s *rangeGroup, path string, selectableValues bool) string {
	if selectableValues && m.cursorPath() == path {
		return m.theme.Cursor(line)
	}
	if s != nil && s.closeBracket != nil {
		if s.closeBracket.parent.index == m.searchResultsCursor {
			return m.theme.Cursor(line)
		} else {
			return m.theme.Search(line)
		}
	} else {
		return m.theme.Syntax(line)
	}
}

func (m *model) printComma(line string, s *rangeGroup) string {
	if s != nil && s.comma != nil {
		if s.comma.parent.index == m.searchResultsCursor {
			return m.theme.Cursor(line)
		} else {
			return m.theme.Search(line)
		}
	} else {
		return m.theme.Syntax(line)
	}
}

type withStyle struct {
	value string
	style theme.Color
}

func (m *model) explode(line string, highlightRanges []*foundRange, defaultStyle theme.Color, path string, selectable bool) []withStyle {
	if selectable && m.cursorPath() == path && m.showCursor {
		return []withStyle{{line, m.theme.Cursor}}
	}

	out := make([]withStyle, 0, 1)
	pos := 0
	for _, r := range highlightRanges {
		style := m.theme.Search
		if r.parent.index == m.searchResultsCursor {
			style = m.theme.Cursor
		}
		out = append(out, withStyle{
			value: line[pos:r.start],
			style: defaultStyle,
		})
		out = append(out, withStyle{
			value: line[r.start:r.end],
			style: style,
		})
		pos = r.end
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
