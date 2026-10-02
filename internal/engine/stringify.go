package engine

import (
	"fmt"
	"math"
	"math/big"
	"reflect"
	"strings"
	"time"

	"github.com/dop251/goja"
)

// Stringify serializes value as indented JSON. A circular structure throws
// a TypeError, as JSON.stringify does, instead of recursing forever.
func Stringify(value goja.Value, vm *goja.Runtime, depth int) string {
	s := stringifier{vm: vm}
	return s.stringify(value, depth)
}

type stringifier struct {
	vm    *goja.Runtime
	stack []*goja.Object // The objects and arrays being serialized.
}

// enter pushes obj, throwing if it is already being serialized.
func (s *stringifier) enter(obj *goja.Object) {
	for _, o := range s.stack {
		if o.SameAs(obj) {
			panic(s.vm.NewTypeError("Converting circular structure to JSON"))
		}
	}
	s.stack = append(s.stack, obj)
}

func (s *stringifier) leave() {
	s.stack = s.stack[:len(s.stack)-1]
}

func (s *stringifier) stringify(value goja.Value, depth int) string {
	vm := s.vm
	rtype := value.ExportType()
	if rtype == nil {
		// Convert both null and undefined to null (save as JSON.stringify)
		return "null"
	}

	switch rtype {
	case bigIntType:
		bi := value.Export().(*big.Int)
		return bi.String()
	case timeTimeType:
		t := value.Export().(time.Time)
		quoted := Quote(t.String())
		return quoted
	}

	switch rtype.Kind() {
	case reflect.Bool:
		if value.ToBoolean() {
			return "true"
		} else {
			return "false"
		}

	case reflect.Int64:
		return value.String()

	case reflect.Float64:
		f := value.ToFloat()
		if math.IsInf(f, 0) {
			return value.String()
		} else if math.IsNaN(f) {
			return value.String()
		}
		return value.String()

	case reflect.String:
		return Quote(value.String())

	case reflect.Map:
		obj := value.ToObject(vm)
		keys := obj.Keys()

		if len(keys) == 0 {
			return "{}"
		}
		s.enter(obj)
		defer s.leave()

		var out strings.Builder
		out.WriteString("{")
		out.WriteString("\n")

		ident := strings.Repeat("  ", depth)
		identKey := strings.Repeat("  ", depth+1)

		for i, key := range keys {
			out.WriteString(identKey)
			out.WriteString(Quote(key))
			out.WriteString(":")
			out.WriteString(" ")
			out.WriteString(s.stringify(obj.Get(key), depth+1))
			if i < len(keys)-1 {
				out.WriteString(",")
			}
			out.WriteString("\n")

		}

		out.WriteString(ident)
		out.WriteString("}")
		return out.String()

	case reflect.Slice:
		arr := value.ToObject(vm)
		keys := arr.Keys()

		if len(keys) == 0 {
			return "[]"
		}
		s.enter(arr)
		defer s.leave()

		var out strings.Builder
		out.WriteString("[")
		out.WriteString("\n")

		for i, key := range keys {
			item := arr.Get(key)
			out.WriteString(strings.Repeat("  ", depth+1))
			out.WriteString(s.stringify(item, depth+1))
			if i < len(keys)-1 {
				out.WriteString(",")
			}
			out.WriteString("\n")
		}

		out.WriteString(strings.Repeat("  ", depth))
		out.WriteString("]")
		return out.String()
	}
	panic(fmt.Sprintf("Unsupported value type: %v", rtype.Kind()))
}
