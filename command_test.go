package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
)

func TestParseCall(t *testing.T) {
	tests := []struct {
		line string
		want call
	}{
		{"", call{}},
		{"5", call{name: "5"}},
		{"-2", call{name: "-2"}},
		{" 12 ", call{name: "12"}},
		{"q", call{name: "q"}},
		{"w", call{name: "w"}},
		{"w out.json", call{name: "w", arg: "out.json"}},
		{"write   out.json  ", call{name: "write", arg: "out.json"}},
		{"w!", call{name: "w", bang: true}},
		{"w! out.json", call{name: "w", bang: true, arg: "out.json"}},
		{"w!out.json", call{name: "w", bang: true, arg: "out.json"}},
		{"w ~/a b.json", call{name: "w", arg: "~/a b.json"}},
		{"!", call{bang: true}},
		{"w-1", call{name: "w", arg: "-1"}},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			require.Equal(t, tt.want, parseCall(tt.line))
		})
	}
}

func TestFindCommand(t *testing.T) {
	tests := []struct {
		name string
		want string // name of the matched command, "" for none
	}{
		{"5", "<n>"},
		{"-2", "<n>"},
		{"q", "q[uit]"},
		{"qu", "q[uit]"},
		{"quit", "q[uit]"},
		{"quit!", ""},
		{"quitting", ""},
		{"w", "w[rite]"},
		{"wr", "w[rite]"},
		{"write", "w[rite]"},
		{"wq", "wq"},
		{"wqa", ""},
		{"W", ""},
		{"writes", ""},
		{"", ""},
		{"invalid", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := findCommand(tt.name)
			if tt.want == "" {
				require.Nil(t, c)
				return
			}
			require.NotNil(t, c)
			require.Equal(t, tt.want, c.name)
		})
	}
}

func TestCommandUsage(t *testing.T) {
	require.Equal(t, "<n>", command{name: "<n>"}.usage())
	require.Equal(t, "q[uit]", command{name: "q[uit]"}.usage())
	require.Equal(t, "w[rite][!] [file]", command{name: "w[rite]", bang: true, args: "[file]"}.usage())
}

func TestCommandUnknownShowsError(t *testing.T) {
	m := newQueryModel(t, `{"a": 1}`)
	typeKeys(m, ":invalid")
	enter(m)

	require.NotNil(t, m.message)
	require.Equal(t, "Not an editor command: invalid", m.message.text)
	require.True(t, m.message.isErr)
	require.Equal(t, m.termHeight-2, m.viewHeight())
	require.Contains(t, m.View(), "Not an editor command: invalid")

	// The next key clears the message and is handled as usual.
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	require.Nil(t, m.message)
	require.Equal(t, 1, m.cursor)
	require.Equal(t, m.termHeight-1, m.viewHeight())
}

func TestCommandUnknownWithoutName(t *testing.T) {
	for _, line := range []string{"$", "!", "+5", "%s/a/b/", " $ "} {
		m := newQueryModel(t, `{"a": 1}`)
		typeKeys(m, ":"+line)
		enter(m)
		require.NotNil(t, m.message, line)
		require.Equal(t, "Not an editor command: "+strings.TrimSpace(line), m.message.text)
	}
}

func TestCommandBangNotAllowed(t *testing.T) {
	m := newQueryModel(t, `{"a": 1}`)
	typeKeys(m, ":5!")
	enter(m)
	require.NotNil(t, m.message)
	require.Equal(t, "No ! allowed: 5!", m.message.text)
}

func TestCommandEmptyDoesNothing(t *testing.T) {
	m := newQueryModel(t, `{"a": 1}`)
	typeKeys(m, ":")
	enter(m)
	require.Nil(t, m.message)
	require.False(t, m.commandInput.Focused())
}

func TestCommandQuit(t *testing.T) {
	for _, line := range []string{"q", "quit", "q!", "quit!"} {
		m := newQueryModel(t, `{"a": 1}`)
		typeKeys(m, ":"+line)
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		require.NotNil(t, cmd, line)
		require.IsType(t, tea.QuitMsg{}, cmd(), line)
	}
}
