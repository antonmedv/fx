package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestYank_SendsOSC52(t *testing.T) {
	m := newQueryModel(t, `{"a": "b"}`)
	m.sshSession = true // keeps the test off this machine's clipboard

	m.Update(press("y"))
	require.True(t, m.yank)

	_, cmd := m.Update(press("y"))
	require.False(t, m.yank)
	require.NotNil(t, cmd, "OSC52 command")
	require.NotNil(t, cmd())
}
