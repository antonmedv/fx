package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func Test_parse(t *testing.T) {
	input := `{
	"a": 1,
	"b": 2,
	"a": 3,
	"slice": [{"z": "z", "1": "1"}]
}`

	p, err := parse(json.NewDecoder(strings.NewReader(input)))
	if err != nil {
		t.Error("JSON parse error", err)
	}
	o := p.(*dict)

	expectedKeys := []string{
		"a",
		"b",
		"slice",
	}
	for i := range o.keys {
		if o.keys[i] != expectedKeys[i] {
			t.Error("Wrong key order ", i, o.keys[i], "!=", expectedKeys[i])
		}
	}

	s, ok := o.get("slice")
	if !ok {
		t.Error("slice missing")
	}
	a := s.(array)
	z := a[0].(*dict)

	expectedKeys = []string{
		"z",
		"1",
	}
	for i := range z.keys {
		if z.keys[i] != expectedKeys[i] {
			t.Error("Wrong key order for nested map ", i, z.keys[i], "!=", expectedKeys[i])
		}
	}
}
