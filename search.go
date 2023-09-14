package main

type match struct {
	start, end int
	index      int
}

func splitBytesByIndexes(b []byte, indexes []match) [][]byte {
	out := make([][]byte, 0, 1)
	pos := 0
	for _, pair := range indexes {
		out = append(out, safeSlice(b, pos, pair[0]))
		out = append(out, safeSlice(b, pair[0], pair[1]))
		pos = pair[1]
	}
	out = append(out, safeSlice(b, pos, len(b)))
	return out
}

func safeSlice(b []byte, start, end int) []byte {
	length := len(b)

	if start > length {
		start = length
	}
	if end > length {
		end = length
	}
	if start < 0 {
		start = 0
	}
	if end < 0 {
		end = 0
	}
	return b[start:end]
}

func splitIndexesToChunks(chunks [][]byte, indexes [][]int) (chunkIndexes [][][]int) {
	// Initialize a slice for indexes of each chunk
	chunkIndexes = make([][][]int, len(chunks))

	// Iterate over each index pair from regex results
	for _, idx := range indexes {
		// Calculate the current position in the whole byte slice
		position := 0
		for i, chunk := range chunks {
			// If start index lies in this chunk
			if idx[0] < position+len(chunk) {
				// Calculate local start and end for this chunk
				localStart := idx[0] - position
				localEnd := idx[1] - position

				// If the end index also lies in this chunk
				if idx[1] <= position+len(chunk) {
					chunkIndexes[i] = append(chunkIndexes[i], []int{localStart, localEnd})
					break
				} else {
					// If the end index is outside this chunk, split the index
					chunkIndexes[i] = append(chunkIndexes[i], []int{localStart, len(chunk)})

					// Adjust the starting index for the next chunk
					idx[0] = position + len(chunk)
				}
			}
			position += len(chunk)
		}
	}

	return
}
