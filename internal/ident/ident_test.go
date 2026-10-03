package ident

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFromEnv(t *testing.T) {
	for value, want := range map[string]string{
		"4":          "    ",
		"0":          "",
		"16":         "                ",
		"-1":         "  ",
		"17":         "  ",
		"1000000000": "  ",
		"\t":         "\t",
		"--":         "--",
	} {
		assert.Equal(t, want, fromEnv(value, "  "), "FX_INDENT=%q", value)
	}
}
