package engine

import (
	"reflect"
	"strconv"
	"strings"

	. "github.com/antonmedv/fx/internal/theme"
	"github.com/dop251/goja"
)

func Print(value goja.Value, vm *goja.Runtime, depth int) string {
	rtype := value.ExportType()
	if rtype == nil {
		return CurrentTheme.Null("null")
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

		var out strings.Builder
		out.WriteString(CurrentTheme.Syntax("{"))
		out.WriteString("\n")

		ident := strings.Repeat("  ", depth)
		identKey := strings.Repeat("  ", depth+1)

		keys := obj.Keys()
		for i, key := range keys {
			out.WriteString(identKey)
			out.WriteString(CurrentTheme.Key(strconv.Quote(key)))
			out.WriteString(CurrentTheme.Syntax(":"))
			out.WriteString(" ")
			out.WriteString(Print(obj.Get(key), vm, depth+1))
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

		var out strings.Builder
		out.WriteString(CurrentTheme.Syntax("["))
		out.WriteString("\n")

		keys := arr.Keys()
		for i, key := range keys {
			item := arr.Get(key)
			out.WriteString(strings.Repeat("  ", depth+1))
			out.WriteString(Print(item, vm, depth+1))
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
