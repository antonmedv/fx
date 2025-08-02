package pretty

import (
	"github.com/antonmedv/fx/internal/jsonx"
)

func isInlineable(n *jsonx.Node) bool {
	if n.Kind == jsonx.Array {
		return len(n.Key) > 0 && isSimpleArray(n)
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

func isSimpleObject(n *jsonx.Node) bool {
	if n.Kind == jsonx.Object {
		// Special case for empty objects
		if n.Size == 0 {
			return true
		}

		// Special case: exactly one key with string value and len(key+value) <= 80 chars
		if n.Size == 1 {
			var hasOneStringValue bool
			var keyLength, valueLength int

			n.ForEach(func(child *jsonx.Node) {
				keyLength = len(child.Key)
				if child.Kind == jsonx.String {
					valueLength = len(child.Value)
					hasOneStringValue = true
				}
			})

			if hasOneStringValue && keyLength+valueLength <= 80 {
				return true
			}
		}

		// Original implementation
		isSimple := true
		count := 0
		numStrings := 0
		numOther := 0

		n.ForEach(func(child *jsonx.Node) {
			count++
			if len(child.Key) > 10 {
				isSimple = false
				return
			}

			if child.Kind == jsonx.String {
				numStrings++
				if len(child.Value) > 20 {
					isSimple = false
				}
			} else if child.Kind == jsonx.Number || child.Kind == jsonx.Bool || child.Kind == jsonx.Null {
				numOther++
			} else {
				isSimple = false
			}
		})

		// Apply limits based on the types of values present
		if numStrings > 2 || numOther > 3 {
			isSimple = false
		}

		return isSimple
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
