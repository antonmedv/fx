package main

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
)

const maxHistoryBytes = 1 << 20

// Keep searches separate from JavaScript queries so neither input recalls the
// other kind. This file is used only when FX_HISTORY_FILE is set.
type inputHistory struct {
	Searches []string `json:"searches"`
	Queries  []string `json:"queries"`
}

func (m *model) loadHistory(path string) error {
	if path == "" {
		return nil
	}
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		m.historyFile = path
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()
	// Refuse an invalid or oversized file without replacing it later.
	data, err := io.ReadAll(io.LimitReader(f, maxHistoryBytes+1))
	if err != nil {
		return err
	}
	if len(data) > maxHistoryBytes {
		return errors.New("history file exceeds 1 MiB")
	}
	var history inputHistory
	if err := json.Unmarshal(data, &history); err != nil {
		return err
	}
	for _, s := range history.Searches {
		m.addSearchHistory(s)
	}
	for _, q := range history.Queries {
		m.addQueryHistory(q)
	}
	m.historyFile = path
	return nil
}

func (m *model) saveHistory() {
	if m.historyFile == "" {
		return
	}
	if err := writeHistory(m.historyFile, inputHistory{m.searchHistory, m.queryHistory}); err != nil {
		m.errorf("Cannot save history: %v", err)
	}
}

func writeHistory(path string, history inputHistory) error {
	data, err := json.Marshal(history)
	if err != nil {
		return err
	}
	if len(data)+1 > maxHistoryBytes {
		return errors.New("history exceeds 1 MiB")
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".fx-history-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(append(data, '\n')); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

func (m *model) addSearchHistory(s string) {
	if s == "" {
		return
	}
	if n := len(m.searchHistory); n > 0 && m.searchHistory[n-1] == s {
		return
	}
	m.searchHistory = append(m.searchHistory, s)
	if len(m.searchHistory) > queryHistorySize {
		m.searchHistory = m.searchHistory[len(m.searchHistory)-queryHistorySize:]
	}
	m.saveHistory()
}

func (m *model) searchHistoryPrev() {
	if m.searchHistoryIndex == 0 {
		return
	}
	if m.searchHistoryIndex == len(m.searchHistory) {
		m.searchHistoryDraft = m.searchInput.Value()
	}
	m.searchHistoryIndex--
	m.searchInput.SetValue(m.searchHistory[m.searchHistoryIndex])
	m.searchInput.CursorEnd()
}

func (m *model) searchHistoryNext() {
	if m.searchHistoryIndex >= len(m.searchHistory) {
		return
	}
	m.searchHistoryIndex++
	if m.searchHistoryIndex == len(m.searchHistory) {
		m.searchInput.SetValue(m.searchHistoryDraft)
	} else {
		m.searchInput.SetValue(m.searchHistory[m.searchHistoryIndex])
	}
	m.searchInput.CursorEnd()
}
