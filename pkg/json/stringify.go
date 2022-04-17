package json

import (
	"fmt"
	. "github.com/antonmedv/fx/pkg/dict"
)

func Stringify(v interface{}) string {
	switch v.(type) {
	case nil:
		return "null"

	case bool:
		if v.(bool) {
			return "true"
		} else {
			return "false"
		}

	case Number:
		return v.(Number).String()

	case string:
		return fmt.Sprintf("%q", v)

	case *Dict:
		result := "{"
		for i, key := range v.(*Dict).Keys {
			line := fmt.Sprintf("%q", key) + ": " + Stringify(v.(*Dict).Values[key])
			if i < len(v.(*Dict).Keys)-1 {
				line += ","
			}
			result += line
		}
		return result + "}"

	case Array:
		result := "["
		for i, value := range v.(Array) {
			line := Stringify(value)
			if i < len(v.(Array))-1 {
				line += ","
			}
			result += line
		}
		return result + "]"

	default:
		return "unknown type"
	}
}
