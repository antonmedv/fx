package main

import (
	"fmt"
	"regexp"
)

type searchResult struct {
	path   string
	index  int
	ranges []*foundRange
}

type rangeKind int

const (
	keyRange rangeKind = 1 + iota
	valueRange
	delimRange
	openBracketRange
	closeBracketRange
	commaRange
)

type foundRange struct {
	parent     *searchResult
	start, end int
	kind       rangeKind
}

type rangeGroup struct {
	key          []*foundRange
	value        []*foundRange
	delim        *foundRange
	openBracket  *foundRange
	closeBracket *foundRange
	comma        *foundRange
}

func (m *model) doSearch(s string) {
	m.searchRegexCompileError = ""
	re, err := regexp.Compile("(?i)" + s)
	if err != nil {
		m.searchRegexCompileError = err.Error()
		m.searchInput.Blur()
		return
	}
	indexes := re.FindAllStringIndex(stringify(m.json), -1)
	m.remapSearchResult(m.json, "", 0, indexes, 0, nil)
	m.indexSearchResults()
	m.searchInput.Blur()
	m.showSearchResults = true
	m.jumpToSearchResult(0)
}

func (m *model) remapSearchResult(object interface{}, path string, pos int, indexes [][]int, id int, current *searchResult) (int, int, *searchResult) {
	switch object.(type) {
	case nil:
		return pos + len("null"), id, current

	case bool:
		if object.(bool) {
			return pos + len("true"), id, current
		} else {
			return pos + len("false"), id, current
		}

	case number:
		return pos + len(object.(number).String()), id, current

	case string:
		s := fmt.Sprintf("%q", object)
		id, current = m.findRanges(valueRange, s, path, pos, indexes, id, current)
		return pos + len(s), id, current
	case *dict:
		pos++ // {
		for i, k := range object.(*dict).keys {
			subpath := path + "." + k
			key := fmt.Sprintf("%q", k)
			id, current = m.findRanges(keyRange, key, subpath, pos, indexes, id, current)
			pos += len(key)
			delim := ": "
			pos += len(delim)
			pos, id, current = m.remapSearchResult(object.(*dict).values[k], subpath, pos, indexes, id, current)
			if i < len(object.(*dict).keys)-1 {
				pos += len(", ")
			}
		}
		pos++ // }
		return pos, id, current

	case array:
		pos++ // [
		for i, v := range object.(array) {
			subpath := fmt.Sprintf("%v[%v]", path, i)
			pos, id, current = m.remapSearchResult(v, subpath, pos, indexes, id, current)
			if i < len(object.(array))-1 {
				pos += len(", ")
			}
		}
		pos++ // ]
		return pos, id, current
	default:
		panic("unexpected object type")
	}
}

func (m *model) findRanges(kind rangeKind, s string, path string, pos int, indexes [][]int, id int, current *searchResult) (int, *searchResult) {
	for ; id < len(indexes); id++ {
		start, end := indexes[id][0]-pos, indexes[id][1]-pos
		if end <= 0 {
			current = nil
			continue
		}
		if start < len(s) {
			if current == nil {
				current = &searchResult{
					path:  path,
					index: len(m.searchResults),
				}
				m.searchResults = append(m.searchResults, current)
			}
			found := &foundRange{
				parent: current,
				start:  max(start, 0),
				end:    min(end, len(s)),
				kind:   kind,
			}
			current.ranges = append(current.ranges, found)
			if end < len(s) {
				current = nil
			} else {
				break
			}
		} else {
			break
		}
	}
	return id, current
}

func (m *model) indexSearchResults() {
	m.highlightIndex = map[string]*rangeGroup{}
	for _, s := range m.searchResults {
		for _, r := range s.ranges {
			highlight, exist := m.highlightIndex[r.parent.path]
			if !exist {
				highlight = &rangeGroup{}
				m.highlightIndex[r.parent.path] = highlight
			}
			switch r.kind {
			case keyRange:
				highlight.key = append(highlight.key, r)
			case valueRange:
				highlight.value = append(highlight.value, r)
			case delimRange:
				highlight.delim = r
			case openBracketRange:
				highlight.openBracket = r
			case closeBracketRange:
				highlight.closeBracket = r
			case commaRange:
				highlight.comma = r
			}
		}
	}
}

func (m *model) jumpToSearchResult(at int) {
	//if m.searchResults == nil || len(m.searchResults.keys) == 0 {
	//	return
	//}
	//m.showCursor = false
	//m.searchResultsCursor = at % len(m.searchResults.keys)
	//desiredPath := m.searchResults.keys[m.searchResultsCursor]
	//lineNumber, ok := m.pathToLineNumber.get(desiredPath)
	//if ok {
	//	m.cursor = m.pathToLineNumber.indexes[desiredPath]
	//	m.SetOffset(lineNumber.(int))
	//	m.render()
	//} else {
	//	m.expandToPath(desiredPath)
	//	m.render()
	//	m.jumpToSearchResult(at)
	//}
}

func (m *model) expandToPath(path string) {
	m.expandedPaths[path] = true
	if path != "" {
		m.expandToPath(m.parents[path])
	}
}

func (m *model) nextSearchResult() {
	//m.jumpToSearchResult((m.searchResultsCursor + 1) % len(m.searchResults.keys))
}

func (m *model) prevSearchResult() {
	//i := m.searchResultsCursor - 1
	//if i < 0 {
	//	i = len(m.searchResults.keys) - 1
	//}
	//m.jumpToSearchResult(i)
}

func (m *model) resultsCursorPath() string {
	//if m.searchResults == nil || len(m.searchResults.keys) == 0 {
	//	return "?"
	//}
	//return m.searchResults.keys[m.searchResultsCursor]
	return ""
}
