package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHistory_SearchNavigation(t *testing.T) {
	m := newQueryModel(t, `{"first":1,"second":2}`)
	runSearch(m, "first")
	m.searchInput.SetValue("")
	runSearch(m, "second")
	result := m.search
	m.searchInput.SetValue("")
	typeKeys(m, "/draft")
	up(m)
	require.Equal(t, "second", m.searchInput.Value())
	up(m)
	up(m)
	require.Equal(t, "first", m.searchInput.Value())
	down(m)
	require.Equal(t, "second", m.searchInput.Value())
	down(m)
	down(m)
	require.Equal(t, "draft", m.searchInput.Value())
	require.Same(t, result, m.search, "browsing must not run a search")
}

func TestHistory_PersistsAcrossModels(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	m := newQueryModel(t, `{"first":1}`)
	require.NoError(t, m.loadHistory(path))
	runSearch(m, "first")
	m.addQueryHistory(".first")

	other := newQueryModel(t, `{"first":2}`)
	require.NoError(t, other.loadHistory(path))
	typeKeys(other, "/")
	up(other)
	require.Equal(t, "first", other.searchInput.Value())
	other.Update(press("esc"))
	typeKeys(other, ".")
	up(other)
	require.Equal(t, ".first", other.queryInput.Value())

	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0600), info.Mode().Perm())
	}
}

func TestHistory_CancelDoesNotPersist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	m := newQueryModel(t, `{}`)
	require.NoError(t, m.loadHistory(path))
	typeKeys(m, "/cancelled")
	m.Update(press("esc"))
	typeKeys(m, ".cancelled")
	m.Update(press("esc"))
	_, err := os.Stat(path)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestHistory_Disabled(t *testing.T) {
	m := newQueryModel(t)
	require.NoError(t, m.loadHistory(""))
	m.addSearchHistory("term")
	m.addQueryHistory(".key")
	require.Equal(t, "", m.historyFile)
	require.Nil(t, m.message)
}

func TestHistory_LimitsAndDuplicates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	m := newQueryModel(t)
	require.NoError(t, m.loadHistory(path))
	for i := range queryHistorySize + 10 {
		m.addSearchHistory(fmt.Sprintf("search %d", i))
	}
	m.addSearchHistory("search 109")
	m.addSearchHistory("")
	require.Len(t, m.searchHistory, queryHistorySize)
	require.Equal(t, "search 10", m.searchHistory[0])
	other := newQueryModel(t)
	require.NoError(t, other.loadHistory(path))
	require.Equal(t, m.searchHistory, other.searchHistory)
}

func TestHistory_LoadFailurePreservesFile(t *testing.T) {
	for name, data := range map[string]string{
		"invalid":   "not JSON",
		"oversized": strings.Repeat(" ", (1<<20)+1),
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "history.json")
			require.NoError(t, os.WriteFile(path, []byte(data), 0600))
			m := newQueryModel(t)
			require.Error(t, m.loadHistory(path))
			m.addSearchHistory("new")
			m.addQueryHistory(".new")
			requireFile(t, path, data)
		})
	}
}

func TestHistory_WriteFailureKeepsInMemoryHistory(t *testing.T) {
	m := newQueryModel(t)
	path := filepath.Join(t.TempDir(), "missing", "history.json")
	require.NoError(t, m.loadHistory(path))
	m.addSearchHistory("term")
	require.Equal(t, []string{"term"}, m.searchHistory)
	require.NotNil(t, m.message)
	require.True(t, m.message.isErr)
}

func TestHistory_LoadNormalizesEntries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	history := inputHistory{Searches: []string{"", "a", "a", " b "}, Queries: []string{".", " .key ", ".key", "x"}}
	data, err := json.Marshal(history)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0600))
	m := newQueryModel(t)
	require.NoError(t, m.loadHistory(path))
	require.Equal(t, []string{"a", " b "}, m.searchHistory)
	require.Equal(t, []string{".key"}, m.queryHistory)
}

func TestHistory_OversizedWritePreservesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	require.NoError(t, writeHistory(path, inputHistory{Searches: []string{"old"}}))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Error(t, writeHistory(path, inputHistory{Searches: []string{strings.Repeat("x", maxHistoryBytes)}}))
	requireFile(t, path, string(data))
	files, err := os.ReadDir(filepath.Dir(path))
	require.NoError(t, err)
	require.Len(t, files, 1, "no temporary files remain")
}
