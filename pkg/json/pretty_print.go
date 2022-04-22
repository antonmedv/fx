package json

import (
	"encoding/json"
	"fmt"
	"github.com/antonmedv/fx/pkg/dict"
	"github.com/antonmedv/fx/pkg/theme"
	"strings"
)

func PrettyPrint(v interface{}, level int, theme theme.Theme) string {
	ident := strings.Repeat("  ", level)
	subident := strings.Repeat("  ", level-1)
	switch v.(type) {
	case nil:
		return theme.Null("null")

	case bool:
		if v.(bool) {
			return theme.Boolean("true")
		}
		return theme.Boolean("false")

	case json.Number:
		return theme.Number(v.(json.Number).String())

	case string:
		return theme.String(fmt.Sprintf("%q", v))

	case *dict.Dict:
		keys := v.(*dict.Dict).Keys
		if len(keys) == 0 {
			return theme.Syntax("{}")
		}
		output := theme.Syntax("{")
		output += "\n"
		for i, k := range keys {
			key := theme.Key(i, len(keys))(fmt.Sprintf("%q", k))
			value, _ := v.(*dict.Dict).Get(k)
			delim := theme.Syntax(": ")
			line := ident + key + delim + PrettyPrint(value, level+1, theme)
			if i < len(keys)-1 {
				line += theme.Syntax(",")
			}
			line += "\n"
			output += line
		}
		return output + subident + theme.Syntax("}")

	case []interface{}:
		slice := v.([]interface{})
		if len(slice) == 0 {
			return theme.Syntax("[]")
		}
		output := theme.Syntax("[\n")
		for i, value := range v.([]interface{}) {
			line := ident + PrettyPrint(value, level+1, theme)
			if i < len(slice)-1 {
				line += ",\n"
			} else {
				line += "\n"
			}
			output += line
		}
		return output + subident + theme.Syntax("]")

	default:
		return "unknown type"
	}
}
