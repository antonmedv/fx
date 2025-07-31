package pretty

import (
	"github.com/antonmedv/fx/internal/jsonx"
)

func isInlineable(n *jsonx.Node) bool {
	if n.Kind == jsonx.Array && len(n.Key) > 0 {
		return isSimpleArray(n)
	}
	if n.Kind == jsonx.Object && len(n.Key) > 0 {
		return isSimpleObject(n)
	}
	return false
}

func isSimpleObject(n *jsonx.Node) bool {
	if n.Kind == jsonx.Object {
		isSimple := true
		var firstKind jsonx.Kind
		first := true
		count := 0

		n.ForEach(func(child *jsonx.Node) {
			count++
			if len(child.Key) > 10 {
				isSimple = false
				return
			}
			if first {
				firstKind = child.Kind
				first = false
			} else if child.Kind != firstKind {
				isSimple = false
				return
			}

			if child.Kind != jsonx.Number &&
				child.Kind != jsonx.Bool &&
				(child.Kind != jsonx.String || len(child.Value) > 20) {
				isSimple = false
			}
		})

		if (firstKind == jsonx.String && count > 2) ||
			((firstKind == jsonx.Number || firstKind == jsonx.Bool) && count > 3) {
			isSimple = false
		}
		return isSimple
	}
	return false
}

func isSimpleArray(n *jsonx.Node) bool {
	if n.Kind == jsonx.Array {
		isAllNumbers := true
		count := 0
		n.ForEach(func(child *jsonx.Node) {
			count++
			if child.Kind != jsonx.Number {
				isAllNumbers = false
			}
		})
		return isAllNumbers && count > 0
	}
	return false
}
