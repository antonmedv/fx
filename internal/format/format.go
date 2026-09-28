// Package format lists the input formats fx converts to JSON.
//
// To add a format:
//  1. write internal/<name> with a ToJSON function,
//  2. add one entry to All,
//  3. add a sample for it in TestAll (the test fails without one).
//
// Then document the flag at fx.wtf.
package format

import (
	"strings"

	"github.com/antonmedv/fx/internal/edn"
	"github.com/antonmedv/fx/internal/toml"
	"github.com/antonmedv/fx/internal/yaml"
)

// Format is an input format fx converts to JSON before parsing.
type Format struct {
	Name string   // "edn", used in messages
	Flag string   // "--edn"
	Help string   // "parse input as EDN", the usage line
	Exts []string // file extensions that select the format: ".edn"
	// ToJSON converts the whole input to a stream of JSON documents. It
	// returns an error on invalid input and never panics.
	ToJSON func(in []byte) ([]byte, error)
}

var YAML = &Format{
	Name:   "yaml",
	Flag:   "--yaml",
	Help:   "parse input as YAML",
	Exts:   []string{".yaml", ".yml"},
	ToJSON: yaml.ToJSON,
}

var TOML = &Format{
	Name:   "toml",
	Flag:   "--toml",
	Help:   "parse input as TOML",
	Exts:   []string{".toml"},
	ToJSON: toml.ToJSON,
}

var EDN = &Format{
	Name:   "edn",
	Flag:   "--edn",
	Help:   "parse input as EDN",
	Exts:   []string{".edn"},
	ToJSON: edn.ToJSON,
}

// All lists the formats in the order of the usage text.
var All = []*Format{YAML, TOML, EDN}

// ByFlag returns the format selected by a command line argument, or nil.
func ByFlag(arg string) *Format {
	for _, f := range All {
		if f.Flag == arg {
			return f
		}
	}
	return nil
}

// ByFile returns the format a file name selects by its extension, or nil.
// The comparison ignores case.
func ByFile(name string) *Format {
	name = strings.ToLower(name)
	for _, f := range All {
		for _, ext := range f.Exts {
			if strings.HasSuffix(name, ext) {
				return f
			}
		}
	}
	return nil
}
