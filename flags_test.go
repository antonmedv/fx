package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/antonmedv/fx/internal/complete"
	"github.com/antonmedv/fx/internal/format"
	"github.com/antonmedv/fx/internal/jsonx"
)

// resetFlags clears the flag variables parseFlags sets.
func resetFlags(t *testing.T) {
	t.Helper()
	reset := func() {
		inputFormat = nil
		flagRaw, flagSlurp, flagStrict, flagNoInline, flagNoTUI, flagComp = false, false, false, false, false, false
	}
	reset()
	t.Cleanup(reset)
}

func TestParseFlags(t *testing.T) {
	tests := []struct {
		name   string
		argv   []string
		args   []string
		action action
		format *format.Format
		raw    bool
		slurp  bool
		err    string
	}{
		{name: "none", argv: nil},
		{name: "args in order", argv: []string{"file.json", ".a", "x"}, args: []string{"file.json", ".a", "x"}},
		{name: "format flag", argv: []string{"--edn", "x"}, args: []string{"x"}, format: format.EDN},
		{name: "same format twice", argv: []string{"--yaml", "--yaml", "x"}, args: []string{"x"}, format: format.YAML},
		{name: "two formats", argv: []string{"--yaml", "--toml"}, err: "can't use --yaml and --toml flags together"},
		{name: "three formats names first two", argv: []string{"--edn", "--yaml", "--toml"}, err: "can't use --edn and --yaml flags together"},
		{name: "format and raw", argv: []string{"--edn", "--raw"}, err: "can't use --edn and --raw flags together"},
		{name: "raw and format", argv: []string{"-r", "--toml"}, err: "can't use --toml and --raw flags together"},
		{name: "raw slurp", argv: []string{"-rs"}, raw: true, slurp: true},
		{name: "raw and slurp", argv: []string{"--raw", "-s", "."}, args: []string{"."}, raw: true, slurp: true},
		{name: "help wins over conflict", argv: []string{"--yaml", "--toml", "-h"}, action: actionHelp},
		{name: "version", argv: []string{"--version"}, action: actionVersion},
		{name: "themes", argv: []string{"--themes"}, action: actionThemes},
		{name: "export themes", argv: []string{"--export-themes"}, action: actionExportThemes},
		{name: "game of life", argv: []string{"--game-of-life"}, action: actionGameOfLife},
		{name: "dungeon", argv: []string{"--dungeon"}, action: actionDungeon},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetFlags(t)
			args, act, err := parseFlags(tt.argv)
			if tt.err != "" {
				require.EqualError(t, err, tt.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.args, args)
			require.Equal(t, tt.action, act)
			if tt.action != actionRead {
				return // The flags before an action do not matter.
			}
			require.Same(t, tt.format, inputFormat)
			require.Equal(t, tt.raw, flagRaw)
			require.Equal(t, tt.slurp, flagSlurp)
		})
	}
}

func TestParseFlags_Others(t *testing.T) {
	resetFlags(t)
	args, act, err := parseFlags([]string{"--strict", "--no-inline", "--no-tui", "--comp=zsh", "x"})
	require.NoError(t, err)
	require.Equal(t, []string{"x"}, args)
	require.Equal(t, actionRead, act)
	require.True(t, flagStrict)
	require.True(t, flagNoInline)
	require.True(t, flagNoTUI)
	require.True(t, flagComp)
}

// Converted input is parsed without --strict: the converter validated the
// source, and the JSON it makes may hold Infinity or NaN.
func TestNewParser_ConvertedInputNotStrict(t *testing.T) {
	resetFlags(t)
	inputFormat, flagStrict = format.EDN, true
	p, err := newParser(strings.NewReader("[##Inf ##NaN]"))
	require.NoError(t, err)
	node, err := p.Parse()
	require.NoError(t, err)
	require.Equal(t, jsonx.Array, node.Kind)
	require.Equal(t, "Infinity", node.Next.Value)
	require.Equal(t, "NaN", node.Next.Next.Value)
}

// --raw wins over a format set from the file extension.
func TestNewParser_RawWinsOverFormat(t *testing.T) {
	resetFlags(t)
	inputFormat, flagRaw = format.YAML, true
	p, err := newParser(strings.NewReader("a: 1\n"))
	require.NoError(t, err)
	node, err := p.Parse()
	require.NoError(t, err)
	require.Equal(t, jsonx.String, node.Kind)
	require.Equal(t, `"a: 1"`, node.Value)
}

func TestCompleteFlagsListFormats(t *testing.T) {
	var values []string
	for _, r := range complete.Flags {
		values = append(values, r.Value)
	}
	for _, f := range format.All {
		require.Contains(t, values, f.Flag)
	}
}
