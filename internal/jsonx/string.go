package jsonx

import (
	"strings"
)

const (
	curlyBracketOpen   = "{"
	curlyBracketClose  = "}"
	curlyBracketPair   = "{}"
	squareBracketOpen  = "["
	squareBracketClose = "]"
	squareBracketPair  = "[]"
	nullValue          = "null"
	trueValue          = "true"
	falseValue         = "false"
)

func (n *Node) String() string {
	var out strings.Builder

	it := n
	for it != nil {
		if it.Key != "" {
			out.WriteString(it.Key)
			out.WriteByte(':')
		}
		if it.Value != "" {
			out.WriteString(it.Value)
		}
		if it.Comma {
			out.WriteByte(',')
		}
		if it.IsCollapsed() {
			it = it.Collapsed
		} else {
			it = it.Next
		}
	}

	return out.String()
}
