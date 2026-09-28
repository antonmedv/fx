package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWindowTitle(t *testing.T) {
	m := newQueryModel(t, `{"a": 1}`)
	require.Equal(t, "", m.View().WindowTitle, "stdin keeps the terminal's title")

	m.fileName = "example.json"
	require.Equal(t, "example.json", m.View().WindowTitle)
}
