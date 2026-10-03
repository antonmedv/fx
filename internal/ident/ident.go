package ident

import (
	"os"
	"strconv"
	"strings"
)

var Ident = "  "
var IdentBytes []byte
var IdentWidth int

// maxIdent is the most spaces FX_INDENT may ask for.
const maxIdent = 16

func init() {
	if identValue, ok := os.LookupEnv("FX_INDENT"); ok {
		Ident = fromEnv(identValue, Ident)
	}
	for _, r := range Ident {
		if r == '\n' {
			continue
		}
		if r == '\t' {
			IdentBytes = append(IdentBytes, ' ', ' ', ' ', ' ', ' ', ' ', ' ', ' ')
			IdentWidth += 8
			continue
		}
		IdentBytes = append(IdentBytes, byte(r))
		IdentWidth++
	}
}

// fromEnv is the indent FX_INDENT asks for: a count of spaces, or the
// indent itself. A count out of range keeps def: a negative one panics, a
// huge one takes all memory.
func fromEnv(value, def string) string {
	n, err := strconv.Atoi(value)
	switch {
	case err != nil:
		return value
	case n < 0 || n > maxIdent:
		return def
	}
	return strings.Repeat(" ", n)
}
