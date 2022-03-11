package main

import (
	"fmt"
	"reflect"
	"regexp"
	"testing"
)

func Test_splitByFoundIndexes(t *testing.T) {
	s := fmt.Sprintf("%q", "0 aaa 123 \"bbb 44\" ccc 5")
	re := regexp.MustCompile("\"?\\d+\"?")
	indexes := re.FindAllStringIndex(s, -1)
	chunks := explode(s, indexes)
	expected := []string{"", "\"0", " aaa ", "123", " \\\"bbb ", "44", "\\\" ccc ", "5\"", ""}
	ok := reflect.DeepEqual(chunks, expected)
	if !ok {
		t.Errorf(
			"split error:\n"+
				"       got %v,\n"+
				"  expected %v",
			stringify(chunks),
			stringify(expected),
		)
	}
}

func Test_splitByFoundIndexes_empty(t *testing.T) {
	s := fmt.Sprintf("%q", "foo")
	chunks := explode(s, nil)
	expected := []string{"\"foo\""}
	ok := reflect.DeepEqual(chunks, expected)
	if !ok {
		t.Errorf(
			"split error:\n"+
				"       got %v,\n"+
				"  expected %v",
			stringify(chunks),
			stringify(expected),
		)
	}
}
