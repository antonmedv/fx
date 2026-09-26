package complete

import (
	"fmt"
	"github.com/antonmedv/fx/internal/jsonx"
	"strings"
	"testing"
)

// bigArray is one document of about 100k lines: an array of objects.
func bigArray() string {
	var b strings.Builder
	b.WriteString(`{"items": [`)
	for i := range 20_000 {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, `{"id": %d, "name": "n%d", "tags": ["a", "b"], "meta": {"k%d": 1}}`, i, i, i%50)
	}
	b.WriteString(`]}`)
	return b.String()
}

// bigLines is 100k JSON Lines documents.
func bigLines() string {
	var b strings.Builder
	for i := range 100_000 {
		fmt.Fprintf(&b, `{"id": %d, "name": "n%d", "user": {"login": "u", "k%d": 1}}`+"\n", i, i, i%50)
	}
	return b.String()
}

func BenchmarkWalk(b *testing.B) {
	for _, bench := range []struct {
		name, data, query string
	}{
		{"array/items[]", bigArray(), ".items."},
		{"array/items[].meta", bigArray(), ".items[].meta."},
		{"lines/root", bigLines(), "."},
		{"lines/user", bigLines(), ".user."},
	} {
		docs := sliceDocsFast(parseDocs(b, bench.data))
		b.Run(bench.name+"/cold", func(b *testing.B) {
			r, _ := ParseQuery(bench.query)
			for b.Loop() {
				var cache Cache
				cache.Complete(r, docs)
			}
		})
		b.Run(bench.name+"/cached", func(b *testing.B) {
			var cache Cache
			r, _ := ParseQuery(bench.query)
			cache.Complete(r, docs)
			r, _ = ParseQuery(bench.query + "k1")
			for b.Loop() {
				cache.Complete(r, docs)
			}
		})
	}
}

func sliceDocsFast(docs []nodeT) Docs {
	next := make(map[nodeT]nodeT, len(docs))
	for i := 1; i < len(docs); i++ {
		next[docs[i-1]] = docs[i]
	}
	return Docs{First: docs[0], Next: func(d nodeT) nodeT { return next[d] }}
}

type nodeT = *jsonx.Node
