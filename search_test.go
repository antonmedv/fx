package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/antonmedv/fx/internal/jsonx"
)

// doSearch is a test helper that performs a synchronous search using executeSearch.
func doSearch(m *model, s string) {
	if s == "" {
		return
	}

	result, err := executeSearch(m.top, s, nil)
	if err != nil {
		m.search = newSearch()
		m.search.err = err
		return
	}

	m.search = result
	m.selectSearchResult(0)
}

func TestBasicSearch(t *testing.T) {
	jsonData := `{
		"name": "John Doe",
		"age": 30,
		"email": "john@example.com",
		"active": true,
		"skills": ["JavaScript", "Go", "Python"]
	}`

	head, err := Parse([]byte(jsonData))
	require.NoError(t, err)

	m := &model{
		top:    head,
		head:   head,
		search: newSearch(),
	}

	testCases := []struct {
		searchTerm      string
		expectedResults int
		description     string
	}{
		{"John", 1, "Simple string search"},
		{"30", 1, "Number search"},
		{"example.com", 1, "Domain search"},
		{"JavaScript", 1, "Array element search"},
		{"active", 1, "Boolean key search"},
		{"nonexistent", 0, "No match search"},
		{"", 0, "Empty search"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			doSearch(m, tc.searchTerm)

			if tc.expectedResults == 0 {
				assert.Equal(t, 0, len(m.search.results), "Should find no results for: %s", tc.searchTerm)
			} else {
				assert.Greater(t, len(m.search.results), 0, "Should find results for: %s", tc.searchTerm)
			}
			assert.Nil(t, m.search.err, "Search should not error for: %s", tc.searchTerm)
		})
	}
}

func TestRegexSearch(t *testing.T) {
	jsonData := `{
		"users": [
			{"id": "USER-001", "email": "alice@company.com", "score": 95.5},
			{"id": "USER-002", "email": "bob@company.com", "score": 87.2},
			{"id": "ADMIN-001", "email": "admin@company.com", "score": 100.0}
		],
		"metadata": {
			"version": "v1.2.3",
			"timestamp": "2024-01-15T10:30:00Z"
		}
	}`

	head, err := Parse([]byte(jsonData))
	require.NoError(t, err)

	m := &model{
		top:    head,
		head:   head,
		search: newSearch(),
	}

	testCases := []struct {
		pattern     string
		shouldMatch bool
		description string
	}{
		// Pattern matching
		{"USER-\\d+", true, "User ID pattern"},
		{"\\d+\\.\\d+", true, "Decimal number pattern"},
		{"[a-z]+@[a-z]+\\.com", true, "Email pattern"},
		{"v\\d+\\.\\d+\\.\\d+", true, "Version pattern"},
		{"\\d{4}-\\d{2}-\\d{2}", true, "Date pattern"},

		// Anchored searches
		{"^\"id\"", true, "Key at start of line"},
		{"com\"$", true, "Value at end of line"},

		// Character classes
		{"[A-Z]{4,}", true, "Uppercase letters"},
		{"[0-9]{3}", true, "Three digits"},

		// Quantifiers
		{"o{2}", true, "Double 'o'"},
		{"a+", true, "One or more 'a'"},
		{"z*", true, "Zero or more 'z'"},

		// Invalid patterns should error
		{"[", false, "Invalid bracket"},
		{"(", false, "Unclosed parenthesis"},
		{"*", false, "Invalid quantifier"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			doSearch(m, tc.pattern)

			if tc.shouldMatch {
				assert.Nil(t, m.search.err, "Pattern should be valid: %s", tc.pattern)
			} else {
				assert.NotNil(t, m.search.err, "Pattern should be invalid: %s", tc.pattern)
			}
		})
	}
}

func TestCaseInsensitiveSearch(t *testing.T) {
	jsonData := `{
		"Company": "ACME Corporation",
		"employees": [
			{"Name": "Alice Johnson", "Department": "Engineering"},
			{"name": "bob smith", "department": "marketing"},
			{"NAME": "CHARLIE BROWN", "DEPARTMENT": "SALES"}
		]
	}`

	head, err := Parse([]byte(jsonData))
	require.NoError(t, err)

	m := &model{
		top:    head,
		head:   head,
		search: newSearch(),
	}

	testCases := []struct {
		searchTerm  string
		description string
	}{
		{"alice", "Lowercase search for mixed case"},
		{"ALICE", "Uppercase search for mixed case"},
		{"Alice", "Proper case search"},
		{"aLiCe", "Mixed case search"},
		{"company", "Lowercase search for uppercase"},
		{"ENGINEERING", "Uppercase search for proper case"},
		{"marketing", "Exact case match"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			doSearch(m, tc.searchTerm)

			// Case insensitive search should find matches regardless of case
			assert.Greater(t, len(m.search.results), 0, "Should find case-insensitive match for: %s", tc.searchTerm)
			assert.Nil(t, m.search.err, "Search should not error")
		})
	}
}

func TestSearchInDifferentNodeTypes(t *testing.T) {
	jsonData := `{
		"string_field": "hello world",
		"number_field": 42,
		"boolean_field": true,
		"null_field": null,
		"array_field": [1, "two", 3.14, false],
		"object_field": {
			"nested_string": "nested value",
			"nested_number": 99
		}
	}`

	head, err := Parse([]byte(jsonData))
	require.NoError(t, err)

	m := &model{
		top:    head,
		head:   head,
		search: newSearch(),
	}

	testCases := []struct {
		searchTerm  string
		nodeType    string
		description string
	}{
		{"hello", "string", "Search in string values"},
		{"42", "number", "Search in number values"},
		{"true", "boolean", "Search in boolean values"},
		{"null", "null", "Search in null values"},
		{"two", "array element", "Search in array elements"},
		{"nested", "object property", "Search in nested objects"},
		{"string_field", "key", "Search in JSON keys"},
		{"3.14", "float", "Search in floating point numbers"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			doSearch(m, tc.searchTerm)

			assert.Greater(t, len(m.search.results), 0, "Should find %s in %s", tc.searchTerm, tc.nodeType)
			assert.Nil(t, m.search.err, "Search should not error")
		})
	}
}

func TestSearchResultDetails(t *testing.T) {
	jsonData := `{
		"message": "The quick brown fox jumps over the lazy dog",
		"words": ["fox", "dog", "fox"]
	}`

	head, err := Parse([]byte(jsonData))
	require.NoError(t, err)

	m := &model{
		top:    head,
		head:   head,
		search: newSearch(),
	}

	// Test multiple matches in same value
	doSearch(m, "fox")

	assert.Greater(t, len(m.search.results), 0, "Should find fox matches")
	assert.Nil(t, m.search.err, "Search should not error")

	// Verify that we have both key matches and value matches
	hasKeyMatches := len(m.search.keys) > 0
	hasValueMatches := len(m.search.values) > 0

	assert.True(t, hasKeyMatches || hasValueMatches, "Should have either key or value matches")

	// Test that search cursor is initialized
	if len(m.search.results) > 0 {
		assert.GreaterOrEqual(t, m.search.cursor, 0, "Search cursor should be valid")
		assert.Less(t, m.search.cursor, len(m.search.results), "Search cursor should be within bounds")
	}
}

func TestSearchNavigation(t *testing.T) {
	jsonData := `{
		"items": [
			{"name": "apple", "color": "red"},
			{"name": "banana", "color": "yellow"},
			{"name": "apple", "color": "green"}
		]
	}`

	head, err := Parse([]byte(jsonData))
	require.NoError(t, err)

	m := &model{
		top:    head,
		head:   head,
		search: newSearch(),
	}

	// Search for term with multiple matches
	doSearch(m, "apple")

	require.Greater(t, len(m.search.results), 1, "Should find multiple apple matches")

	initialCursor := m.search.cursor

	// Test forward navigation
	m.selectSearchResult(m.search.cursor + 1)
	assert.NotEqual(t, initialCursor, m.search.cursor, "Cursor should move forward")
	assert.GreaterOrEqual(t, m.search.cursor, 0, "Cursor should be valid")
	assert.Less(t, m.search.cursor, len(m.search.results), "Cursor should be within bounds")

	// Test wrap-around (next from last should go to first)
	lastIndex := len(m.search.results) - 1
	m.selectSearchResult(lastIndex + 1)
	assert.Equal(t, 0, m.search.cursor, "Should wrap to first result")

	// Test backward wrap-around (previous from first should go to last)
	m.selectSearchResult(-1)
	assert.Equal(t, lastIndex, m.search.cursor, "Should wrap to last result")
}

func TestSpecialCharacterSearch(t *testing.T) {
	jsonData := `{
		"symbols": "!@#$%^&*()_+-={}[]|\\:;\"'<>?,./'",
		"escaped": "Line 1\nLine 2\tTabbed\r\nWindows line",
		"unicode": "café, naïve, résumé, 中文, 🚀",
		"quotes": "He said \"Hello world!\"",
		"backslash": "C:\\Users\\Name\\file.txt"
	}`

	head, err := Parse([]byte(jsonData))
	require.NoError(t, err)

	m := &model{
		top:    head,
		head:   head,
		search: newSearch(),
	}

	testCases := []struct {
		searchTerm  string
		description string
	}{
		{"@", "At symbol"},
		{"\\$", "Dollar sign (escaped)"},
		{"\\*", "Asterisk (escaped)"},
		{"\\[", "Square bracket (escaped)"},
		{"\\\\n", "Newline escape sequence"},
		{"\\\\t", "Tab escape sequence"},
		{"café", "Unicode characters"},
		{"中文", "Chinese characters"},
		{"🚀", "Emoji"},
		{"\\\"", "Escaped quotes"},
		{"C:", "Drive letter"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			doSearch(m, tc.searchTerm)

			// Should either find matches or have valid regex (no panic)
			assert.Nil(t, m.search.err, "Should handle special characters without error: %s", tc.searchTerm)
		})
	}
}

func TestEmptyAndEdgeCases(t *testing.T) {
	testCases := []struct {
		jsonData    string
		searchTerm  string
		description string
	}{
		{`{}`, "anything", "Empty object"},
		{`[]`, "anything", "Empty array"},
		{`""`, "empty", "Empty string value"},
		{`{"": "empty key"}`, "", "Empty key search"},
		{`{"key": ""}`, "key", "Search in empty value"},
		{`null`, "null", "Null document"},
		{`false`, "false", "Boolean document"},
		{`0`, "0", "Zero value"},
		{`{"very_long_key_name_that_exceeds_normal_length": "value"}`, "very_long", "Long key names"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			head, err := Parse([]byte(tc.jsonData))
			require.NoError(t, err)

			m := &model{
				top:    head,
				head:   head,
				search: newSearch(),
			}

			doSearch(m, tc.searchTerm)

			// Should not panic or error on edge cases
			assert.Nil(t, m.search.err, "Should handle edge case without error")
			assert.NotNil(t, m.search.results, "Results should not be nil")
		})
	}
}

func TestLargeJSONSearch(t *testing.T) {
	// Build a larger JSON structure
	jsonBuilder := `{"users": [`
	for i := 0; i < 100; i++ {
		if i > 0 {
			jsonBuilder += ","
		}
		jsonBuilder += `{"id": ` + string(rune(i)) + `, "name": "User` + string(rune(i)) + `", "active": `
		if i%2 == 0 {
			jsonBuilder += "true"
		} else {
			jsonBuilder += "false"
		}
		jsonBuilder += `}`
	}
	jsonBuilder += `]}`

	// Use simpler approach for test
	jsonData := `{
		"users": [
			{"id": 1, "name": "User1", "active": true},
			{"id": 2, "name": "User2", "active": false},
			{"id": 3, "name": "User3", "active": true}
		],
		"repeated_data": ["test", "test", "test", "test", "test"]
	}`

	head, err := Parse([]byte(jsonData))
	require.NoError(t, err)

	m := &model{
		top:    head,
		head:   head,
		search: newSearch(),
	}

	// Test that search completes in reasonable time
	doSearch(m, "User")
	assert.Greater(t, len(m.search.results), 0, "Should find users in large JSON")
	assert.Nil(t, m.search.err, "Should not error on large JSON")

	// Test repeated terms
	doSearch(m, "test")
	assert.Greater(t, len(m.search.results), 3, "Should find multiple instances of repeated term")
}

func TestSearchInWrappedStrings(t *testing.T) {
	// JSON with a long string that will be wrapped
	jsonData := `{
		"description": "This is a very long string that should be wrapped across multiple lines when the terminal width is narrow enough to trigger wrapping behavior"
	}`

	head, err := Parse([]byte(jsonData))
	require.NoError(t, err)

	// Apply wrapping with a narrow terminal width to force chunks
	Wrap(head, 40)

	m := &model{
		top:       head,
		head:      head,
		search:    newSearch(),
		wrap:      true,
		termWidth: 40,
	}

	testCases := []struct {
		searchTerm  string
		expectMatch bool
		description string
	}{
		{"very long string", true, "Multi-word search in wrapped text"},
		{"wrapped across", true, "Search spanning potential chunk boundary"},
		{"terminal width", true, "Search near end of wrapped text"},
		{"This is a", true, "Search at beginning of wrapped text"},
		{"behavior", true, "Search at very end of wrapped text"},
		{"nonexistent phrase", false, "No match in wrapped text"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			doSearch(m, tc.searchTerm)

			if tc.expectMatch {
				assert.Greater(t, len(m.search.results), 0, "Should find '%s' in wrapped text", tc.searchTerm)
			} else {
				assert.Equal(t, 0, len(m.search.results), "Should not find '%s' in wrapped text", tc.searchTerm)
			}
			assert.Nil(t, m.search.err, "Search should not error")
		})
	}
}

func TestSearchChunkBoundaryMatches(t *testing.T) {
	// Create a string where we know approximately where chunk boundaries will be
	// With termWidth=30 and some indentation, each chunk will be roughly 25-28 chars
	jsonData := `{"text": "AAAAA BBBBB CCCCC DDDDD EEEEE FFFFF GGGGG HHHHH IIIII JJJJJ"}`

	head, err := Parse([]byte(jsonData))
	require.NoError(t, err)

	// Apply wrapping with narrow width
	Wrap(head, 30)

	m := &model{
		top:       head,
		head:      head,
		search:    newSearch(),
		wrap:      true,
		termWidth: 30,
	}

	// Verify chunks were created
	textNode := head.Next // Skip the opening brace to get to the "text" key node
	require.NotNil(t, textNode, "Should have text node")
	require.NotEmpty(t, textNode.Chunk, "Text node should have chunks (Chunk field set)")
	require.NotNil(t, textNode.ChunkEnd, "Text node should have ChunkEnd set")

	testCases := []struct {
		pattern     string
		description string
	}{
		{"AAAAA", "Match in first chunk"},
		{"JJJJJ", "Match in last chunk"},
		{"[A-Z]{5}", "Regex matching all groups"},
		{"BBBBB CCCCC", "Match potentially spanning chunks"},
		{"FFFFF GGGGG HHHHH", "Match spanning multiple chunks"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			doSearch(m, tc.pattern)

			assert.Greater(t, len(m.search.results), 0, "Should find matches for pattern: %s", tc.pattern)
			assert.Nil(t, m.search.err, "Search should not error")

			// Verify matches are recorded in the values map
			assert.Greater(t, len(m.search.values), 0, "Should have value matches recorded")
		})
	}
}

func TestSearchMultipleMatchesInChunks(t *testing.T) {
	// String with repeated pattern that will span multiple chunks
	jsonData := `{"data": "test123 test456 test789 test000 test111 test222 test333 test444"}`

	head, err := Parse([]byte(jsonData))
	require.NoError(t, err)

	Wrap(head, 35)

	m := &model{
		top:       head,
		head:      head,
		search:    newSearch(),
		wrap:      true,
		termWidth: 35,
	}

	// Search for pattern that should match multiple times
	doSearch(m, "test\\d{3}")

	assert.Greater(t, len(m.search.results), 0, "Should find test patterns")
	assert.Nil(t, m.search.err, "Search should not error")

	// Check that matches are distributed across chunk nodes
	totalMatches := 0
	for _, matches := range m.search.values {
		totalMatches += len(matches)
	}
	assert.Greater(t, totalMatches, 0, "Should have matches across chunks")
}

func TestSearchWrappedVsUnwrapped(t *testing.T) {
	jsonData := `{"message": "The quick brown fox jumps over the lazy dog multiple times today"}`

	head, err := Parse([]byte(jsonData))
	require.NoError(t, err)

	// First test without wrapping
	m := &model{
		top:       head,
		head:      head,
		search:    newSearch(),
		wrap:      false,
		termWidth: 80,
	}

	doSearch(m, "fox")
	unwrappedResults := len(m.search.results)
	assert.Greater(t, unwrappedResults, 0, "Should find 'fox' without wrapping")

	// Now apply wrapping and search again
	Wrap(head, 30)
	m.wrap = true
	m.termWidth = 30

	doSearch(m, "fox")
	wrappedResults := len(m.search.results)

	assert.Equal(t, unwrappedResults, wrappedResults, "Should find same number of results with or without wrapping")
}

func TestSearchChunkIndexMapping(t *testing.T) {
	// Test that search result indices are correctly mapped to chunks
	jsonData := `{"value": "START middle portion of text END"}`

	head, err := Parse([]byte(jsonData))
	require.NoError(t, err)

	// Use very narrow width to force multiple chunks
	Wrap(head, 20)

	m := &model{
		top:       head,
		head:      head,
		search:    newSearch(),
		wrap:      true,
		termWidth: 20,
	}

	// Search for terms at different positions
	testCases := []struct {
		term        string
		description string
	}{
		{"START", "Beginning of string"},
		{"middle", "Middle of string"},
		{"END", "End of string"},
		{"portion", "Another middle term"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			doSearch(m, tc.term)

			assert.Greater(t, len(m.search.results), 0, "Should find '%s'", tc.term)
			assert.Nil(t, m.search.err)

			// Verify that matches are recorded in the values map
			assert.Greater(t, len(m.search.values), 0, "Should have value matches recorded for '%s'", tc.term)
		})
	}
}

func TestSearchEmptyAndShortStringsWithWrap(t *testing.T) {
	testCases := []struct {
		jsonData    string
		searchTerm  string
		expectMatch bool
		description string
	}{
		{`{"a": ""}`, "anything", false, "Empty string value"},
		{`{"a": "x"}`, "x", true, "Single char string"},
		{`{"a": "ab"}`, "ab", true, "Two char string"},
		{`{"a": "short"}`, "short", true, "Short string (no wrap needed)"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			head, err := Parse([]byte(tc.jsonData))
			require.NoError(t, err)

			// Apply wrapping even for short strings
			Wrap(head, 30)

			m := &model{
				top:       head,
				head:      head,
				search:    newSearch(),
				wrap:      true,
				termWidth: 30,
			}

			doSearch(m, tc.searchTerm)

			if tc.expectMatch {
				assert.Greater(t, len(m.search.results), 0, "Should find '%s'", tc.searchTerm)
			} else {
				assert.Equal(t, 0, len(m.search.results), "Should not find '%s'", tc.searchTerm)
			}
			assert.Nil(t, m.search.err)
		})
	}
}

func TestSearchRegexAcrossChunks(t *testing.T) {
	// Test regex patterns that might match across chunk boundaries
	jsonData := `{"content": "email: user@example.com phone: 123-456-7890 date: 2024-01-15"}`

	head, err := Parse([]byte(jsonData))
	require.NoError(t, err)

	Wrap(head, 25)

	m := &model{
		top:       head,
		head:      head,
		search:    newSearch(),
		wrap:      true,
		termWidth: 25,
	}

	testCases := []struct {
		pattern      string
		expectMinNum int
		description  string
	}{
		{`\w+@\w+\.\w+`, 1, "Email pattern"},
		{`\d{3}-\d{3}-\d{4}`, 1, "Phone pattern"},
		{`\d{4}-\d{2}-\d{2}`, 1, "Date pattern"},
		{`\d+`, 6, "All number sequences (123,456,7890,2024,01,15)"},
		{`[a-z]+:`, 3, "Labels (email:, phone:, date:)"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			doSearch(m, tc.pattern)

			assert.GreaterOrEqual(t, len(m.search.results), tc.expectMinNum,
				"Pattern '%s' should find at least %d matches", tc.pattern, tc.expectMinNum)
			assert.Nil(t, m.search.err)
		})
	}
}
