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

func Stringify(value goja.Value, vm *goja.Runtime, depth int) string {
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
			out.WriteString(Stringify(obj.Get(key), vm, depth+1))
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

		var out strings.Builder
		out.WriteString("[")
		out.WriteString("\n")

		for i, key := range keys {
			item := arr.Get(key)
			out.WriteString(strings.Repeat("  ", depth+1))
			out.WriteString(Stringify(item, vm, depth+1))
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
