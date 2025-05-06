package jsonx

import (
	"strings"
)

func (n *Node) String() string {
	var out strings.Builder

	it := n
	for it != nil {
		if it.Key != nil {
			out.Write(it.Key)
			out.WriteByte(':')
		}
		if it.Value != nil {
			out.Write(it.Value)
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
