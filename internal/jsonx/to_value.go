package jsonx

import (
	"fmt"
	"math/big"
	"strconv"

	"github.com/dop251/goja"
)

func (n *Node) ToValue(vm *goja.Runtime) goja.Value {
	switch n.Kind {
	case Null:
		return goja.Null()

	case Bool:
		if string(n.Value) == "true" {
			return vm.ToValue(true)
		} else {
			return vm.ToValue(false)
		}

	case Number:
		i, ok := ParseNumber(string(n.Value))
		if ok {
			return vm.ToValue(i)
		}
		f, err := strconv.ParseFloat(string(n.Value), 64)
		if err == nil {
			return vm.ToValue(f)
		}
		panic(err)

	case String:
		unquoted, err := strconv.Unquote(string(n.Value))
		if err != nil {
			panic(err)
		}
		return vm.ToValue(unquoted)

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
				unquotedKey, err := strconv.Unquote(string(it.Key))
				if err != nil {
					panic(err)
				}

				err = obj.Set(unquotedKey, it.ToValue(vm))
				if err != nil {
					panic(err)
				}

				if it.HasChildren() {
					it = it.End.Next
				} else {
					it = it.Next
				}
			}
		}

		return obj

	case Array:
		var arr []any

		if n.HasChildren() {
			it := n
			if it.IsCollapsed() {
				it = it.Collapsed
			} else {
				it = it.Next
			}

			for it != nil && it != n.End {
				arr = append(arr, it.ToValue(vm))

				if it.HasChildren() {
					it = it.End.Next
				} else {
					it = it.Next
				}
			}
		}

		return vm.NewArray(arr...)
	}
	panic(fmt.Sprintf("unsupported node kind %d", n.Kind))
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
