package main

import (
	. "github.com/antonmedv/fx/pkg/dict"
	. "github.com/antonmedv/fx/pkg/json"
	"github.com/stretchr/testify/require"
	"regexp"
	"testing"
)

func Test_search_values(t *testing.T) {
	tests := []struct {
		name   string
		object interface{}
		want   *foundRange
	}{
		{name: "null", object: nil},
		{name: "true", object: true},
		{name: "false", object: false},
		{name: "Number", object: Number("42")},
		{name: "string", object: "Hello, World!"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &model{
				json: tt.object,
			}
			re, _ := regexp.Compile(".+")
			str := Stringify(m.json)
			indexes := re.FindAllStringIndex(str, -1)
			m.remapSearchResult(m.json, "", 0, indexes, 0, nil)

			s := &searchResult{path: ""}
			s.ranges = append(s.ranges, &foundRange{
				parent: s,
				path:   "",
				start:  0,
				end:    len(str),
				kind:   valueRange,
			})
			require.Equal(t, []*searchResult{s}, m.searchResults)
		})
	}
}

func Test_search_array(t *testing.T) {
	msg := `
	["first","second"]
	  ^^^^^   ^^^^^^
`
	m := &model{
		json: Array{"first", "second"},
	}
	re, _ := regexp.Compile("\\w+")
	indexes := re.FindAllStringIndex(Stringify(m.json), -1)
	m.remapSearchResult(m.json, "", 0, indexes, 0, nil)

	s1 := &searchResult{path: "[0]"}
	s1.ranges = append(s1.ranges,
		&foundRange{
			parent: s1,
			path:   "[0]",
			start:  1,
			end:    6,
			kind:   valueRange,
		},
	)
	s2 := &searchResult{path: "[1]", index: 1}
	s2.ranges = append(s2.ranges,
		&foundRange{
			parent: s2,
			path:   "[1]",
			start:  1,
			end:    7,
			kind:   valueRange,
		},
	)
	require.Equal(t, []*searchResult{s1, s2}, m.searchResults, msg)
}

func Test_search_between_array(t *testing.T) {
	msg := `
	["first","second"]
	  ^^^^^^^^^^^^^^
`
	m := &model{
		json: Array{"first", "second"},
	}
	re, _ := regexp.Compile("\\w.+\\w")
	indexes := re.FindAllStringIndex(Stringify(m.json), -1)
	m.remapSearchResult(m.json, "", 0, indexes, 0, nil)

	s := &searchResult{path: "[0]"}
	s.ranges = append(s.ranges,
		&foundRange{
			parent: s,
			path:   "[0]",
			start:  1,
			end:    7,
			kind:   valueRange,
		},
		&foundRange{
			parent: s,
			path:   "[0]",
			start:  0,
			end:    1,
			kind:   commaRange,
		},
		&foundRange{
			parent: s,
			path:   "[1]",
			start:  0,
			end:    7,
			kind:   valueRange,
		},
	)
	require.Equal(t, []*searchResult{s}, m.searchResults, msg)
}

func Test_search_dict(t *testing.T) {
	msg := `
	{"key": "hello world"}
	 ^^^^^  ^^^^^^^^^^^^^
`
	d := NewDict()
	d.Set("key", "hello world")
	m := &model{
		json: d,
	}
	re, _ := regexp.Compile("\"[\\w\\s]+\"")
	indexes := re.FindAllStringIndex(Stringify(m.json), -1)
	m.remapSearchResult(m.json, "", 0, indexes, 0, nil)

	s1 := &searchResult{path: ".key"}
	s1.ranges = append(s1.ranges,
		&foundRange{
			parent: s1,
			path:   ".key",
			start:  0,
			end:    5,
			kind:   keyRange,
		},
	)
	s2 := &searchResult{path: ".key", index: 1}
	s2.ranges = append(s2.ranges,
		&foundRange{
			parent: s2,
			path:   ".key",
			start:  0,
			end:    13,
			kind:   valueRange,
		},
	)
	require.Equal(t, []*searchResult{s1, s2}, m.searchResults, msg)
}

func Test_search_dict_with_array(t *testing.T) {
	msg := `
	{"first": [1,2],"second": []}
	^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
`
	d := NewDict()
	d.Set("first", Array{Number("1"), Number("2")})
	d.Set("second", Array{})
	m := &model{
		json: d,
	}
	re, _ := regexp.Compile(".+")
	indexes := re.FindAllStringIndex(Stringify(m.json), -1)
	m.remapSearchResult(m.json, "", 0, indexes, 0, nil)

	s := &searchResult{path: ""}
	s.ranges = append(s.ranges,
		/* { */ &foundRange{parent: s, path: "", start: 0, end: 1, kind: openBracketRange},
		/*   "first" */ &foundRange{parent: s, path: ".first", start: 0, end: 7, kind: keyRange},
		/*         : */ &foundRange{parent: s, path: ".first", start: 0, end: 2, kind: delimRange},
		/*            [ */ &foundRange{parent: s, path: ".first", start: 0, end: 1, kind: openBracketRange},
		/*               1 */ &foundRange{parent: s, path: ".first[0]", start: 0, end: 1, kind: valueRange},
		/*               , */ &foundRange{parent: s, path: ".first[0]", start: 0, end: 1, kind: commaRange},
		/*               2 */ &foundRange{parent: s, path: ".first[1]", start: 0, end: 1, kind: valueRange},
		/*            ] */ &foundRange{parent: s, path: ".first", start: 0, end: 1, kind: closeBracketRange},
		/*            , */ &foundRange{parent: s, path: ".first", start: 0, end: 1, kind: commaRange},
		/*   "second" */ &foundRange{parent: s, path: ".second", start: 0, end: 8, kind: keyRange},
		/*          : */ &foundRange{parent: s, path: ".second", start: 0, end: 2, kind: delimRange},
		/*             [ */ &foundRange{parent: s, path: ".second", start: 0, end: 1, kind: openBracketRange},
		/*             ] */ &foundRange{parent: s, path: ".second", start: 0, end: 1, kind: closeBracketRange},
		/* } */ &foundRange{parent: s, path: "", start: 0, end: 1, kind: closeBracketRange},
	)
	require.Equal(t, []*searchResult{s}, m.searchResults, msg)
}
