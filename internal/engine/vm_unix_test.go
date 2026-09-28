//go:build !windows

package engine_test

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/antonmedv/fx/internal/engine"
)

// A new file gets the mode the umask allows, like any created file, while
// an existing file keeps its mode.
func TestWriteFileModes(t *testing.T) {
	old := syscall.Umask(0o077)
	t.Cleanup(func() { syscall.Umask(old) })
	dir := t.TempDir()

	created := filepath.Join(dir, "new.json")
	require.NoError(t, engine.WriteFile(created, []byte("1\n")))
	info, err := os.Stat(created)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	existing := filepath.Join(dir, "old.json")
	require.NoError(t, os.WriteFile(existing, []byte("0"), 0o644))
	require.NoError(t, os.Chmod(existing, 0o644))
	require.NoError(t, engine.WriteFile(existing, []byte("2\n")))
	info, err = os.Stat(existing)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o644), info.Mode().Perm())
	data, err := os.ReadFile(existing)
	require.NoError(t, err)
	require.Equal(t, "2\n", string(data))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 2, "no temp file left behind")
}

func TestWriteFileRefusesSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.json")
	require.NoError(t, os.WriteFile(target, []byte("0"), 0o644))
	link := filepath.Join(dir, "link.json")
	require.NoError(t, os.Symlink(target, link))

	err := engine.WriteFile(link, []byte("1\n"))
	require.ErrorContains(t, err, "symbolic link")
	data, err := os.ReadFile(target)
	require.NoError(t, err)
	require.Equal(t, "0", string(data))
}
