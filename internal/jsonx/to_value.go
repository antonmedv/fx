package jsonx

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"

	"github.com/dop251/goja"

	"github.com/antonmedv/fx/internal/jsonpath"
	"github.com/antonmedv/fx/internal/utils"
)

// ValueError is a value the lenient parser accepted that is not valid JSON,
// as the string "\g", so it has no JS value.
type ValueError struct {
	kind  string   // "string", "key" or "number".
	token string   // As in the input.
	path  []string // Innermost first.
}

func (e *ValueError) Error() string {
	var b strings.Builder
	b.WriteString("Invalid JSON ")
	b.WriteString(e.kind)
	b.WriteByte(' ')
	writeToken(&b, e.token)
	if len(e.path) > 0 {
		b.WriteString(" at ")
		for i := len(e.path) - 1; i >= 0; i-- {
			b.WriteString(e.path[i])
		}
	}
	return b.String()
}

// writeToken writes token shortened, with control characters escaped, so
// the message stays on one line.
func writeToken(b *strings.Builder, token string) {
	const max = 40
	n := 0
	for _, r := range token {
		if n == max {
			b.WriteString("…")
			return
		}
		n++
		switch {
		case r == '\t':
			b.WriteString(`\t`)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\r':
			b.WriteString(`\r`)
		case r < 0x20 || r == 0x7f:
			fmt.Fprintf(b, `\u%04x`, r)
		default:
			b.WriteRune(r)
		}
	}
}

// InvalidString is the error for a string token that is not valid JSON.
func InvalidString(token string) error {
	return &ValueError{kind: "string", token: token}
}

// ToValue converts n to a JS value. Values that are not valid JSON, which
// the lenient parser accepts, are a *ValueError rather than a guess.
func (n *Node) ToValue(vm *goja.Runtime) (goja.Value, error) {
	v, err := n.toValue(vm)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (n *Node) toValue(vm *goja.Runtime) (goja.Value, *ValueError) {
	switch n.Kind {
	case Null:
		return goja.Null(), nil

	case Bool:
		return vm.ToValue(n.Value == "true"), nil

	case Number:
		i, ok := ParseNumber(n.Value)
		if ok {
			return vm.ToValue(i), nil
		}
		// Out of range, as 1e400, is ±Inf, as JSON.parse reads it.
		f, err := strconv.ParseFloat(n.Value, 64)
		if err != nil && !errors.Is(err, strconv.ErrRange) {
			return nil, &ValueError{kind: "number", token: n.Value}
		}
		return vm.ToValue(f), nil

	case String:
		unquoted, err := utils.Unquote(n.Value)
		if err != nil {
			return nil, &ValueError{kind: "string", token: n.Value}
		}
		return vm.ToValue(unquoted), nil

	case Object:
		obj := vm.NewObject()

		if n.HasChildren() {
			it := n
			if it.IsCollapsed() {
				it = it.Collapsed
			} else {
				it = it.Next
			}

			for it != nil && it != n.End {
				key, err := utils.Unquote(it.Key)
				if err != nil {
					return nil, &ValueError{kind: "key", token: it.Key}
				}
				value, verr := it.toValue(vm)
				if verr != nil {
					verr.path = append(verr.path, keyPath(key))
					return nil, verr
				}
				if err := obj.Set(key, value); err != nil {
					return nil, &ValueError{kind: "key", token: it.Key}
				}

				it = it.nextSibling()
			}
		}

		return obj, nil

	case Array:
		var arr []any

		if n.HasChildren() {
			it := n
			if it.IsCollapsed() {
				it = it.Collapsed
			} else {
				it = it.Next
			}

			for i := 0; it != nil && it != n.End; i++ {
				value, verr := it.toValue(vm)
				if verr != nil {
					verr.path = append(verr.path, "["+strconv.Itoa(i)+"]")
					return nil, verr
				}
				arr = append(arr, value)

				it = it.nextSibling()
			}
		}

		return vm.NewArray(arr...), nil

	case NaN:
		return vm.ToValue(math.NaN()), nil

	case Infinity:
		if n.Value[0] == '-' {
			return vm.ToValue(math.Inf(-1)), nil
		}
		return vm.ToValue(math.Inf(1)), nil

	}
	// Undefined, and kinds that are no JSON value.
	return goja.Undefined(), nil
}

// keyPath is the path segment of an object key: .key or ["key"].
func keyPath(key string) string {
	if jsonpath.Identifier.MatchString(key) {
		return "." + key
	}
	quoted, _ := json.Marshal(key)
	return "[" + string(quoted) + "]"
}

// nextSibling returns the node after n and all its children and wrap chunks.
func (n *Node) nextSibling() *Node {
	if n.HasChildren() {
		return n.End.Next
	}
	if n.ChunkEnd != nil {
		return n.ChunkEnd.Next
	}
	return n.Next
}

// maxSafeInt is 2^53 - 1, the largest integer JS can represent exactly.
const maxSafeInt = 1<<53 - 1

// minSafeInt is -(2^53 - 1).
const minSafeInt = -maxSafeInt

// ParseNumber parses a number from a string as int64 or *big.Int.
func ParseNumber(s string) (interface{}, bool) {
	bi := new(big.Int)
	if _, ok := bi.SetString(s, 10); !ok {
		return nil, false
	}

	// Quickly reject values whose bit-length exceeds 54 (i.e. >= 2^53).
	// big.Int.BitLen returns the length of the absolute value in bits.
	if bi.BitLen() <= 53 {
		// Safe to convert to int64 and check full range.
		v := bi.Int64()
		if v >= minSafeInt && v <= maxSafeInt {
			return int(v), true
		}
	}

	return bi, true
}
