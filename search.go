package main

import (
	"regexp"
	"time"

	. "github.com/antonmedv/fx/internal/jsonx"
)

type search struct {
	err     error
	results []*Node
	cursor  int
	values  map[*Node][]match
	keys    map[*Node][]match
}

type searchCacheEntry struct {
	query       string
	regex       *regexp.Regexp
	search      *search
	timestamp   time.Time
	dataVersion int64 // Version of the data when this cache was created
}

// searchCache manages cached search results to avoid O(n×m) complexity on repeated searches
type searchCache struct {
	entries     map[string]*searchCacheEntry
	maxEntries  int
	dataVersion int64
}

func newSearchCache(maxEntries int) *searchCache {
	return &searchCache{
		entries:     make(map[string]*searchCacheEntry),
		maxEntries:  maxEntries,
		dataVersion: 0,
	}
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
func executeSearch(top *Node, s string, cancel <-chan struct{}) (*search, *regexp.Regexp, error) {
	code, ci := regexCase(s)
	if ci {
		code = "(?i)" + code
	}

	re, err := regexp.Compile(code)
	if err != nil {
		return nil, nil, err
	}

	result := newSearch()
	n := top
	searchIndex := 0

	for n != nil {
		// Check for cancellation if channel provided
		if cancel != nil {
			select {
			case <-cancel:
				return nil, nil, nil // cancelled
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

	return result, re, nil
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

func (sc *searchCache) get(query string) (*search, *regexp.Regexp, bool) {
	entry, exists := sc.entries[query]
	if !exists {
		return nil, nil, false
	}

	if entry.dataVersion != sc.dataVersion {
		delete(sc.entries, query)
		return nil, nil, false
	}

	// Update timestamp for LRU
	entry.timestamp = time.Now()
	return entry.search, entry.regex, true
}

func (sc *searchCache) put(query string, regex *regexp.Regexp, searchResult *search) {
	if len(sc.entries) >= sc.maxEntries {
		sc.evictOldest()
	}

	sc.entries[query] = &searchCacheEntry{
		query:       query,
		regex:       regex,
		search:      searchResult,
		timestamp:   time.Now(),
		dataVersion: sc.dataVersion,
	}
}

// evictOldest removes the oldest cache entry (LRU)
func (sc *searchCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time
	first := true

	for key, entry := range sc.entries {
		if first || entry.timestamp.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.timestamp
			first = false
		}
	}

	if oldestKey != "" {
		delete(sc.entries, oldestKey)
	}
}

func (sc *searchCache) invalidate() {
	sc.dataVersion++
}

// size returns the number of cached entries
func (sc *searchCache) size() int {
	return len(sc.entries)
}
