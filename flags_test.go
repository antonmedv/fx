package main

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/antonmedv/fx/internal/complete"
	"github.com/antonmedv/fx/internal/format"
)

// resetFlags clears the flag variables parseFlags sets.
func resetFlags(t *testing.T) {
	t.Helper()
	reset := func() {
		inputFormat = nil
		flagRaw, flagSlurp, flagStrict, flagNoInline, flagComp = false, false, false, false, false
	}
	reset()
	t.Cleanup(reset)
}

func TestParseFlags(t *testing.T) {
	tests := []struct {
		name   string
		argv   []string
		args   []string
		action string
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
		{name: "help wins over conflict", argv: []string{"--yaml", "--toml", "-h"}, action: "help"},
		{name: "version", argv: []string{"--version"}, action: "version"},
		{name: "themes", argv: []string{"--themes"}, action: "themes"},
		{name: "export themes", argv: []string{"--export-themes"}, action: "export-themes"},
		{name: "game of life", argv: []string{"--game-of-life"}, action: "game-of-life"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetFlags(t)
			args, action, err := parseFlags(tt.argv)
			if tt.err != "" {
				require.EqualError(t, err, tt.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.args, args)
			require.Equal(t, tt.action, action)
			if tt.action != "" {
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
	args, action, err := parseFlags([]string{"--strict", "--no-inline", "--comp=zsh", "x"})
	require.NoError(t, err)
	require.Equal(t, []string{"x"}, args)
	require.Equal(t, "", action)
	require.True(t, flagStrict)
	require.True(t, flagNoInline)
	require.True(t, flagComp)
}

func TestChooseFormat(t *testing.T) {
	require.Same(t, format.TOML, chooseFormat(format.TOML, "x.yaml"), "flag wins over extension")
	require.Same(t, format.YAML, chooseFormat(nil, "x.yaml"))
	require.Same(t, format.YAML, chooseFormat(nil, "dir/X.YML"))
	require.Same(t, format.EDN, chooseFormat(nil, "deps.edn"))
	require.Nil(t, chooseFormat(nil, "x.json"))
	require.Nil(t, chooseFormat(nil, "x"))
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
