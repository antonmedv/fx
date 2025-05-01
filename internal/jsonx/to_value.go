package jsonx

import (
	"fmt"
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
		i, err := strconv.Atoi(string(n.Value))
		if err == nil {
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
