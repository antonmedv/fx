package engine_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/antonmedv/fx/internal/engine"
	"github.com/antonmedv/fx/internal/jsonx"
)

func TestFastPath_FilterStream(t *testing.T) {
	input := `{"id": 1, "name": "alice"}
{"id": 2, "name": "bob"}
{"id": 3, "name": "charlie"}
{"id": 4, "name": "david"}`

	tests := []struct {
		name    string
		arg     string
		expects []string
	}{
		{
			name: "loose equality number",
			arg:  "?.id == 2",
			expects: []string{
				"{\n  \"id\": 2,\n  \"name\": \"bob\"\n}",
			},
		},
		{
			name: "strict equality number",
			arg:  "?.id === 3",
			expects: []string{
				"{\n  \"id\": 3,\n  \"name\": \"charlie\"\n}",
			},
		},
		{
			name: "string equality double quote",
			arg:  `?.name == "david"`,
			expects: []string{
				"{\n  \"id\": 4,\n  \"name\": \"david\"\n}",
			},
		},
		{
			name: "string equality single quote",
			arg:  `?.name == 'alice'`,
			expects: []string{
				"{\n  \"id\": 1,\n  \"name\": \"alice\"\n}",
			},
		},
		{
			name: "greater than",
			arg:  "?.id > 2",
			expects: []string{
				"{\n  \"id\": 3,\n  \"name\": \"charlie\"\n}",
				"{\n  \"id\": 4,\n  \"name\": \"david\"\n}",
			},
		},
		{
			name: "greater than or equal",
			arg:  "?.id >= 3",
			expects: []string{
				"{\n  \"id\": 3,\n  \"name\": \"charlie\"\n}",
				"{\n  \"id\": 4,\n  \"name\": \"david\"\n}",
			},
		},
		{
			name: "less than",
			arg:  "?.id < 2",
			expects: []string{
				"{\n  \"id\": 1,\n  \"name\": \"alice\"\n}",
			},
		},
		{
			name: "ternary skip filter",
			arg:  "x.id == 2 ? x : skip",
			expects: []string{
				"{\n  \"id\": 2,\n  \"name\": \"bob\"\n}",
			},
		},
		{
			name: "ternary skip with this",
			arg:  "this.id == 1 ? this : skip",
			expects: []string{
				"{\n  \"id\": 1,\n  \"name\": \"alice\"\n}",
			},
		},
		{
			name: "includes substring",
			arg:  `?.name.includes("li")`,
			expects: []string{
				"{\n  \"id\": 1,\n  \"name\": \"alice\"\n}",
				"{\n  \"id\": 3,\n  \"name\": \"charlie\"\n}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := jsonx.NewJsonParser(strings.NewReader(input), false)
			exitCode, outs, errs := runEngine(parser, []string{tt.arg})
			assert.Equal(t, 0, exitCode)
			assert.Empty(t, errs)
			assert.Equal(t, tt.expects, outs)
		})
	}
}

func TestFastPath_FilterArray(t *testing.T) {
	input := `[
  {"id": 1, "active": true, "tags": ["admin", "staff"]},
  {"id": 2, "active": false, "tags": ["guest"]},
  {"id": 3, "active": true, "tags": ["staff"]}
]`

	tests := []struct {
		name    string
		arg     string
		expects []string
	}{
		{
			name: "filter boolean truthy",
			arg:  "?.active",
			expects: []string{
				"[\n  {\n    \"id\": 1,\n    \"active\": true,\n    \"tags\": [\n      \"admin\",\n      \"staff\"\n    ]\n  },\n  {\n    \"id\": 3,\n    \"active\": true,\n    \"tags\": [\n      \"staff\"\n    ]\n  }\n]",
			},
		},
		{
			name: "filter boolean falsy",
			arg:  "?!.active",
			expects: []string{
				"[\n  {\n    \"id\": 2,\n    \"active\": false,\n    \"tags\": [\n      \"guest\"\n    ]\n  }\n]",
			},
		},
		{
			name: "filter array includes",
			arg:  `?.tags.includes("admin")`,
			expects: []string{
				"[\n  {\n    \"id\": 1,\n    \"active\": true,\n    \"tags\": [\n      \"admin\",\n      \"staff\"\n    ]\n  }\n]",
			},
		},
		{
			name: "filter no matches",
			arg:  "?.id == 999",
			expects: []string{
				"[\n]",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := jsonx.NewJsonParser(strings.NewReader(input), false)
			exitCode, outs, errs := runEngine(parser, []string{tt.arg})
			assert.Equal(t, 0, exitCode)
			assert.Empty(t, errs)
			assert.Equal(t, tt.expects, outs)
		})
	}
}

func TestFastPath_SlurpFilter(t *testing.T) {
	input := `{"id": 10, "val": "first"}
{"id": 20, "val": "second"}
{"id": 30, "val": "third"}`

	rawParser := jsonx.NewJsonParser(strings.NewReader(input), false)
	slurped, err := engine.Slurp(rawParser)
	require.NoError(t, err)

	exitCode, outs, errs := runEngine(slurped, []string{"?.id == 20"})
	assert.Equal(t, 0, exitCode)
	assert.Empty(t, errs)
	require.Len(t, outs, 1)
	assert.Contains(t, outs[0], `"id": 20`)
	assert.Contains(t, outs[0], `"val": "second"`)
	assert.NotContains(t, outs[0], `"id": 10`)
	assert.NotContains(t, outs[0], `"id": 30`)
}

func TestFastPath_LooseCoercion(t *testing.T) {
	t.Run("number string loose equality", func(t *testing.T) {
		input := "{\"x\": 1}\n{\"x\": \"1\"}"
		rawParser := jsonx.NewJsonParser(strings.NewReader(input), false)
		slurped, err := engine.Slurp(rawParser)
		require.NoError(t, err)

		exitCode, outs, errs := runEngine(slurped, []string{"?.x == 1"})
		assert.Equal(t, 0, exitCode)
		assert.Empty(t, errs)
		require.Len(t, outs, 1)
		assert.Contains(t, outs[0], `"x": 1`)
		assert.Contains(t, outs[0], `"x": "1"`)
	})

	t.Run("number string strict equality distinguishes", func(t *testing.T) {
		input := "{\"x\": 1}\n{\"x\": \"1\"}"
		rawParser := jsonx.NewJsonParser(strings.NewReader(input), false)
		slurped, err := engine.Slurp(rawParser)
		require.NoError(t, err)

		exitCode, outs, errs := runEngine(slurped, []string{"?.x === 1"})
		assert.Equal(t, 0, exitCode)
		assert.Empty(t, errs)
		require.Len(t, outs, 1)
		assert.Contains(t, outs[0], `"x": 1`)
		assert.NotContains(t, outs[0], `"x": "1"`)
	})

	t.Run("null and string comparison", func(t *testing.T) {
		input := "{\"x\": null}\n{\"x\": \"1\"}"
		rawParser := jsonx.NewJsonParser(strings.NewReader(input), false)
		slurped, err := engine.Slurp(rawParser)
		require.NoError(t, err)

		exitCode, outs, errs := runEngine(slurped, []string{"?.x <= 1"})
		assert.Equal(t, 0, exitCode)
		assert.Empty(t, errs)
		require.Len(t, outs, 1)
		assert.Contains(t, outs[0], `"x": null`)
		assert.Contains(t, outs[0], `"x": "1"`)
	})
}

func TestFastPath_Lookup(t *testing.T) {
	input := `{"user": {"name": "alice", "scores": [95, 88]}, "count": 2}`

	tests := []struct {
		name    string
		arg     string
		expects []string
	}{
		{
			name:    "nested property",
			arg:     ".user.name",
			expects: []string{"alice"},
		},
		{
			name:    "array index in nested property",
			arg:     ".user.scores[0]",
			expects: []string{"95"},
		},
		{
			name:    "second array index",
			arg:     ".user.scores[1]",
			expects: []string{"88"},
		},
		{
			name:    "top-level number",
			arg:     ".count",
			expects: []string{"2"},
		},
		{
			name: "top-level object",
			arg:  ".user",
			expects: []string{
				"{\n  \"name\": \"alice\",\n  \"scores\": [\n    95,\n    88\n  ]\n}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := jsonx.NewJsonParser(strings.NewReader(input), false)
			exitCode, outs, errs := runEngine(parser, []string{tt.arg})
			assert.Equal(t, 0, exitCode)
			assert.Empty(t, errs)
			assert.Equal(t, tt.expects, outs)
		})
	}
}

func BenchmarkFastPathVsJS(b *testing.B) {
	var sb strings.Builder
	for i := 0; i < 1000; i++ {
		fmt.Fprintf(&sb, `{"id": %d, "name": "user_%d"}`+"\n", i, i)
	}
	data := sb.String()

	b.Run("Filter_FastPath", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			parser := jsonx.NewJsonParser(strings.NewReader(data), false)
			runEngine(parser, []string{"?.id == 500"})
		}
	})

	b.Run("Filter_JS_Engine", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			parser := jsonx.NewJsonParser(strings.NewReader(data), false)
			runEngine(parser, []string{"x => x.id === 500 ? x : skip"})
		}
	})

	b.Run("Lookup_FastPath", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			parser := jsonx.NewJsonParser(strings.NewReader(data), false)
			runEngine(parser, []string{".id"})
		}
	})

	b.Run("Lookup_JS_Engine", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			parser := jsonx.NewJsonParser(strings.NewReader(data), false)
			runEngine(parser, []string{"x => x.id"})
		}
	})
}
