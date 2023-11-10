package main

import (
	"strings"
)

type ring struct {
	buf        [70]byte
	start, end int
}

func (r *ring) writeByte(b byte) {
	r.buf[r.end] = b
	r.end = (r.end + 1) % len(r.buf)

	if r.end == r.start {
		r.start = (r.start + 1) % len(r.buf)
	}
}

func (r *ring) string() string {
	var lines []byte
	newlineCount := 0

	for i := r.end - 1; ; i-- {
		if i < 0 {
			i = len(r.buf) - 1
		}

		if r.buf[i] == '\n' {
			newlineCount++
			if newlineCount == 2 {
				break
			}
		}

		lines = append(lines, r.buf[i])

		if i == r.start {
			break
		}
	}

	for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
		lines[i], lines[j] = lines[j], lines[i]
	}

	return strings.TrimRight(string(lines), "\t \n")
}
