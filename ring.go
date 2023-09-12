package main

type ring struct {
	buf        [100]byte
	start, end int
}

func (r *ring) writeByte(b byte) {
	if b == '\n' {
		r.end = 0
		r.start = r.end
		return
	}
	r.buf[r.end] = b
	r.end++
	if r.end >= len(r.buf) {
		r.end = 0
	}
	if r.end == r.start {
		r.start++
		if r.start >= len(r.buf) {
			r.start = 0
		}
	}
}

func (r *ring) string() string {
	if r.start < r.end {
		return string(r.buf[r.start:r.end])
	}
	return string(r.buf[r.start:]) + string(r.buf[:r.end])
}
