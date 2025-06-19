package jsonx_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/antonmedv/fx/internal/jsonx"
)

func TestJsonParser_Parse(t *testing.T) {
	tests := []struct {
		input    string
		wantKind jsonx.Kind
		wantErr  bool
	}{
		{`"hello"`, jsonx.String, false},
		{`42`, jsonx.Number, false},
		{`-123.45`, jsonx.Number, false},
		{`true`, jsonx.Bool, false},
		{`false`, jsonx.Bool, false},
		{`null`, jsonx.Null, false},
		{`{}`, jsonx.Object, false},
		{`[]`, jsonx.Array, false},
		{`{"key":"value"}`, jsonx.Object, false},
		{`[1, 2, 3]`, jsonx.Array, false},
		{`   "test"   `, jsonx.String, false},
		{`// comment
		"test"`, jsonx.String, false},
		{`/* comment */"test"`, jsonx.String, false},
		{`{"a":1,}`, jsonx.Object, false},
		{`[1,2,]`, jsonx.Array, false},
		{`"abc`, jsonx.Err, true},
		{`"ab\q"`, jsonx.Err, true},
		{`truth`, jsonx.Err, true},
		{`1e`, jsonx.Err, true},
		{`[1, 2`, jsonx.Err, true},
		{`/* test`, jsonx.Err, true},
		{`NaN`, jsonx.NaN, false},
		{`-NaN`, jsonx.NaN, false},
		{`nan`, jsonx.NaN, false},
		{`Infinity`, jsonx.Infinity, false},
		{`-Infinity`, jsonx.Infinity, false},
		{`infinity`, jsonx.Infinity, false},
		{`inf`, jsonx.Infinity, false},
		{`INF`, jsonx.Infinity, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			node, err := jsonx.Parse([]byte(tt.input))
			if tt.wantErr {
				assert.Error(t, err, "expected error for input: %s", tt.input)
			} else {
				assert.NoError(t, err, "unexpected error for input: %s", tt.input)
				assert.Equal(t, tt.wantKind, node.Kind, "unexpected kind for input: %s", tt.input)
			}
		})
	}
}

func TestJsonParser_Recovery(t *testing.T) {
	brokenJSON := `{ "a": 1 }here goes the text`
	t.Run("Recover", func(t *testing.T) {
		p := jsonx.NewJsonParser(strings.NewReader(brokenJSON))
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
