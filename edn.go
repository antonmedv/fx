package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"gopkg.in/edn.v1"
)

// parseEDN converts EDN values into a JSON stream for fx's existing parser.
// EDN-only values are intentionally reduced to their familiar data shapes.
func parseEDN(src []byte) ([]byte, error) {
	decoder := edn.NewDecoder(bytes.NewReader(src))
	// Keep tagged values' contents and ignore the built-in tag interpretation.
	if err := decoder.AddTagFn("inst", func(s string) (string, error) { return s, nil }); err != nil {
		return nil, err
	}
	if err := decoder.AddTagFn("base64", func(s string) (string, error) { return s, nil }); err != nil {
		return nil, err
	}

	var out bytes.Buffer
	for {
		var value any
		if err := decoder.Decode(&value); err != nil {
			if err == io.EOF {
				return out.Bytes(), nil
			}
			return nil, err
		}
		value = normalizeEDN(value)
		b, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("convert EDN value to JSON: %w", err)
		}
		out.Write(b)
		out.WriteByte('\n')
	}
}

func normalizeEDN(value any) any {
	switch value := value.(type) {
	case edn.Keyword:
		return ":" + string(value)
	case edn.Symbol:
		return string(value)
	case rune:
		return string(value)
	case edn.Tag:
		return normalizeEDN(value.Value)
	case []any:
		for i := range value {
			value[i] = normalizeEDN(value[i])
		}
		return value
	case map[any]bool:
		items := make([]any, 0, len(value))
		for item := range value {
			items = append(items, normalizeEDN(unwrapEDNKey(item)))
		}
		sort.Slice(items, func(i, j int) bool {
			return ednMapKey(items[i]) < ednMapKey(items[j])
		})
		return items
	case map[any]any:
		result := make(map[string]any, len(value))
		for key, item := range value {
			key = unwrapEDNKey(key)
			result[ednMapKey(normalizeEDN(key))] = normalizeEDN(item)
		}
		return result
	default:
		return value
	}
}

func unwrapEDNKey(key any) any {
	if keyPtr, ok := key.(*any); ok {
		return *keyPtr
	}
	return key
}

func ednMapKey(key any) string {
	if s, ok := key.(string); ok {
		return s
	}
	b, err := json.Marshal(key)
	if err != nil {
		return fmt.Sprint(key)
	}
	return string(b)
}
