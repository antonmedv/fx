package pretty

import (
	"github.com/antonmedv/fx/internal/jsonx"
)

func isInlineable(n *jsonx.Node) bool {
	if n.Kind == jsonx.Object && n.Key != "" {
		return checkRecursive(n)
	}
	return false
}

func checkRecursive(n *jsonx.Node) bool {
	if n.Kind == jsonx.Array {
		return false
	}
	if n.Kind == jsonx.Object {
		if n.Size <= 2 {
			childrenInlineable := true
			n.ForEach(func(child *jsonx.Node) {
				childrenInlineable = childrenInlineable && checkRecursive(child)
			})
			return childrenInlineable
		} else {
			return false
		}
	}
	return true
}
