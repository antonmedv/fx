package jsonx_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/antonmedv/fx/internal/jsonx"
)

func TestJsonParser_Parse(t *testing.T) {
	tests := []struct {
		input    string
		wantKind jsonx.Kind
	}{
		{`"hello"`, jsonx.String},
		{`42`, jsonx.Number},
		{`-123.45`, jsonx.Number},
		{`true`, jsonx.Bool},
		{`false`, jsonx.Bool},
		{`null`, jsonx.Null},
		{`{}`, jsonx.Object},
		{`[]`, jsonx.Array},
		{`{"key":"value"}`, jsonx.Object},
		{`[1, 2, 3]`, jsonx.Array},
		{`   "test"   `, jsonx.String},
		{`// comment
		"test"`, jsonx.String},
		{`/* comment */"test"`, jsonx.String},
		{`{"a":1,}`, jsonx.Object},
		{`[1,2,]`, jsonx.Array},
		{`NaN`, jsonx.NaN},
		{`-NaN`, jsonx.NaN},
		{`nan`, jsonx.NaN},
		{`Infinity`, jsonx.Infinity},
		{`-Infinity`, jsonx.Infinity},
		{`infinity`, jsonx.Infinity},
		{`inf`, jsonx.Infinity},
		{`INF`, jsonx.Infinity},
		{`undefined`, jsonx.Undefined},
		{`"\g"`, jsonx.String},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			node, err := jsonx.Parse([]byte(tt.input))
			assert.NoError(t, err, "unexpected error for input: %s", tt.input)
			assert.Equal(t, tt.wantKind, node.Kind, "unexpected kind for input: %s", tt.input)
		})
	}
}

func TestJsonParser_Parse_error(t *testing.T) {
	tests := []struct {
		input string
	}{
		{`"abc`},
		{`truth`},
		{`1e`},
		{`[1, 2`},
		{`/* test`},
		{`[,]`},
		{`{,}`},
		{`[1,,]`},
		{`{"a":1,,}`},
		{`-null`},
		{`Null`},
		{`-Null`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := jsonx.Parse([]byte(tt.input))
			assert.Error(t, err, "expected error for input: %s", tt.input)
		})
	}
}

func TestJsonParser_Parse_strict(t *testing.T) {
	tests := []struct {
		input string
	}{
		{`{"a":1,}`},
		{`[1,2,]`},
		{`NaN`},
		{`-NaN`},
		{`nan`},
		{`Infinity`},
		{`-Infinity`},
		{`infinity`},
		{`inf`},
		{`INF`},
		{`-null`},
		{`Null`},
		{`-Null`},
		{`/*comment*/ 42`},
		{`undefined`},
		{`"\g"`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			parser := jsonx.NewJsonParser(strings.NewReader(tt.input), true)
			_, err := parser.Parse()
			assert.Error(t, err, "expected error for input: %s", tt.input)
		})
	}
}

func TestJsonParser_Recovery(t *testing.T) {
	brokenJSON := `{ "a": 1 }here goes the text`
	t.Run("Recover", func(t *testing.T) {
		p := jsonx.NewJsonParser(strings.NewReader(brokenJSON), false)
		_, _ = p.Parse() // trigger error
		node := p.Recover()

		assert.Equal(t, jsonx.Err, node.Kind, "expected recovery node to be of Kind Err")
		assert.NotEmpty(t, node.Value, "expected recovery node to contain error snippet")
		assert.Equal(t, string(node.Value), "here goes the text", "expected recovery node to contain error snippet")
	})
}

func TestJsonParser_NestedStructureVerification(t *testing.T) {
	input := `{
		"user": {
			"name": "John",
			"age": 30,
			"active": true,
			"contacts": {
				"email": "john@example.com",
				"phone": "123456789"
			},
			"roles": ["admin", "editor"]
		}
	}`

	node, err := jsonx.Parse([]byte(input))
	assert.NoError(t, err)
	assert.Equal(t, jsonx.Object, node.Kind)

	// user object
	user := node.FindByPath([]any{"user"})
	assert.NotNil(t, user)
	assert.Equal(t, jsonx.Object, user.Kind)

	// user.name
	name := node.FindByPath([]any{"user", "name"})
	assert.NotNil(t, name)
	assert.Equal(t, jsonx.String, name.Kind)
	assert.Equal(t, `"John"`, string(name.Value))

	// user.age
	age := node.FindByPath([]any{"user", "age"})
	assert.NotNil(t, age)
	assert.Equal(t, jsonx.Number, age.Kind)
	assert.Equal(t, "30", string(age.Value))

	// user.active
	active := node.FindByPath([]any{"user", "active"})
	assert.NotNil(t, active)
	assert.Equal(t, jsonx.Bool, active.Kind)
	assert.Equal(t, "true", string(active.Value))

	// user.contacts.email
	email := node.FindByPath([]any{"user", "contacts", "email"})
	assert.NotNil(t, email)
	assert.Equal(t, jsonx.String, email.Kind)
	assert.Equal(t, `"john@example.com"`, string(email.Value))

	// user.roles[1]
	role := node.FindByPath([]any{"user", "roles", 1})
	assert.NotNil(t, role)
	assert.Equal(t, jsonx.String, role.Kind)
	assert.Equal(t, `"editor"`, string(role.Value))
}

func TestJsonParser_FindByPathWithCollapsedNodes(t *testing.T) {
	t.Run("collapsed array access", func(t *testing.T) {
		node, err := jsonx.Parse([]byte(`{
			"items": [1, 2, 3, 4, 5]
		}`))
		require.NoError(t, err)

		node.CollapseRecursively()

		items := node.FindByPath([]any{"items"})
		require.NotNil(t, items)

		element := node.FindByPath([]any{"items", 2})
		require.NotNil(t, element)
		require.Equal(t, jsonx.Number, element.Kind)
		require.Equal(t, "3", element.Value)
	})

	t.Run("collapsed object access", func(t *testing.T) {
		node, err := jsonx.Parse([]byte(`{
			"user": {
				"settings": {
					"theme": "dark",
					"notifications": true
				}
			}
		}`))
		require.NoError(t, err)

		node.CollapseRecursively()

		settings := node.FindByPath([]any{"user", "settings"})
		require.NotNil(t, settings)

		theme := node.FindByPath([]any{"user", "settings", "theme"})
		require.NotNil(t, theme)
		require.Equal(t, jsonx.String, theme.Kind)
		require.Equal(t, `"dark"`, string(theme.Value))
	})

	t.Run("nested collapsed structures", func(t *testing.T) {
		node, err := jsonx.Parse([]byte(`{
			"data": {
				"users": [
					{"id": 1, "name": "John"},
					{"id": 2, "name": "Jane"}
				]
			}
		}`))
		require.NoError(t, err)

		node.CollapseRecursively()

		users := node.FindByPath([]any{"data", "users"})
		require.NotNil(t, users)

		userName := node.FindByPath([]any{"data", "users", 1, "name"})
		require.NotNil(t, userName)
		require.Equal(t, jsonx.String, userName.Kind)
		require.Equal(t, `"Jane"`, string(userName.Value))
	})

	t.Run("nested collapsed structures with arrays", func(t *testing.T) {
		node, err := jsonx.Parse([]byte(`{
		  "data": [
			{
			  "first": [
				"tmp",
				{
				  "foo": [
					1,
					2,
					true
				  ]
				}
			  ]
			},
			{
			  "second": []
			}
		  ]
		}`))
		require.NoError(t, err)

		node.CollapseRecursively()

		value := node.FindByPath([]any{"data", 0, "first", 1, "foo", 2})
		require.NotNil(t, value)
		require.Equal(t, jsonx.Bool, value.Kind)
	})
}
