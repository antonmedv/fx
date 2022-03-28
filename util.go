package main

import (
	"encoding/json"
	"fmt"
	"github.com/charmbracelet/lipgloss"
)

func clamp(v, low, high int) int {
	if high < low {
		low, high = high, low
	}
	return min(high, max(low, v))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

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

	case json.Number:
		return v.(json.Number).String()

	case string:
		return fmt.Sprintf("%q", v)

	case *dict:
		result := "{"
		for i, key := range v.(*dict).keys {
			line := fmt.Sprintf("%q", key) + ":" + stringify(v.(*dict).values[key])
			if i < len(v.(*dict).keys)-1 {
				line += ","
			}
			result += line
		}
		return result + "}"

	case array:
		result := "["
		for i, value := range v.(array) {
			line := stringify(value)
			if i < len(v.(array))-1 {
				line += ","
			}
			result += line
		}
		return result + "]"

	default:
		return "unknown type"
	}
}

func width(s string) int {
	return lipgloss.Width(s)
}

func accessor(path string, to interface{}) string {
	return fmt.Sprintf("%v[%v]", path, to)
}
