package main

import (
	"math"
)

func (m *model) AtTop() bool {
	return m.offset <= 0
}

func (m *model) AtBottom() bool {
	return m.offset >= m.maxYOffset()
}

func (m *model) PastBottom() bool {
	return m.offset > m.maxYOffset()
}

func (m *model) ScrollPercent() float64 {
	if m.height >= len(m.lines) {
		return 1.0
	}
	y := float64(m.offset)
	h := float64(m.height)
	t := float64(len(m.lines) - 1)
	v := y / (t - h)
	return math.Max(0.0, math.Min(1.0, v))
}

func (m *model) maxYOffset() int {
	return max(0, len(m.lines)-m.height)
}

func (m *model) visibleLines() (lines []string) {
	if len(m.lines) > 0 {
		top := max(0, m.offset)
		bottom := clamp(m.offset+m.height, top, len(m.lines))
		lines = m.lines[top:bottom]
	}
	return lines
}

func (m *model) SetOffset(n int) {
	m.offset = clamp(n, 0, m.maxYOffset())
}

func (m *model) ViewDown() {
	if m.AtBottom() {
		return
	}

	m.SetOffset(m.offset + m.height)
}

func (m *model) ViewUp() {
	if m.AtTop() {
		return
	}

	m.SetOffset(m.offset - m.height)
}

func (m *model) HalfViewDown() {
	if m.AtBottom() {
		return
	}

	m.SetOffset(m.offset + m.height/2)
}

func (m *model) HalfViewUp() {
	if m.AtTop() {
		return
	}

	m.SetOffset(m.offset - m.height/2)
}

func (m *model) LineDown(n int) {
	if m.AtBottom() || n == 0 {
		return
	}

	// Make sure the number of lines by which we're going to scroll isn't
	// greater than the number of lines we actually have left before we reach
	// the bottom.
	m.SetOffset(m.offset + n)
}

func (m *model) LineUp(n int) {
	if m.AtTop() || n == 0 {
		return
	}

	// Make sure the number of lines by which we're going to scroll isn't
	// greater than the number of lines we are from the top.
	m.SetOffset(m.offset - n)
}

func (m *model) GotoTop() {
	if m.AtTop() {
		return
	}

	m.SetOffset(0)
}

func (m *model) GotoBottom() {
	m.SetOffset(m.maxYOffset())
}

func (m *model) scrollDownToCursor() {
	at := m.cursorLineNumber()
	if m.offset <= at { // cursor is lower
		m.LineDown(max(0, at-(m.offset+m.height-1))) // minus one is due to cursorLineNumber() starts from 0
	} else {
		m.SetOffset(at)
	}
}

func (m *model) scrollUpToCursor() {
	at := m.cursorLineNumber()
	if at < m.offset+m.height { // cursor is above
		m.LineUp(max(0, m.offset-at))
	} else {
		m.SetOffset(at)
	}
}
