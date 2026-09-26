package complete

import (
	"slices"
	"strconv"
	"strings"

	"github.com/antonmedv/fx/internal/jsonx"
	"github.com/antonmedv/fx/internal/utils"
)

// Docs is a list of top-level documents. It may grow while being walked, as
// documents stream in.
type Docs struct {
	First *jsonx.Node
	// Next returns the document after doc, nil if none (yet). A nil Next
	// means First is the only document.
	Next func(doc *jsonx.Node) *jsonx.Node
}

func (d Docs) next(doc *jsonx.Node) *jsonx.Node {
	if d.Next == nil {
		return nil
	}
	return d.Next(doc)
}

// value is a JS value while walking a path: a JSON node, an array built by
// `[]` or `@`, or undefined. Walking is lenient: accessing a property of
// undefined gives undefined instead of a TypeError.
type value struct {
	node   *jsonx.Node
	list   []value
	isList bool
}

func (v value) isArray() bool {
	return v.isList || v.node != nil && v.node.Kind == jsonx.Array
}

func (v value) isObject() bool {
	return v.node != nil && v.node.Kind == jsonx.Object
}

// forEach calls fn for the elements of an array.
func (v value) forEach(fn func(value)) {
	if v.isList {
		for _, e := range v.list {
			fn(e)
		}
		return
	}
	forChildren(v.node, func(child *jsonx.Node) bool {
		fn(value{node: child})
		return true
	})
}

// forChildren calls fn for the children of an object or array node until
// it returns false. Collapsed nodes and wrapped strings are skipped over.
func forChildren(n *jsonx.Node, fn func(*jsonx.Node) bool) {
	if n == nil || !n.HasChildren() {
		return
	}
	it := n.Next
	if n.IsCollapsed() {
		it = n.Collapsed
	}
	for it != nil && it != n.End {
		if !fn(it) {
			return
		}
		switch {
		case it.HasChildren():
			it = it.End.Next
		case it.ChunkEnd != nil:
			it = it.ChunkEnd.Next
		default:
			it = it.Next
		}
	}
}

// unquoteKey decodes a JSON key, fast for keys without escapes.
func unquoteKey(raw string) string {
	if len(raw) >= 2 && strings.IndexByte(raw, '\\') < 0 {
		return raw[1 : len(raw)-1]
	}
	key, err := utils.Unquote(raw)
	if err != nil {
		return raw
	}
	return key
}

// keyIs reports whether the JSON key raw decodes to key.
func keyIs(raw, key string) bool {
	if strings.IndexByte(raw, '\\') < 0 {
		return len(raw) == len(key)+2 && raw[1:len(raw)-1] == key
	}
	return unquoteKey(raw) == key
}

func (v value) access(s step) value {
	if v.node != nil && v.node.Kind == jsonx.Object {
		key := s.key
		if s.kind == stepIndex {
			key = strconv.Itoa(s.index)
		}
		var found *jsonx.Node
		forChildren(v.node, func(child *jsonx.Node) bool {
			if keyIs(child.Key, key) {
				found = child // The last duplicate wins, as in JS.
			}
			return true
		})
		return value{node: found}
	}
	if v.isArray() {
		index := s.index
		if s.kind == stepKey {
			i, err := strconv.Atoi(s.key)
			if err != nil || i < 0 || strconv.Itoa(i) != s.key {
				return value{} // .length and such have no keys
			}
			index = i
		}
		if v.isList {
			if index < len(v.list) {
				return v.list[index]
			}
			return value{}
		}
		var found *jsonx.Node
		i := 0
		forChildren(v.node, func(child *jsonx.Node) bool {
			if i == index {
				found = child
				return false
			}
			i++
			return true
		})
		return value{node: found}
	}
	return value{}
}

// eval returns the value of a whole argument, as the engine computes it.
func (p path) eval(v value) value {
	return mapN(p.maps, v, func(v value) value { return evalSteps(p.steps, v) })
}

// mapN applies fn like n nested fx `@`: over the elements of an array, or
// to a non-array itself.
func mapN(n int, v value, fn func(value) value) value {
	if n == 0 {
		return fn(v)
	}
	if !v.isArray() {
		return mapN(n-1, v, fn)
	}
	var out []value
	v.forEach(func(e value) {
		out = append(out, mapN(n-1, e, fn))
	})
	return value{list: out, isList: true}
}

// evalSteps applies accesses; `[]` is a flatMap over the rest of the steps.
func evalSteps(steps []step, v value) value {
	for i, s := range steps {
		if s.kind == stepEach {
			var out []value
			if v.isArray() {
				v.forEach(func(e value) {
					r := evalSteps(steps[i+1:], e)
					if r.isArray() {
						r.forEach(func(e value) { out = append(out, e) })
					} else {
						out = append(out, r)
					}
				})
			}
			return value{list: out, isList: true}
		}
		v = v.access(s)
	}
	return v
}

// visit calls fn with every value the next accessor of a base applies to:
// `@` and `[]` visit each element instead of building arrays.
func (p path) visit(v value, fn func(value)) {
	var walk func(n int, v value)
	walk = func(n int, v value) {
		if n > 0 && v.isArray() {
			v.forEach(func(e value) { walk(n-1, e) })
		} else if n > 0 {
			walk(n-1, v)
		} else {
			visitSteps(p.steps, v, fn)
		}
	}
	walk(p.maps, v)
}

func visitSteps(steps []step, v value, fn func(value)) {
	for i, s := range steps {
		if s.kind == stepEach {
			if v.isArray() {
				v.forEach(func(e value) { visitSteps(steps[i+1:], e, fn) })
			}
			return
		}
		v = v.access(s)
	}
	fn(v)
}

// keySet is an insertion-ordered set of keys.
type keySet struct {
	keys []string
	seen map[string]struct{} // raw JSON keys
}

func (s *keySet) addChildren(n *jsonx.Node) {
	forChildren(n, func(child *jsonx.Node) bool {
		if _, ok := s.seen[child.Key]; !ok {
			if s.seen == nil {
				s.seen = make(map[string]struct{})
			}
			s.seen[child.Key] = struct{}{}
			s.keys = append(s.keys, unquoteKey(child.Key))
		}
		return true
	})
}

func (s *keySet) addAll(keys []string) {
	for _, k := range keys {
		raw := strconv.Quote(k)
		if _, ok := s.seen[raw]; !ok {
			if s.seen == nil {
				s.seen = make(map[string]struct{})
			}
			s.seen[raw] = struct{}{}
			s.keys = append(s.keys, k)
		}
	}
}

// baseKeys is what can follow a base: keys of its objects, and keys of the
// elements of its arrays.
type baseKeys struct {
	obj     keySet
	elem    keySet
	arrays  bool     // an array was seen, so `[]` may follow
	strings bool     // a string was seen, so its methods may follow
	numbers bool     // a number was seen
	engine  bool     // obj and methods come from EngineKeys
	methods []string // from EngineKeys

	first, last *jsonx.Node // documents walked, for incremental walks
}

// collect adds the keys of v.
func (k *baseKeys) collect(v value) {
	if v.isObject() {
		k.obj.addChildren(v.node)
	} else if v.isArray() {
		k.arrays = true
		v.forEach(func(e value) {
			if e.isObject() {
				k.elem.addChildren(e.node)
			}
		})
	} else if v.node != nil && v.node.Kind == jsonx.String {
		k.strings = true
	} else if v.node != nil && v.node.Kind == jsonx.Number {
		k.numbers = true
	}
}

// methodNames returns the methods of the values seen, sorted.
func (k *baseKeys) methodNames() []string {
	if k.engine {
		return sortStrings(slices.Clone(k.methods))
	}
	n := jsNames()
	var lists [][]string
	if k.arrays {
		lists = append(lists, n.array)
	}
	if k.strings {
		lists = append(lists, n.string)
	}
	if k.numbers {
		lists = append(lists, n.number)
	}
	if len(lists) == 1 {
		return lists[0]
	}
	names := slices.Concat(lists...)
	slices.Sort(names)
	return slices.Compact(names)
}

// walk adds the keys after base for documents not walked yet.
func (k *baseKeys) walk(docs Docs, args []path, base path) {
	if k.first != docs.First {
		*k = baseKeys{first: docs.First}
	}
	doc := docs.First
	if k.last != nil {
		doc = docs.next(k.last)
	}
	for ; doc != nil; doc = docs.next(doc) {
		k.last = doc
		if doc.Kind == jsonx.Err {
			continue // recovered text, not an engine input
		}
		v := value{node: doc}
		for _, arg := range args {
			v = arg.eval(v)
		}
		base.visit(v, k.collect)
	}
}

// Cache keeps the keys after recently completed bases, so that typing a key
// and streaming documents walk only what is new. Not safe for concurrent use.
type Cache struct {
	entries map[string]*baseKeys
}

const cacheSize = 32

func (c *Cache) get(key string) *baseKeys {
	if c.entries == nil || len(c.entries) >= cacheSize {
		if k, ok := c.entries[key]; ok {
			return k
		}
		c.entries = make(map[string]*baseKeys)
	}
	k, ok := c.entries[key]
	if !ok {
		k = &baseKeys{}
		c.entries[key] = k
	}
	return k
}

func (r *Request) cacheKey() string {
	return strings.Join(r.Args, "\x00") + "\x01" + r.base
}
