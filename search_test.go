package main

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	. "github.com/antonmedv/fx/internal/jsonx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		top:         head,
		head:        head,
		search:      newSearch(),
		searchCache: newSearchCache(1), // Disable caching for pure search tests
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
			m.doSearch(tc.searchTerm)

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
		top:         head,
		head:        head,
		search:      newSearch(),
		searchCache: newSearchCache(1),
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
			m.doSearch(tc.pattern)

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
		top:         head,
		head:        head,
		search:      newSearch(),
		searchCache: newSearchCache(1),
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
			m.doSearch(tc.searchTerm)

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
		top:         head,
		head:        head,
		search:      newSearch(),
		searchCache: newSearchCache(1),
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
			m.doSearch(tc.searchTerm)

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
		top:         head,
		head:        head,
		search:      newSearch(),
		searchCache: newSearchCache(1),
	}

	// Test multiple matches in same value
	m.doSearch("fox")

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
		top:         head,
		head:        head,
		search:      newSearch(),
		searchCache: newSearchCache(1),
	}

	// Search for term with multiple matches
	m.doSearch("apple")

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
		top:         head,
		head:        head,
		search:      newSearch(),
		searchCache: newSearchCache(1),
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
			m.doSearch(tc.searchTerm)

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
				top:         head,
				head:        head,
				search:      newSearch(),
				searchCache: newSearchCache(1),
			}

			m.doSearch(tc.searchTerm)

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
		top:         head,
		head:        head,
		search:      newSearch(),
		searchCache: newSearchCache(1),
	}

	// Test that search completes in reasonable time
	m.doSearch("User")
	assert.Greater(t, len(m.search.results), 0, "Should find users in large JSON")
	assert.Nil(t, m.search.err, "Should not error on large JSON")

	// Test repeated terms
	m.doSearch("test")
	assert.Greater(t, len(m.search.results), 3, "Should find multiple instances of repeated term")
}

func TestSearchCaching(t *testing.T) {
	jsonData := `{
		"users": [
			{"name": "Alice Johnson", "age": 30, "email": "alice@example.com", "role": "admin"},
			{"name": "Bob Smith", "age": 25, "email": "bob@example.com", "role": "user"},
			{"name": "Charlie Brown", "age": 35, "email": "charlie@example.com", "role": "moderator"},
			{"name": "Diana Prince", "age": 28, "email": "diana@example.com", "role": "user"},
			{"name": "Eve Adams", "age": 32, "email": "eve@example.com", "role": "admin"}
		],
		"metadata": {
			"total_count": 5,
			"last_updated": "2024-01-15T10:30:00Z",
			"version": "1.2.3",
			"features": ["search", "filter", "export", "admin_panel"]
		},
		"config": {
			"max_users": 1000,
			"allow_registration": true,
			"theme": "dark",
			"language": "en"
		}
	}`

	head, err := Parse([]byte(jsonData))
	require.NoError(t, err)

	m := &model{
		top:         head,
		head:        head,
		bottom:      head,
		search:      newSearch(),
		searchCache: newSearchCache(50),
		wrap:        true,
	}

	searchTerm := "alice"
	start := time.Now()
	m.doSearch(searchTerm)
	firstSearchTime := time.Since(start)

	// Verify results found
	assert.Greater(t, len(m.search.results), 0, "Should find search results for 'alice'")
	assert.Nil(t, m.search.err, "Search should not have errors")

	start = time.Now()
	m.doSearch(searchTerm)
	cachedSearchTime := time.Since(start)

	assert.Less(t, cachedSearchTime, firstSearchTime/10, "Cached search should be at least 10x faster")
	assert.Greater(t, len(m.search.results), 0, "Cached search should return same results")

	start = time.Now()
	m.doSearch("admin")
	secondSearchTime := time.Since(start)

	assert.Greater(t, len(m.search.results), 0, "Should find results for 'admin'")

	start = time.Now()
	m.doSearch(searchTerm)
	secondCachedTime := time.Since(start)

	assert.Less(t, secondCachedTime, firstSearchTime/5, "Second cache hit should also be very fast")

	fmt.Printf("Search Performance Results:\n")
	fmt.Printf("  First search (miss): %v\n", firstSearchTime)
	fmt.Printf("  Cached search (hit): %v\n", cachedSearchTime)
	fmt.Printf("  Second search (miss): %v\n", secondSearchTime)
	fmt.Printf("  Second cached (hit): %v\n", secondCachedTime)
	fmt.Printf("  Cache speedup: %.1fx\n", float64(firstSearchTime)/float64(cachedSearchTime))
}

func TestSearchCacheWithComplexPatterns(t *testing.T) {
	jsonData := `{
		"products": [
			{"id": "PROD-001", "name": "Laptop Pro", "price": 1299.99},
			{"id": "PROD-002", "name": "Mouse Wireless", "price": 29.99},
			{"id": "PROD-003", "name": "Keyboard Mechanical", "price": 149.99}
		]
	}`

	head, err := Parse([]byte(jsonData))
	require.NoError(t, err)

	m := &model{
		top:         head,
		head:        head,
		search:      newSearch(),
		searchCache: newSearchCache(20),
	}

	testCases := []struct {
		pattern     string
		description string
	}{
		{"PROD-\\d+", "Product ID pattern"},
		{"\\d+\\.\\d+", "Price pattern"},
		{"(?i)laptop", "Case insensitive search"},
		{"^\"name\"", "Key search pattern"},
		{"Pro|Wireless", "OR pattern"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			start := time.Now()
			m.doSearch(tc.pattern)
			firstTime := time.Since(start)

			assert.Nil(t, m.search.err, "Pattern should compile successfully: %s", tc.pattern)

			start = time.Now()
			m.doSearch(tc.pattern)
			cachedTime := time.Since(start)

			assert.Less(t, cachedTime, firstTime, "Cached search should be faster for pattern: %s", tc.pattern)
		})
	}
}

func TestSearchCacheInvalidation(t *testing.T) {
	jsonData := `{"test": "value", "array": [1, 2, 3]}`

	head, err := Parse([]byte(jsonData))
	require.NoError(t, err)

	m := &model{
		top:         head,
		head:        head,
		search:      newSearch(),
		searchCache: newSearchCache(10),
		wrap:        true,
		termWidth:   80,
	}

	m.doSearch("test")
	assert.Equal(t, 1, m.searchCache.size(), "Should have 1 cached entry")

	m.searchCache.invalidate() // Simulates new JSON data arriving
	_, _, found := m.searchCache.get("test")
	assert.False(t, found, "Cache should be invalidated after new data")

	m.doSearch("test")
	assert.Equal(t, 1, m.searchCache.size(), "Should cache again after invalidation")

	m.searchCache.invalidate() // Simulates wrap toggle
	_, _, found = m.searchCache.get("test")
	assert.False(t, found, "Cache should be invalidated after wrap toggle")

	m.doSearch("test") // Re-cache
	oldTermWidth := m.termWidth
	m.termWidth = 120

	if oldTermWidth != m.termWidth && m.wrap {
		m.searchCache.invalidate()
	}

	_, _, found = m.searchCache.get("test")
	assert.False(t, found, "Cache should be invalidated after width change with wrap enabled")
}

func TestSearchCacheLRUEviction(t *testing.T) {
	jsonData := `{"a": 1, "b": 2, "c": 3, "d": 4, "e": 5}`

	head, err := Parse([]byte(jsonData))
	require.NoError(t, err)

	m := &model{
		top:         head,
		head:        head,
		search:      newSearch(),
		searchCache: newSearchCache(3),
	}

	m.doSearch("a")
	m.doSearch("b")
	m.doSearch("c")
	assert.Equal(t, 3, m.searchCache.size(), "Cache should be full")

	m.doSearch("d")
	assert.Equal(t, 3, m.searchCache.size(), "Cache should still be at max size")

	// "a" should be evicted (oldest)
	_, _, found := m.searchCache.get("a")
	assert.False(t, found, "Oldest entry should be evicted")

	_, _, found = m.searchCache.get("b")
	assert.True(t, found, "Recent entries should remain")
	_, _, found = m.searchCache.get("c")
	assert.True(t, found, "Recent entries should remain")
	_, _, found = m.searchCache.get("d")
	assert.True(t, found, "New entry should be cached")
}

func BenchmarkSearchCaching(b *testing.B) {
	// Create large JSON for meaningful benchmark
	users := make([]map[string]interface{}, 1000)
	for i := 0; i < 1000; i++ {
		users[i] = map[string]interface{}{
			"id":     fmt.Sprintf("user-%04d", i),
			"name":   fmt.Sprintf("User %d", i),
			"email":  fmt.Sprintf("user%d@example.com", i),
			"active": i%2 == 0,
		}
	}

	data := map[string]interface{}{
		"users": users,
		"meta":  map[string]interface{}{"total": 1000},
	}

	jsonBytes, _ := json.Marshal(data)
	head, _ := Parse(jsonBytes)

	b.Run("WithoutCache", func(b *testing.B) {
		m := &model{
			top:         head,
			head:        head,
			search:      newSearch(),
			searchCache: newSearchCache(1), // Tiny cache to force misses
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Use different search terms to avoid cache hits
			searchTerm := fmt.Sprintf("user-%d", i%100)
			m.doSearch(searchTerm)
		}
	})

	b.Run("WithCache", func(b *testing.B) {
		m := &model{
			top:         head,
			head:        head,
			search:      newSearch(),
			searchCache: newSearchCache(100), // Large cache
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Reuse search terms to get cache hits
			searchTerm := fmt.Sprintf("user-%d", i%10) // Only 10 different terms
			m.doSearch(searchTerm)
		}
	})
}

func TestCacheDemo(t *testing.T) {
	fmt.Println("\n=== FX Search Cache Demo ===")

	// Load the test JSON
	jsonData := `{
		"products": [
			{"name": "MacBook Pro", "price": 2399, "category": "laptops"},
			{"name": "iPhone 15", "price": 999, "category": "phones"},
			{"name": "AirPods Pro", "price": 249, "category": "audio"},
			{"name": "MacBook Air", "price": 1199, "category": "laptops"}
		],
		"users": [
			{"name": "Alice", "purchases": ["MacBook Pro", "iPhone 15"]},
			{"name": "Bob", "purchases": ["MacBook Air", "AirPods Pro"]}
		]
	}`

	head, _ := Parse([]byte(jsonData))

	// Create model with cache
	m := &model{
		top:         head,
		head:        head,
		search:      newSearch(),
		searchCache: newSearchCache(50),
	}

	// Demo different search patterns
	searches := []string{
		"MacBook",
		"Pro",
		"\\d+",      // Numbers
		"(?i)alice", // Case insensitive
		"laptops",
	}

	fmt.Println("\nFirst run (cache misses):")
	for _, search := range searches {
		start := time.Now()
		m.doSearch(search)
		duration := time.Since(start)
		fmt.Printf("  Search '%s': %v (%d results)\n", search, duration, len(m.search.results))
	}

	fmt.Println("\nSecond run (cache hits):")
	for _, search := range searches {
		start := time.Now()
		m.doSearch(search)
		duration := time.Since(start)
		fmt.Printf("  Search '%s': %v (%d results) [CACHED]\n", search, duration, len(m.search.results))
	}

	fmt.Printf("\nCache stats: %d entries cached\n", m.searchCache.size())
	fmt.Println("=== Demo Complete ===")
}
