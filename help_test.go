package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHelpSections(t *testing.T) {
	out := help(keyMap)
	for _, want := range []string{
		"Key Bindings",
		"Commands",
		":<n>",
		":q",
		"Interactive Query",
		"Syntax",
		".items[].name",
		"Input Keys",
		"tab, shift+tab",
		"Press q or ? to close",
	} {
		require.Contains(t, out, want)
	}
}
