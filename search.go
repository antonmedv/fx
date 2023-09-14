package main

type search struct {
	err     error
	results []*node
	cursor  int
	values  map[*node][]match
	keys    map[*node][]match
}

func NewSearch() *search {
	return &search{
		results: make([]*node, 0),
		values:  make(map[*node][]match),
		keys:    make(map[*node][]match),
	}
}

type match struct {
	start, end int
	index      int
}

type кусочек struct {
	b     []byte
	index int
}

func splitBytesByIndexes(b []byte, indexes []match) []кусочек {
	out := make([]кусочек, 0, 1)
	pos := 0
	for _, pair := range indexes {
		out = append(out, кусочек{safeSlice(b, pos, pair.start), -1})
		out = append(out, кусочек{safeSlice(b, pair.start, pair.end), pair.index})
		pos = pair.end
	}
	out = append(out, кусочек{safeSlice(b, pos, len(b)), -1})
	return out
}

func splitIndexesToChunks(chunks [][]byte, indexes [][]int, searchIndex int) (chunkIndexes [][]match) {
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
