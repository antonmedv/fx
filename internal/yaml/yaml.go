// Package yaml converts YAML to JSON.
package yaml

import (
	"bytes"
	"io"

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
		j, err := goyaml.MarshalWithOptions(v, goyaml.JSON())
		if err != nil {
			return nil, err
		}
		out = append(out, j...)
	}
	return out, nil
}
