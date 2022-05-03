package main

import (
	"fmt"
	"regexp"

	. "github.com/antonmedv/fx/pkg/dict"
	. "github.com/antonmedv/fx/pkg/json"
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
	parent *searchResult
	// Range needs separate path, as for one searchResult's path
	// there can be multiple ranges for different paths (within parent path).
	path       string
	start, end int
	kind       rangeKind
}

type rangeGroup struct {
	key          []*foundRange
	value        []*foundRange
	delim        []*foundRange
	openBracket  *foundRange
	closeBracket *foundRange
	comma        *foundRange
}

func (m *model) clearSearchResults() {
	m.searchRegexCompileError = ""
	m.searchResults = nil
	m.highlightIndex = nil
}

func (m *model) doSearch(s string) {
	m.clearSearchResults()
	re, err := regexp.Compile("(?i)" + s)
	if err != nil {
		m.searchRegexCompileError = err.Error()
		m.searchInput.Blur()
		return
	}
	indexes := re.FindAllStringIndex(Stringify(m.json), -1)
	m.remapSearchResult(m.json, "", 0, indexes, 0, nil)
	m.indexSearchResults()
	m.searchInput.Blur()
	m.showSearchResults = true
	m.jumpToSearchResult(0)
}

func (m *model) remapSearchResult(object interface{}, path string, pos int, indexes [][]int, id int, current *searchResult) (int, int, *searchResult) {
	switch object.(type) {
	case nil:
		s := "null"
		id, current = m.findRanges(valueRange, s, path, pos, indexes, id, current)
		return pos + len(s), id, current

	case bool:
		var s string
		if object.(bool) {
			s = "true"
		} else {
			s = "false"
		}
		id, current = m.findRanges(valueRange, s, path, pos, indexes, id, current)
		return pos + len(s), id, current

	case Number:
		s := object.(Number).String()
		id, current = m.findRanges(valueRange, s, path, pos, indexes, id, current)
		return pos + len(s), id, current

	case string:
		// TODO: Wrap the string, save a line number according to current wrap or no wrap mode.
		s := fmt.Sprintf("%q", object)
		id, current = m.findRanges(valueRange, s, path, pos, indexes, id, current)
		return pos + len(s), id, current

	case *Dict:
		id, current = m.findRanges(openBracketRange, "{", path, pos, indexes, id, current)
		pos++ // {
		for i, k := range object.(*Dict).Keys {
			subpath := path + "." + k

			key := fmt.Sprintf("%q", k)
			id, current = m.findRanges(keyRange, key, subpath, pos, indexes, id, current)
			pos += len(key)

			delim := ": "
			id, current = m.findRanges(delimRange, delim, subpath, pos, indexes, id, current)
			pos += len(delim)

			pos, id, current = m.remapSearchResult(object.(*Dict).Values[k], subpath, pos, indexes, id, current)
			if i < len(object.(*Dict).Keys)-1 {
				comma := ","
				id, current = m.findRanges(commaRange, comma, subpath, pos, indexes, id, current)
				pos += len(comma)
			}
		}
		id, current = m.findRanges(closeBracketRange, "}", path, pos, indexes, id, current)
		pos++ // }
		return pos, id, current

	case Array:
		id, current = m.findRanges(openBracketRange, "[", path, pos, indexes, id, current)
		pos++ // [
		for i, v := range object.(Array) {
			subpath := fmt.Sprintf("%v[%v]", path, i)
			pos, id, current = m.remapSearchResult(v, subpath, pos, indexes, id, current)
			if i < len(object.(Array))-1 {
				comma := ","
				id, current = m.findRanges(commaRange, comma, subpath, pos, indexes, id, current)
				pos += len(comma)
			}
		}
		id, current = m.findRanges(closeBracketRange, "]", path, pos, indexes, id, current)
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
				path:   path,
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
			highlight, exist := m.highlightIndex[r.path]
			if !exist {
				highlight = &rangeGroup{}
				m.highlightIndex[r.path] = highlight
			}
			switch r.kind {
			case keyRange:
				highlight.key = append(highlight.key, r)
			case valueRange:
				highlight.value = append(highlight.value, r)
			case delimRange:
				highlight.delim = append(highlight.delim, r)
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
	if len(m.searchResults) == 0 {
		return
	}
	m.showCursor = false
	m.searchResultsCursor = at % len(m.searchResults)
	desiredPath := m.searchResults[m.searchResultsCursor].path
	_, ok := m.pathToLineNumber[desiredPath]
	if ok {
		m.cursor = m.pathToIndex[desiredPath]
		m.scrollDownToCursor()
		m.render()
	} else {
		m.expandToPath(desiredPath)
		m.render()
		m.jumpToSearchResult(at)
	}
}

func (m *model) expandToPath(path string) {
	m.expandedPaths[path] = true
	if path != "" {
		m.expandToPath(m.parents[path])
	}
}

func (m *model) nextSearchResult() {
	if len(m.searchResults) > 0 {
		m.jumpToSearchResult((m.searchResultsCursor + 1) % len(m.searchResults))
	}
}

func (m *model) prevSearchResult() {
	i := m.searchResultsCursor - 1
	if i < 0 {
		i = len(m.searchResults) - 1
	}
	m.jumpToSearchResult(i)
}

func (m *model) resultsCursorPath() string {
	if len(m.searchResults) == 0 {
		return "?"
	}
	return m.searchResults[m.searchResultsCursor].path
}
