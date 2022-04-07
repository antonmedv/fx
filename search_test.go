package main

import (
	"github.com/stretchr/testify/require"
	"regexp"
	"testing"
)

func Test_model_remapSearchResult_array(t *testing.T) {
	msg := `
	["first", "second"]
	  ^^^^^    ^^^^^^
`
	m := &model{
		json: array{"first", "second"},
	}
	re, _ := regexp.Compile("\\w+")
	indexes := re.FindAllStringIndex(stringify(m.json), -1)
	m.remapSearchResult(m.json, "", 0, indexes, 0, nil)

	s1 := &searchResult{path: "[0]"}
	s1.ranges = append(s1.ranges,
		&foundRange{
			parent: s1,
			start:  1,
			end:    6,
			kind:   valueRange,
		},
	)
	s2 := &searchResult{path: "[1]", index: 1}
	s2.ranges = append(s2.ranges,
		&foundRange{
			parent: s2,
			start:  1,
			end:    7,
			kind:   valueRange,
		},
	)
	require.Equal(t, []*searchResult{s1, s2}, m.searchResults, msg)
}

func Test_model_remapSearchResult_between_array(t *testing.T) {
	msg := `
	["first", "second"]
	  ^^^^^^^^^^^^^^^
`
	m := &model{
		json: array{"first", "second"},
	}
	re, _ := regexp.Compile("\\w.+\\w")
	indexes := re.FindAllStringIndex(stringify(m.json), -1)
	m.remapSearchResult(m.json, "", 0, indexes, 0, nil)

	s := &searchResult{path: "[0]"}
	s.ranges = append(s.ranges,
		&foundRange{
			parent: s,
			start:  1,
			end:    7,
			kind:   valueRange,
		},
		&foundRange{
			parent: s,
			start:  0,
			end:    7,
			kind:   valueRange,
		},
	)
	require.Equal(t, []*searchResult{s}, m.searchResults, msg)
}

func Test_model_remapSearchResult_dict(t *testing.T) {
	msg := `
	{"key": "hello world"}
	 ^^^^^  ^^^^^^^^^^^^^
`
	d := newDict()
	d.set("key", "hello world")
	m := &model{
		json: d,
	}
	re, _ := regexp.Compile("\"[\\w\\s]+\"")
	indexes := re.FindAllStringIndex(stringify(m.json), -1)
	m.remapSearchResult(m.json, "", 0, indexes, 0, nil)

	s1 := &searchResult{path: ".key"}
	s1.ranges = append(s1.ranges,
		&foundRange{
			parent: s1,
			start:  0,
			end:    5,
			kind:   keyRange,
		},
	)
	s2 := &searchResult{path: ".key", index: 1}
	s2.ranges = append(s2.ranges,
		&foundRange{
			parent: s2,
			start:  0,
			end:    13,
			kind:   valueRange,
		},
	)
	require.Equal(t, []*searchResult{s1, s2}, m.searchResults, msg)
}
