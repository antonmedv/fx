package json

import (
	"encoding/json"
	"fmt"
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
	// This function gets the comments from the json file. It first finds the positions of the comments, then parses the JSON and relates the comments to the JSON elements.
	comments := make(CommentsData)

	// Find the positions of the comments.
	// Comments may be inline or block comments.
	// Inline comments are of the form "// comment"
	// Block comments are of the form "/* comment */"
	type commentPosition struct {
		Position int
		Type     int
		Content  string
	}

	commentPositions := make([]commentPosition, 0)

	// Remove all the strings from the data
	// This is done so that the comments inside strings are not counted
	const (
		NoString          = iota
		SingleQuoteString = iota
		DoubleQuoteString = iota
	)
	stringType := NoString
	escape := false
	for index := 0; index < len(data); index++ {
		char := string(data[index])

		if escape {
			escape = false
			continue
		}

		if char == "\"" && stringType == NoString {
			stringType = DoubleQuoteString
		} else if char == "\"" && stringType == DoubleQuoteString {
			stringType = NoString
		} else if char == "'" && stringType == NoString {
			stringType = SingleQuoteString
		} else if char == "'" && stringType == SingleQuoteString {
			stringType = NoString
		} else if char == "\\" {
			escape = true
		}

		// If we're not in a string, check for comments
		if stringType == NoString {
			dataPart := data[index:]
			// Inline comments
			if strings.HasPrefix(dataPart, "//") {
				// Inline comment
				comment := ""
				for _, c := range dataPart {
					if c == '\n' {
						break
					}
					comment += string(c)
				}

				commentPositions = append(commentPositions, commentPosition{
					Position: index,
					Type:     InlineComment,
					Content:  comment,
				})

				// Remove the comment from the data
				data = data[:index] + data[index+len(comment):]
			}
			// Block comments
			if strings.HasPrefix(dataPart, "/*") {
				// Block comment
				comment := ""
				for _, c := range dataPart {
					if strings.HasSuffix(comment, "*/") {
						break
					}
					comment += string(c)
				}

				commentPositions = append(commentPositions, commentPosition{
					Position: index,
					Type:     BlockComment,
					Content:  comment,
				})

				// Remove the comment from the data
				data = data[:index] + data[index+len(comment):]
			}
		}
	}

	// Data should now be parsable by the JSON decoder. Parse it and find the path to every character.
	characterToPath, err := getPathCharacters(data)
	if err != nil {
		return nil
	}

	maxKey := 0
	for k := range characterToPath {
		if k > int64(maxKey) {
			maxKey = int(k)
		}
	}

	// For each comment, find the path
	totalAdded := 0
	for _, c := range commentPositions {
		character := int64(c.Position)
		if character < 0 {
			character = 0
		}
		if character > int64(maxKey)-1 {
			character = int64(maxKey)
		}

		commentPath, ok := characterToPath[character]
		if !ok {
			println("didn't exist uh oh ", c.Position, character, totalAdded, len(data))
			continue
		}

		_, ok = comments[commentPath]
		if !ok {
			comments[commentPath] = &CommentData{
				CommentsBefore: []Comment{},
				CommentAt:      Comment{},
				CommentsAfter:  []Comment{},
			}
		}

		content := c.Content
		if c.Type == InlineComment {
			content = strings.TrimPrefix(content, "//")
			content = strings.TrimSpace(content)
		} else if c.Type == BlockComment {
			content = strings.TrimPrefix(content, "/*")
			content = strings.TrimSuffix(content, "*/")
			lines := strings.Split(content, "\n")
			for i, line := range lines {
				lines[i] = strings.Trim(line, " \t")
			}
			content = strings.Join(lines, "\n")
			println(content)
		}

		comments[commentPath].CommentsBefore = append(comments[commentPath].CommentsBefore, Comment{
			Comment: content,
			Type:    c.Type,
		})
		totalAdded += len(c.Content)
	}

	// Print the comments
	// for k, v := range comments {
	// 	println("'" + k + "':")
	// 	println("  Before:")
	// 	for _, c := range v.CommentsBefore {
	// 		println("   | ", k, ":", strings.Split(c.Comment, "\n")[0])
	// 	}
	// 	println("  Inline:")
	// 	println("   | ", k, ":", v.CommentAt.Comment)
	// 	println("  After:")
	// 	for _, c := range v.CommentsAfter {
	// 		println("   | ", k, ":", c.Comment)
	// 	}
	// 	println("------------")
	// }

	// os.Exit(0)

	return comments
}

func getPathCharacters(data string) (map[int64]string, error) {
	dec := json.NewDecoder(strings.NewReader(data))
	characterToPath := make(map[int64]string)

	path := ""

	characterToPath[0] = path

	oldOffset := dec.InputOffset()
	token, err := dec.Token()
	offset := dec.InputOffset()
	for i := oldOffset; i < offset; i++ {
		characterToPath[i] = path
	}

	if err != nil {
		return nil, err
	}
	if delim, ok := token.(json.Delim); ok {
		switch delim {
		case '{':
			newMap, err := getDictCharacters(dec, path)
			for k, v := range newMap {
				characterToPath[k] = v
			}
			return characterToPath, err
		case '[':
			newMap, err := getArrayCharacters(dec, path)
			for k, v := range newMap {
				characterToPath[k] = v
			}
			return characterToPath, err
		}
	}

	return characterToPath, nil
}

func getDictCharacters(dec *json.Decoder, path string) (map[int64]string, error) {
	characterToPath := make(map[int64]string)

	oldOffset := dec.InputOffset()
	for {
		token, err := dec.Token()
		if err != nil {
			return nil, err
		}
		if delim, ok := token.(json.Delim); ok && delim == '}' {
			offset := dec.InputOffset()
			for i := oldOffset; i < offset; i++ {
				_, ok = characterToPath[i]
				if !ok {
					characterToPath[i] = path
				}
			}
			return characterToPath, nil
		}
		key := token.(string)

		subpath := path + "." + key

		token, err = dec.Token()
		if err != nil {
			return nil, err
		}

		if delim, ok := token.(json.Delim); ok {
			switch delim {
			case '{':
				newMap, err := getDictCharacters(dec, subpath)
				if err != nil {
					return nil, err
				}
				for k, v := range newMap {
					characterToPath[k] = v
				}
			case '[':
				newMap, err := getArrayCharacters(dec, subpath)
				if err != nil {
					return nil, err
				}
				for k, v := range newMap {
					characterToPath[k] = v
				}
			}
		}
	}
}

func getArrayCharacters(dec *json.Decoder, path string) (map[int64]string, error) {
	characterToPath := make(map[int64]string)
	for index := 0; ; index++ {
		token, err := dec.Token()
		if err != nil {
			return nil, err
		}
		subpath := path + "[" + fmt.Sprint(index) + "]"
		if delim, ok := token.(json.Delim); ok {
			switch delim {
			case '{':
				newMap, err := getDictCharacters(dec, subpath)
				if err != nil {
					return nil, err
				}
				for k, v := range newMap {
					characterToPath[k] = v
				}
			case '[':
				newMap, err := getArrayCharacters(dec, subpath)
				if err != nil {
					return nil, err
				}
				for k, v := range newMap {
					characterToPath[k] = v
				}
			case ']':
				return characterToPath, nil
			}
			continue
		}
	}
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
