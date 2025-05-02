package engine

import (
	"math/big"
	"reflect"
	"strconv"
	"strings"

	. "github.com/antonmedv/fx/internal/theme"
	"github.com/dop251/goja"
)

func Stringify(value goja.Value, vm *goja.Runtime, depth int) string {
	rtype := value.ExportType()
	if rtype == nil {
		return CurrentTheme.Null("null")
	}
	if isBigIntPtr(rtype) {
		bi := value.Export().(*big.Int)
		return CurrentTheme.Number(bi.String())
	}
	switch rtype.Kind() {
	case reflect.Bool:
		if value.ToBoolean() {
			return CurrentTheme.Boolean("true")
		} else {
			return CurrentTheme.Boolean("false")
		}

	case reflect.Int64, reflect.Float64:
		return CurrentTheme.Number(value.String())

	case reflect.String:
		return CurrentTheme.String(strconv.Quote(value.String()))

	case reflect.Map:
		obj := value.ToObject(vm)
		keys := obj.Keys()

		if len(keys) == 0 {
			return CurrentTheme.Syntax("{}")
		}

		var out strings.Builder
		out.WriteString(CurrentTheme.Syntax("{"))
		out.WriteString("\n")

		ident := strings.Repeat("  ", depth)
		identKey := strings.Repeat("  ", depth+1)

		for i, key := range keys {
			out.WriteString(identKey)
			out.WriteString(CurrentTheme.Key(strconv.Quote(key)))
			out.WriteString(CurrentTheme.Syntax(":"))
			out.WriteString(" ")
			out.WriteString(Stringify(obj.Get(key), vm, depth+1))
			if i < len(keys)-1 {
				out.WriteString(CurrentTheme.Syntax(","))
			}
			out.WriteString("\n")

		}

		out.WriteString(ident)
		out.WriteString(CurrentTheme.Syntax("}"))
		return out.String()

	case reflect.Slice:
		arr := value.ToObject(vm)
		keys := arr.Keys()

		if len(keys) == 0 {
			return CurrentTheme.Syntax("[]")
		}

		var out strings.Builder
		out.WriteString(CurrentTheme.Syntax("["))
		out.WriteString("\n")

		for i, key := range keys {
			item := arr.Get(key)
			out.WriteString(strings.Repeat("  ", depth+1))
			out.WriteString(Stringify(item, vm, depth+1))
			if i < len(keys)-1 {
				out.WriteString(CurrentTheme.Syntax(","))
			}
			out.WriteString("\n")
		}

		out.WriteString(strings.Repeat("  ", depth))
		out.WriteString(CurrentTheme.Syntax("]"))
		return out.String()
	}
	return value.String()
}

var bigIntPtrType = reflect.TypeOf((*big.Int)(nil))

func isBigIntPtr(t reflect.Type) bool {
	return t == bigIntPtrType
}
