package ident

import (
	"os"
	"strconv"
	"strings"
)

var Ident = "  "
var IdentBytes []byte
var IdentWidth int

func init() {
	identValue, ok := os.LookupEnv("FX_INDENT")
	if ok {
		identInt, err := strconv.Atoi(identValue)
		if err == nil {
			Ident = strings.Repeat(" ", identInt)
		} else {
			Ident = identValue
		}
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
