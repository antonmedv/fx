package main

import (
	"fmt"
)

func stringify(v interface{}) string {
	switch v.(type) {
	case nil:
		return "null"

	case bool:
		if v.(bool) {
			return "true"
		} else {
			return "false"
		}

	case number:
		return v.(number).String()

	case string:
		return fmt.Sprintf("%q", v)

	case *dict:
		result := "{"
		for i, key := range v.(*dict).keys {
			line := fmt.Sprintf("%q", key) + ": " + stringify(v.(*dict).values[key])
			if i < len(v.(*dict).keys)-1 {
				line += ", "
			}
			result += line
		}
		return result + "}"

	case array:
		result := "["
		for i, value := range v.(array) {
			line := stringify(value)
			if i < len(v.(array))-1 {
				line += ", "
			}
			result += line
		}
		return result + "]"

	default:
		return "unknown type"
	}
}
