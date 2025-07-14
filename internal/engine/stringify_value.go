package engine

import (
	"fmt"
	"math"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/dop251/goja"

	. "github.com/antonmedv/fx/internal/theme"
)

func Stringify(value goja.Value, vm *goja.Runtime, theme Theme, depth int) string {
	if value.StrictEquals(goja.Undefined()) {
		return theme.Error("undefined")
	}

	rtype := value.ExportType()
	if rtype == nil {
		return theme.Null("null")
	}

	switch rtype {
	case bigIntType:
		bi := value.Export().(*big.Int)
		return theme.Number(bi.String())
	case timeTimeType:
		t := value.Export().(time.Time)
		quoted := strconv.Quote(t.String())
		return theme.String(quoted)
	}

	switch rtype.Kind() {
	case reflect.Bool:
		if value.ToBoolean() {
			return theme.Boolean("true")
		} else {
			return theme.Boolean("false")
		}

	case reflect.Int64:
		return theme.Number(value.String())

	case reflect.Float64:
		f := value.ToFloat()
		if math.IsInf(f, 0) {
			return theme.Error(value.String())
		} else if math.IsNaN(f) {
			return theme.Error(value.String())
		}
		return theme.Number(value.String())

	case reflect.String:
		return theme.String(strconv.Quote(value.String()))

	case reflect.Map:
		obj := value.ToObject(vm)
		keys := obj.Keys()

		if len(keys) == 0 {
			return theme.Syntax("{}")
		}

		var out strings.Builder
		out.WriteString(theme.Syntax("{"))
		out.WriteString("\n")

		ident := strings.Repeat("  ", depth)
		identKey := strings.Repeat("  ", depth+1)

		for i, key := range keys {
			out.WriteString(identKey)
			out.WriteString(theme.Key(strconv.Quote(key)))
			out.WriteString(theme.Syntax(":"))
			out.WriteString(" ")
			out.WriteString(Stringify(obj.Get(key), vm, theme, depth+1))
			if i < len(keys)-1 {
				out.WriteString(theme.Syntax(","))
			}
			out.WriteString("\n")

		}

		out.WriteString(ident)
		out.WriteString(theme.Syntax("}"))
		return out.String()

	case reflect.Slice:
		arr := value.ToObject(vm)
		keys := arr.Keys()

		if len(keys) == 0 {
			return theme.Syntax("[]")
		}

		var out strings.Builder
		out.WriteString(theme.Syntax("["))
		out.WriteString("\n")

		for i, key := range keys {
			item := arr.Get(key)
			out.WriteString(strings.Repeat("  ", depth+1))
			out.WriteString(Stringify(item, vm, theme, depth+1))
			if i < len(keys)-1 {
				out.WriteString(theme.Syntax(","))
			}
			out.WriteString("\n")
		}

		out.WriteString(strings.Repeat("  ", depth))
		out.WriteString(theme.Syntax("]"))
		return out.String()
	}
	panic(fmt.Sprintf("Unsupported value type: %v", rtype.Kind()))
}
