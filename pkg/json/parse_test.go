package json

import (
	"encoding/json"
	"strings"
	"testing"

	. "github.com/antonmedv/fx/pkg/dict"
)

func Test_parse(t *testing.T) {
	input := `{
	"a": 1,
	"b": 2,
	"a": 3,
	"slice": [{"z": "z", "1": "1"}]
}`
	// TODO: Test comments
	p, _, err := Parse(json.NewDecoder(strings.NewReader(input)), "")
	if err != nil {
		t.Error("JSON parse error", err)
	}
	o := p.(*Dict)

	expectedKeys := []string{
		"a",
		"b",
		"slice",
	}
	for i := range o.Keys {
		if o.Keys[i] != expectedKeys[i] {
			t.Error("Wrong key order ", i, o.Keys[i], "!=", expectedKeys[i])
		}
	}

	s, ok := o.Get("slice")
	if !ok {
		t.Error("slice missing")
	}
	a := s.(Array)
	z := a[0].(*Dict)

	expectedKeys = []string{
		"z",
		"1",
	}
	for i := range z.Keys {
		if z.Keys[i] != expectedKeys[i] {
			t.Error("Wrong key order for nested map ", i, z.Keys[i], "!=", expectedKeys[i])
		}
	}
}
