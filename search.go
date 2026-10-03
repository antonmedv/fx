package main

import (
	"regexp"
	"sync"

	tea "charm.land/bubbletea/v2"

	. "github.com/antonmedv/fx/internal/jsonx"
)

func (m *model) doSearch(s string) tea.Cmd {
	if s == "" {
		return nil
	}

	m.searching = true
	m.searchID++
	run := &searchRun{cancel: make(chan struct{}), done: make(chan struct{})}
	m.searchRun = run
	id := m.searchID
	top := m.top
	// Documents still streaming in are attached after the last line of the
	// last one: the search stops there, so it never reads that link.
	last := lastLine(m.bottom)
	query := s

	return tea.Batch(m.spinner.Tick, func() tea.Msg {
		if !run.begin() {
			return searchCancelledMsg{id: id}
		}
		result, err := executeSearch(top, last, query, run.cancel)
		run.end()
		if err != nil {
			errSearch := newSearch()
			errSearch.err = err
			return searchResultMsg{id: id, query: query, search: errSearch}
		}
		if result == nil {
			// Search was cancelled
			return searchCancelledMsg{id: id}
		}
		return searchResultMsg{id: id, query: query, search: result}
	})
}

// cancelSearch stops the search running in the background, waiting for it
// to stop: it reads the list, which the caller may then edit. A result in
// flight is dropped.
func (m *model) cancelSearch() {
	if m.searchRun == nil {
		return
	}
	m.searchRun.stop()
	m.searchRun = nil
	m.searching = false
	m.searchID++
}

// searchRun is a search running in the background.
type searchRun struct {
	mu      sync.Mutex
	stopped bool
	running bool
	cancel  chan struct{}
	done    chan struct{} // Closed once a started search returns.
}

// begin reports whether the search may start: it was not stopped already.
func (r *searchRun) begin() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.stopped {
		return false
	}
	r.running = true
	return true
}

func (r *searchRun) end() {
	close(r.done)
}

// stop cancels the search and waits for it to return, if it started. A
// search that has not started never will.
func (r *searchRun) stop() {
	r.mu.Lock()
	if r.stopped {
		r.mu.Unlock()
		return
	}
	r.stopped = true
	close(r.cancel)
	running := r.running
	r.mu.Unlock()
	if running {
		<-r.done
	}
}

// lastLine is the last line of the document doc, after which the next
// document is attached.
func lastLine(doc *Node) *Node {
	switch {
	case doc == nil:
		return nil
	case doc.End != nil:
		return doc.End
	case doc.ChunkEnd != nil:
		return doc.ChunkEnd
	}
	return doc
}

// selectSearchResult selects result i, wrapping around. A result deleted
// since the search is skipped, in the direction from the current one.
func (m *model) selectSearchResult(i int) {
	n := len(m.search.results)
	if n == 0 {
		return
	}
	step := 1
	if i < m.search.cursor {
		step = -1
	}
	found := false
	for range n {
		i = (i%n + n) % n
		if m.search.results[i].InDocument() {
			found = true
			break
		}
		i += step
	}
	if !found {
		return
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
	result, err := executeSearch(m.top, nil, s, nil)
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
// executeSearch searches the list from top to last, or to its end if last
// is nil.
func executeSearch(top, last *Node, s string, cancel <-chan struct{}) (*search, error) {
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

		if n == last {
			break
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
