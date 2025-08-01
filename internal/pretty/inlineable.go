package pretty

import (
	"github.com/antonmedv/fx/internal/jsonx"
)

// isInlineable determines if a JSON node should be inlined for better readability.
// We only inline simple structures that remain easily readable in a single line,
// such as short arrays of numbers or objects with few, simple values.
// This selective approach maintains readability while avoiding inlining of complex structures.
func isInlineable(n *jsonx.Node) bool {
	if n.Kind == jsonx.Array {
		return len(n.Key) > 0 && isSimpleArray(n)
	}
	if n.Kind == jsonx.Object {
		return len(n.Key) > 0 && isSimpleObject(n)
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

func isNestedArrays(n *jsonx.Node) bool {
	if n.Kind != jsonx.Array || n.Size == 0 {
		return false
	}

	isValid := true
	n.ForEach(func(child *jsonx.Node) {
		if child.Kind != jsonx.Array {
			isValid = false
			return
		}
		child.ForEach(func(innerChild *jsonx.Node) {
			if innerChild.Kind != jsonx.Number {
				isValid = false
			}
		})
	})
	return isValid
}

func isArrayOfSimpleObject(n *jsonx.Node) bool {
	if n.Kind != jsonx.Array || n.Size == 0 {
		return false
	}

	isValid := true
	n.ForEach(func(child *jsonx.Node) {
		if !isSimpleObject(child) {
			isValid = false
		}
	})
	return isValid
}
