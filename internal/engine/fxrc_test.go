package engine

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadFxrc_UnreadableIsSkipped(t *testing.T) {
	if runtime.GOOS == "windows" || os.Getuid() == 0 {
		t.Skip("file permissions don't stop reading")
	}
	cwd := t.TempDir()
	home := t.TempDir()
	t.Chdir(cwd)
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_CONFIG_DIRS", t.TempDir())

	unreadable := filepath.Join(cwd, ".fxrc.js")
	require.NoError(t, os.WriteFile(unreadable, []byte("const a = 1"), 0o000))
	require.NoError(t, os.WriteFile(filepath.Join(home, ".fxrc.js"), []byte("const b = 2"), 0o644))

	fxrc, errs := readFxrc()
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), unreadable)
	assert.Contains(t, fxrc, "const b = 2", "the readable one is still loaded")
	assert.NotContains(t, fxrc, "const a = 1")
}
