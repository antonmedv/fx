package json

import (
	"encoding/json"
	"os"
	"strings"

	. "github.com/antonmedv/fx/pkg/dict"
)

type CommentsData map[string]*CommentData
type CommentData struct {
	CommentsBefore []Comment
	CommentAt      Comment
	CommentsAfter  []Comment
}
type Comment struct {
	Comment string
	Type    int
}

const (
	InlineComment = iota
	BlockComment  = iota
)

func Parse(dec *json.Decoder, data string) (interface{}, CommentsData, error) {
	token, err := dec.Token()
	if err != nil {
		return nil, nil, err
	}
	comments := getComments(data)
	if delim, ok := token.(json.Delim); ok {
		switch delim {
		case '{':
			output, err := decodeDict(dec)
			return output, comments, err
		case '[':
			output, err := decodeArray(dec)
			return output, comments, err
		}
	}
	return token, comments, nil
}

func getComments(data string) CommentsData {
	comments := make(CommentsData)

	// Loop until we find something that isn't a comment
	takeUntil := func(until ...string) string {
		takenData := ""
	take:
		for {
			if len(data) == 0 {
				break
			}
			// Check if any of the until strings are at the start of the data
			for _, u := range until {
				if strings.HasPrefix(data, u) {
					data = data[len(u):]
					break take
				}
			}
			takenData += string(data[0])
			data = data[1:]
		}
		return takenData
	}

	path := ""

	takeComments := func() {
		// Comments at the start
		for len(data) > 0 {
			data = strings.TrimSpace(data)
			if strings.HasPrefix(data, "//") {
				if comments[path] == nil {
					comments[path] = &CommentData{}
				}
				comments[path].CommentsBefore = append(comments[path].CommentsBefore, Comment{
					Comment: strings.Trim(strings.TrimPrefix(takeUntil("\n"), "//"), " "),
					Type:    InlineComment,
				})
			} else if strings.HasPrefix(data, "/*") {
				if comments[path] == nil {
					comments[path] = &CommentData{}
				}
				comments[path].CommentsBefore = append(comments[path].CommentsBefore, Comment{
					Comment: strings.Trim(strings.TrimPrefix(takeUntil("*/"), "/*"), " "),
					Type:    BlockComment,
				})
			} else {
				break
			}
		}
		return
	}

	takeComments()

	// Half-parse the JSON (We don't care about the values, just getting the path and comments).
	// We can't use a JSON parser because it will error on the comments.
	for len(data) > 0 {
		takeUntil("//", "/*")
		takeComments()
	}

	// Print the comments
	for k, v := range comments {
		println("'" + k + "':")
		println("  Before:")
		for _, c := range v.CommentsBefore {
			println("   | ", k, ":", strings.Split(c.Comment, "\n")[0])
		}
		println("  Inline:")
		println("   | ", k, ":", v.CommentAt.Comment)
		println("  After:")
		for _, c := range v.CommentsAfter {
			println("   | ", k, ":", c.Comment)
		}
		println("------------")
	}

	os.Exit(0)

	return comments
}

func decodeDict(dec *json.Decoder) (*Dict, error) {
	d := NewDict()
	for {
		token, err := dec.Token()
		if err != nil {
			return nil, err
		}
		if delim, ok := token.(json.Delim); ok && delim == '}' {
			return d, nil
		}
		key := token.(string)
		token, err = dec.Token()
		if err != nil {
			return nil, err
		}
		var value interface{} = token
		if delim, ok := token.(json.Delim); ok {
			switch delim {
			case '{':
				value, err = decodeDict(dec)
				if err != nil {
					return nil, err
				}
			case '[':
				value, err = decodeArray(dec)
				if err != nil {
					return nil, err
				}
			}
		}
		d.Set(key, value)
	}
}

func decodeArray(dec *json.Decoder) ([]interface{}, error) {
	slice := make(Array, 0)
	for index := 0; ; index++ {
		token, err := dec.Token()
		if err != nil {
			return nil, err
		}
		if delim, ok := token.(json.Delim); ok {
			switch delim {
			case '{':
				value, err := decodeDict(dec)
				if err != nil {
					return nil, err
				}
				slice = append(slice, value)
			case '[':
				value, err := decodeArray(dec)
				if err != nil {
					return nil, err
				}
				slice = append(slice, value)
			case ']':
				return slice, nil
			}
			continue
		}
		slice = append(slice, token)
	}
}
