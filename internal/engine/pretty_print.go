package engine

import (
	"reflect"
	"strings"

	. "github.com/antonmedv/fx/internal/theme"
	"github.com/dop251/goja"
)

func PrettyPrint(value goja.Value, vm *goja.Runtime, depth int) string {
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
		return CurrentTheme.String(value.String())
	case reflect.Map:
		obj := value.ToObject(vm)

		var out strings.Builder
		out.WriteString(CurrentTheme.Syntax("{"))

		ident := strings.Repeat("  ", depth)
		for _, key := range obj.Keys() {
			out.WriteString(ident)
			out.WriteString(key)
			out.WriteString(": \n")
		}
		out.WriteString("}\n")
		return out.String()

	}
	return value.String()
}
