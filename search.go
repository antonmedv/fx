package main

type searchResult struct {
	path       string
	index      int
	start, end int
}

type searchResultGroup struct {
	key          []*searchResult
	value        []*searchResult
	delim        *searchResult
	openBracket  *searchResult
	closeBracket *searchResult
	comma        *searchResult
}

// TODO: Implement search.
// TODO: Uncomment all code blocks.

func (m *model) doSearch(s string) {
	//re, err := regexp.Compile("(?i)" + s)
	//if err != nil {
	//	m.searchRegexCompileError = err.Error()
	//	m.searchInput.Blur()
	//	return
	//}
	//m.searchRegexCompileError = ""
	//results := newDict()
	//addSearchResult := func(path string, indexes [][]int) {
	//	if indexes != nil {
	//		sr := searchResult{}
	//		prev, ok := results.get(path)
	//		if ok {
	//			sr = prev.(searchResult)
	//		}
	//		sr.value = indexes
	//		results.set(path, sr)
	//	}
	//}
	//
	//dfs(m.json, func(it iterator) {
	//	switch it.object.(type) {
	//	case nil:
	//		line := "null"
	//		found := re.FindAllStringIndex(line, -1)
	//		addSearchResult(it.path, found)
	//	case bool:
	//		line := stringify(it.object)
	//		found := re.FindAllStringIndex(line, -1)
	//		addSearchResult(it.path, found)
	//	case number:
	//		line := it.object.(number).String()
	//		found := re.FindAllStringIndex(line, -1)
	//		addSearchResult(it.path, found)
	//	case string:
	//		line := fmt.Sprintf("%q", it.object)
	//		found := re.FindAllStringIndex(line, -1)
	//		addSearchResult(it.path, found)
	//	case *dict:
	//		keys := it.object.(*dict).keys
	//		for _, key := range keys {
	//			line := fmt.Sprintf("%q", key)
	//			subpath := it.path + "." + key
	//			indexes := re.FindAllStringIndex(line, -1)
	//			if indexes != nil {
	//				sr := searchResult{}
	//				prev, ok := results.get(subpath)
	//				if ok {
	//					sr = prev.(searchResult)
	//				}
	//				sr.key = indexes
	//				results.set(subpath, sr)
	//			}
	//		}
	//	}
	//})
	//m.searchResults = results
	//m.searchInput.Blur()
	//m.showSearchResults = true
	//m.jumpToSearchResult(0)
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
