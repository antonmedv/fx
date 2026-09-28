package format

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// samples gives, per format name, an input that converts to {"a":1} and
// one that is invalid. Every format in All needs an entry.
var samples = map[string]struct{ ok, bad string }{
	"yaml": {"a: 1\n", "a: [1"},
	"toml": {"a = 1\n", "a = [1"},
	"edn":  {"{:a 1}", "{:a"},
}

func TestAll(t *testing.T) {
	flags := map[string]bool{}
	for _, f := range All {
		t.Run(f.Name, func(t *testing.T) {
			require.NotEmpty(t, f.Name)
			require.NotEmpty(t, f.Help)
			require.NotEmpty(t, f.Exts)
			require.NotNil(t, f.ToJSON)
			require.True(t, strings.HasPrefix(f.Flag, "--"), "flag %q", f.Flag)
			require.False(t, flags[f.Flag], "flag %q used twice", f.Flag)
			flags[f.Flag] = true

			require.Same(t, f, ByFlag(f.Flag))
			for _, ext := range f.Exts {
				require.True(t, strings.HasPrefix(ext, "."), "ext %q", ext)
				require.Same(t, f, ByFile("x"+ext), "ext %q", ext)
				require.Same(t, f, ByFile("dir/x"+strings.ToUpper(ext)), "ext %q", ext)
			}

			sample, ok := samples[f.Name]
			require.True(t, ok, "add a sample for %q to samples", f.Name)
			got, err := f.ToJSON([]byte(sample.ok))
			require.NoError(t, err)
			var buf bytes.Buffer
			require.NoError(t, json.Compact(&buf, got))
			require.Equal(t, `{"a":1}`, buf.String())

			_, err = f.ToJSON([]byte(sample.bad))
			require.Error(t, err)
		})
	}
}

func TestLookupMisses(t *testing.T) {
	require.Nil(t, ByFlag("--json"))
	require.Nil(t, ByFlag("yaml"))
	require.Nil(t, ByFile("x.json"))
	require.Nil(t, ByFile("x"))
	require.Nil(t, ByFile("yaml"))
}

func TestChoose(t *testing.T) {
	require.Same(t, TOML, Choose(TOML, "x.yaml"), "flag wins over extension")
	require.Same(t, YAML, Choose(nil, "x.yaml"))
	require.Same(t, YAML, Choose(nil, "dir/X.YML"))
	require.Same(t, EDN, Choose(nil, "deps.edn"))
	require.Nil(t, Choose(nil, "x.json"))
	require.Nil(t, Choose(nil, "x"))
}
