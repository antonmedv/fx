package main

import (
	"regexp"

	tea "github.com/charmbracelet/bubbletea"

	. "github.com/antonmedv/fx/internal/jsonx"
)

func (m *model) doSearch(s string) tea.Cmd {
	if s == "" {
		return nil
	}

	m.searching = true
	m.searchID++
	m.searchCancel = make(chan struct{})
	id := m.searchID
	cancel := m.searchCancel
	top := m.top
	query := s

	return tea.Batch(m.spinner.Tick, func() tea.Msg {
		result, err := executeSearch(top, query, cancel)
		if err != nil {
			errSearch := newSearch()
			errSearch.err = err
			return searchResultMsg{id: id, query: query, search: errSearch}
		}
		if result == nil {
			// Search was cancelled
			return searchCancelledMsg{}
		}
		return searchResultMsg{id: id, query: query, search: result}
	})
}

func (m *model) cancelSearch() {
	if m.searchCancel != nil {
		close(m.searchCancel)
		m.searchCancel = nil
		m.searching = false
	}
}

func (m *model) selectSearchResult(i int) {
	if len(m.search.results) == 0 {
		return
	}
	if i < 0 {
		i = len(m.search.results) - 1
	}
	if i >= len(m.search.results) {
		i = 0
	}
	m.search.cursor = i
	result := m.search.results[i]
	m.selectNode(result)
	m.showCursor = false
}

func (m *model) redoSearch() {
	s := m.searchInput.Value()
	if s == "" || len(m.search.results) == 0 {
		return
	}

	cursor := m.search.cursor

	// Perform search synchronously (no cancellation needed for redo)
	result, err := executeSearch(m.top, s, nil)
	if err != nil {
		m.search = newSearch()
		m.search.err = err
		return
	}

	m.search = result
	m.selectSearchResult(cursor)
}

type search struct {
	err     error
	results []*Node
	cursor  int
	values  map[*Node][]match
	keys    map[*Node][]match
}

func newSearch() *search {
	return &search{
		results: make([]*Node, 0),
		values:  make(map[*Node][]match),
		keys:    make(map[*Node][]match),
	}
}

type match struct {
	start, end int
	index      int
}

type piece struct {
	b     string
	index int
}

// executeSearch performs the core search logic and returns the results.
// It can be cancelled via the cancel channel (pass nil for non-cancellable search).
func executeSearch(top *Node, s string, cancel <-chan struct{}) (*search, error) {
	code, ci := regexCase(s)
	if ci {
		code = "(?i)" + code
	}

	re, err := regexp.Compile(code)
	if err != nil {
		return nil, err
	}

	result := newSearch()
	n := top
	searchIndex := 0

	for n != nil {
		// Check for cancellation if channel provided
		if cancel != nil {
			select {
			case <-cancel:
				return nil, nil // cancelled
			default:
			}
		}

		if n.Key != "" {
			indexes := re.FindAllStringIndex(n.Key, -1)
			if len(indexes) > 0 {
				for i, pair := range indexes {
					result.results = append(result.results, n)
					result.keys[n] = append(result.keys[n], match{start: pair[0], end: pair[1], index: searchIndex + i})
				}
				searchIndex += len(indexes)
			}
		}
		indexes := re.FindAllStringIndex(n.Value, -1)
		if len(indexes) > 0 {
			for range indexes {
				result.results = append(result.results, n)
			}
			if n.Chunk != "" {
				// String can be split into chunks, so we need to map the indexes to the chunks.
				chunks := []string{n.Chunk}
				chunkNodes := []*Node{n}

				it := n.Next
				for it != nil {
					chunkNodes = append(chunkNodes, it)
					chunks = append(chunks, it.Chunk)
					if it == n.ChunkEnd {
						break
					}
					it = it.Next
				}

				chunkMatches := splitIndexesToChunks(chunks, indexes, searchIndex)
				for i, matches := range chunkMatches {
					result.values[chunkNodes[i]] = matches
				}
			} else {
				for i, pair := range indexes {
					result.values[n] = append(result.values[n], match{start: pair[0], end: pair[1], index: searchIndex + i})
				}
			}
			searchIndex += len(indexes)
		}

		if n.IsCollapsed() {
			n = n.Collapsed
		} else {
			n = n.Next
		}
	}

	return result, nil
}

func splitByIndexes(s string, indexes []match) []piece {
	out := make([]piece, 0, 1)
	pos := 0
	for _, pair := range indexes {
		out = append(out, piece{safeSlice(s, pos, pair.start), -1})
		out = append(out, piece{safeSlice(s, pair.start, pair.end), pair.index})
		pos = pair.end
	}
	out = append(out, piece{safeSlice(s, pos, len(s)), -1})
	return out
}

func splitIndexesToChunks(chunks []string, indexes [][]int, searchIndex int) (chunkIndexes [][]match) {
	chunkIndexes = make([][]match, len(chunks))

	for index, idx := range indexes {
		position := 0
		for i, chunk := range chunks {
			// If start index lies in this chunk
			if idx[0] < position+len(chunk) {
				// Calculate local start and end for this chunk
				localStart := idx[0] - position
				localEnd := idx[1] - position

				// If the end index also lies in this chunk
				if idx[1] <= position+len(chunk) {
					chunkIndexes[i] = append(chunkIndexes[i], match{start: localStart, end: localEnd, index: searchIndex + index})
					break
				} else {
					// If the end index is outside this chunk, split the index
					chunkIndexes[i] = append(chunkIndexes[i], match{start: localStart, end: len(chunk), index: searchIndex + index})

					// Adjust the starting index for the next chunk
					idx[0] = position + len(chunk)
				}
			}
			position += len(chunk)
		}
	}

	return
}
