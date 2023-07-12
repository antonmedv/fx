package json

import (
	"encoding/json"
	"strings"
	"testing"

	. "github.com/antonmedv/fx/pkg/dict"
	"github.com/hedhyw/jsoncjson"
)

func Test_parse(t *testing.T) {
	// Comments with random whitespace
	input := `  // Before ""    
{
	"a": 1,    /* Inline ".a" */   
	"b": 2,
	 //   Before ".slice" 
	"slice": [{"z": "z",/*Inline ".slice.z" */ "1": "1"}],
	"a": 3,
	"d": 4
	/* After ".d"*/
}
 // After ""   `
	reader := jsoncjson.NewReader(strings.NewReader(input))
	p, comments, err := Parse(json.NewDecoder(reader), input)
	if err != nil {
		t.Error("JSON parse error", err)
	}
	o := p.(*Dict)

	expectedKeys := []string{
		"a",
		"b",
		"slice",
		"d",
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

	if len(comments) != 6 {
		t.Error("Wrong number of comments", len(comments))
	}

	// Map from comment to value
	expectedComments := map[string]CommentData{
		"": {
			CommentsBefore: []Comment{
				{Comment: "Before \"\"", Type: InlineComment},
			},
			CommentAt: Comment{Comment: "", Type: InlineComment},
			CommentsAfter: []Comment{
				{Comment: "After \"\"", Type: InlineComment},
			},
		},
		".a": {
			CommentsBefore: []Comment{},
			CommentAt:      Comment{Comment: "Inline \"a\"", Type: BlockComment},
			CommentsAfter:  []Comment{},
		},
		".slice": {
			CommentsBefore: []Comment{
				{Comment: "Before \".slice\"", Type: InlineComment},
			},
			CommentAt:     Comment{Comment: "", Type: InlineComment},
			CommentsAfter: []Comment{},
		},
		".slice[0].z": {
			CommentsBefore: []Comment{},
			CommentAt:      Comment{Comment: "Inline \".slice.z\"", Type: BlockComment},
			CommentsAfter:  []Comment{},
		},
		".d": {
			CommentsBefore: []Comment{},
			CommentAt:      Comment{Comment: "", Type: InlineComment},
			CommentsAfter: []Comment{
				{Comment: "After \".d\"", Type: BlockComment},
			},
		},
	}
	for key := range comments {
		expected, ok := expectedComments[key]

		if !ok {
			continue
		}

		value := *comments[key]

		if len(value.CommentsBefore) != len(expected.CommentsBefore) {
			t.Error("Wrong number of comments before", len(value.CommentsBefore), "!=", len(expected.CommentsBefore))
		}
		if len(value.CommentsAfter) != len(expected.CommentsAfter) {
			t.Error("Wrong number of comments after", len(value.CommentsAfter), "!=", len(expected.CommentsAfter))
		}

		for i := range expected.CommentsBefore {
			if expected.CommentsBefore[i] != value.CommentsBefore[i] {
				t.Error("Wrong comment before", i, expected.CommentsBefore[i], "!=", value.CommentsBefore[i])
			}
		}
		if expected.CommentAt != value.CommentAt {
			t.Error("Wrong comment at", expected.CommentAt, "!=", value.CommentAt)
		}
		for i := range expected.CommentsAfter {
			if expected.CommentsAfter[i] != value.CommentsAfter[i] {
				t.Error("Wrong comment after", i, expected.CommentsAfter[i], "!=", value.CommentsAfter[i])
			}
		}
	}
}
