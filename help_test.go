package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHelpSections(t *testing.T) {
	out := help(keyMap)
	for _, want := range []string{
		"Key Bindings",
		"Commands",
		":<n>",
		":q",
		"Query",
		"Syntax",
		".items[].name",
		"Input Keys",
		"tab, shift+tab",
		"Press q or ? to close",
	} {
		require.Contains(t, out, want)
	}
}

func firstTuesdayOfDecember(year int) time.Time {
	t := time.Date(year, time.December, 1, 12, 0, 0, 0, time.UTC)
	for t.Weekday() != time.Tuesday {
		t = t.AddDate(0, 0, 1)
	}
	return t
}

func TestShowLetter_OncePerYearBeforeChristmas(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "fx")
	dec10 := time.Date(2026, time.December, 10, 12, 0, 0, 0, time.UTC)

	require.True(t, showLetter(dec10, dir), "first run in December shows the letter")
	data, err := os.ReadFile(filepath.Join(dir, "letter"))
	require.NoError(t, err)
	require.Equal(t, "2026\n", string(data))

	require.False(t, showLetter(dec10, dir), "second run is quiet")
	require.False(t, showLetter(dec10.AddDate(0, 0, 5), dir), "later in the same December is quiet")
	require.False(t, showLetter(firstTuesdayOfDecember(2026), dir), "first Tuesday is quiet once shown")

	require.True(t, showLetter(dec10.AddDate(1, 0, 0), dir), "next year shows again")
	data, err = os.ReadFile(filepath.Join(dir, "letter"))
	require.NoError(t, err)
	require.Equal(t, "2027\n", string(data))
}

func TestShowLetter_OutsideWindow(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "fx")
	for _, d := range []time.Time{
		time.Date(2026, time.November, 30, 12, 0, 0, 0, time.UTC),
		time.Date(2026, time.December, 25, 12, 0, 0, 0, time.UTC),
		time.Date(2026, time.December, 31, 12, 0, 0, 0, time.UTC),
		time.Date(2027, time.January, 1, 12, 0, 0, 0, time.UTC),
	} {
		require.False(t, showLetter(d, dir), d.Format(time.DateOnly))
	}
	_, err := os.Stat(filepath.Join(dir, "letter"))
	require.True(t, os.IsNotExist(err), "no marker written outside the window")
}

func TestShowLetter_FallsBackToTuesdayWhenUnwritable(t *testing.T) {
	// A directory path under a regular file can never be created.
	file := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(file, nil, 0o644))
	unwritable := filepath.Join(file, "fx")

	for _, dir := range []string{unwritable, ""} {
		tuesday := firstTuesdayOfDecember(2026)
		require.True(t, showLetter(tuesday, dir), "first Tuesday shows")
		require.True(t, showLetter(tuesday, dir), "no marker, so it shows again on the same day")
		require.False(t, showLetter(tuesday.AddDate(0, 0, 1), dir), "day after first Tuesday is quiet")
		require.False(t, showLetter(tuesday.AddDate(0, 0, 7), dir), "second Tuesday is quiet")
	}
}
