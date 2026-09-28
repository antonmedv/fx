// Package yaml converts YAML to JSON.
package yaml

import (
	"bytes"
	"io"
	"math"

	goyaml "github.com/goccy/go-yaml"
)

// ToJSON converts a YAML stream to JSON, one document after another. Map
// keys keep their order.
func ToJSON(in []byte) ([]byte, error) {
	var out []byte
	decoder := goyaml.NewDecoder(
		bytes.NewReader(in),
		goyaml.UseOrderedMap(),
	)
	for {
		var v any
		if err := decoder.Decode(&v); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		j, err := goyaml.MarshalWithOptions(jsonSpecials(v), goyaml.JSON())
		if err != nil {
			return nil, err
		}
		out = append(out, j...)
	}
	return out, nil
}

// special is written to the JSON output as is.
type special string

func (s special) MarshalYAML() ([]byte, error) { return []byte(s), nil }

// jsonSpecials replaces the floats JSON cannot hold, which goyaml would
// write as .inf or .nan, with the words fx's JSON parser reads.
func jsonSpecials(v any) any {
	switch x := v.(type) {
	case goyaml.MapSlice:
		for i := range x {
			x[i].Value = jsonSpecials(x[i].Value)
		}
		return x
	case []any:
		for i := range x {
			x[i] = jsonSpecials(x[i])
		}
		return x
	case float64:
		switch {
		case math.IsInf(x, 1):
			return special("Infinity")
		case math.IsInf(x, -1):
			return special("-Infinity")
		case math.IsNaN(x):
			return special("NaN")
		}
	}
	return v
}
