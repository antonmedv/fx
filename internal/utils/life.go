package utils

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"time"

	"github.com/charmbracelet/x/term"
)

func GameOfLife() {
	w, rows, err := term.GetSize(os.Stdout.Fd())
	if err != nil || w <= 0 || rows <= 0 {
		w, rows = 80, 24
	}
	h := rows * 2
	size := w * h
	s := make([]bool, size)

	switch rand.Int() % 3 {
	case 0:
		for i := 0; i < size; i++ {
			if rand.Float64() < 0.16 {
				s[i] = true
			}
		}
	case 1:
		cx := w/2 - 6
		cy := h/2 - 7
		s[cx+1+(2+cy)*w] = true
		s[cx+2+(1+cy)*w] = true
		s[cx+2+(3+cy)*w] = true
		s[cx+3+(2+cy)*w] = true
		s[cx+5+(15+cy)*w] = true
		s[cx+6+(13+cy)*w] = true
		s[cx+6+(15+cy)*w] = true
		s[cx+7+(12+cy)*w] = true
		s[cx+7+(13+cy)*w] = true
		s[cx+7+(15+cy)*w] = true
		s[cx+9+(11+cy)*w] = true
		s[cx+9+(12+cy)*w] = true
		s[cx+9+(13+cy)*w] = true
	case 2:
		s[1+5*w] = true
		s[1+6*w] = true
		s[2+5*w] = true
		s[2+6*w] = true
		s[12+5*w] = true
		s[12+6*w] = true
		s[12+7*w] = true
		s[13+4*w] = true
		s[13+8*w] = true
		s[14+3*w] = true
		s[14+9*w] = true
		s[15+4*w] = true
		s[15+8*w] = true
		s[16+5*w] = true
		s[16+6*w] = true
		s[16+7*w] = true
		s[17+5*w] = true
		s[17+6*w] = true
		s[17+7*w] = true
		s[22+3*w] = true
		s[22+4*w] = true
		s[22+5*w] = true
		s[23+2*w] = true
		s[23+3*w] = true
		s[23+5*w] = true
		s[23+6*w] = true
		s[24+2*w] = true
		s[24+3*w] = true
		s[24+5*w] = true
		s[24+6*w] = true
		s[25+2*w] = true
		s[25+3*w] = true
		s[25+4*w] = true
		s[25+5*w] = true
		s[25+6*w] = true
		s[26+w] = true
		s[26+2*w] = true
		s[26+6*w] = true
		s[26+7*w] = true
		s[35+3*w] = true
		s[35+4*w] = true
		s[36+3*w] = true
		s[36+4*w] = true
	}

	out := bufio.NewWriter(os.Stdout)

	esc := func(codes ...string) {
		for _, c := range codes {
			fmt.Fprintf(out, "\x1b[%s", c)
		}
	}

	esc("2J", "?25l")

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt)
	go func() {
		<-sigc
		esc("?25h")
		out.Flush()
		fmt.Printf("\n")
		os.Exit(2)
	}()

	at := func(i, j int) bool {
		if i < 0 {
			i = h - 1
		}
		if i >= h {
			i = 0
		}
		if j < 0 {
			j = w - 1
		}
		if j >= w {
			j = 0
		}
		return s[i*w+j]
	}

	fullBlock := "\u2588"
	topHalf := "\u2580"
	botHalf := "\u2584"

	ticker := time.NewTicker(30 * time.Millisecond)
	defer ticker.Stop()

	for {
		<-ticker.C

		esc("H")

		gen := make([]bool, size)
		for i := h - 1; i >= 0; i-- {
			for j := w - 1; j >= 0; j-- {
				n := 0
				if at(i-1, j-1) {
					n++
				}
				if at(i-1, j) {
					n++
				}
				if at(i-1, j+1) {
					n++
				}
				if at(i, j-1) {
					n++
				}
				if at(i, j+1) {
					n++
				}
				if at(i+1, j-1) {
					n++
				}
				if at(i+1, j) {
					n++
				}
				if at(i+1, j+1) {
					n++
				}
				z := i*w + j
				if s[z] {
					gen[z] = n == 2 || n == 3
				} else {
					gen[z] = n == 3
				}
			}
		}
		s = gen

		for i := 0; i < rows; i++ {
			for j := 0; j < w; j++ {
				top := s[i*2*w+j]
				bot := s[(i*2+1)*w+j]
				switch {
				case top && bot:
					out.WriteString(fullBlock)
				case top && !bot:
					out.WriteString(topHalf)
				case !top && bot:
					out.WriteString(botHalf)
				default:
					out.WriteByte(' ')
				}
			}
			if i != rows-1 {
				out.WriteByte('\n')
			}
		}
		out.Flush()
	}
}
