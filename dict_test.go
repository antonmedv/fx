package main

import "testing"

func Test_dict(t *testing.T) {
	d := newDict()
	d.set("number", 3)
	v, _ := d.get("number")
	if v.(int) != 3 {
		t.Error("Set number")
	}
	// string
	d.set("string", "x")
	v, _ = d.get("string")
	if v.(string) != "x" {
		t.Error("Set string")
	}
	// string slice
	d.set("strings", []string{
		"t",
		"u",
	})
	v, _ = d.get("strings")
	if v.([]string)[0] != "t" {
		t.Error("Set strings first index")
	}
	if v.([]string)[1] != "u" {
		t.Error("Set strings second index")
	}
	// mixed slice
	d.set("mixed", []interface{}{
		1,
		"1",
	})
	v, _ = d.get("mixed")
	if v.([]interface{})[0].(int) != 1 {
		t.Error("Set mixed int")
	}
	if v.([]interface{})[1].(string) != "1" {
		t.Error("Set mixed string")
	}
	// overriding existing key
	d.set("number", 4)
	v, _ = d.get("number")
	if v.(int) != 4 {
		t.Error("Override existing key")
	}
	// keys
	expectedKeys := []string{
		"number",
		"string",
		"strings",
		"mixed",
	}
	for i, key := range d.keys {
		if key != expectedKeys[i] {
			t.Error("Keys method", key, "!=", expectedKeys[i])
		}
	}
	for i, key := range expectedKeys {
		if key != expectedKeys[i] {
			t.Error("Keys method", key, "!=", expectedKeys[i])
		}
	}
}
