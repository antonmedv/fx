package toml

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
	"github.com/pelletier/go-toml/v2/unstable"

	"github.com/antonmedv/fx/internal/engine"
)

type jnode interface{}

type jobject struct {
	fields []jfield
}

type jfield struct {
	key string
	val jnode
}

type jarray struct {
	elems []jnode
}

func ToJSON(in []byte) ([]byte, error) {
	var typed any
	if err := toml.Unmarshal(in, &typed); err != nil {
		panic(in)
	}

	root := &jobject{}
	p := unstable.Parser{}
	p.Reset(in)

	aotActive := map[string]int{} // path -> current index for that AOT
	aotCount := map[string]int{}  // path -> how many elems seen
	currentTablePath := []string{}

	for p.NextExpression() {
		e := p.Expression()
		switch e.Kind {
		case unstable.Table:
			currentTablePath = keyParts(e.Key())
			_ = ensureContainer(root, currentTablePath, aotActive)

		case unstable.ArrayTable:
			path := keyParts(e.Key())
			k := dot(path)
			idx := aotCount[k]
			aotCount[k] = idx + 1
			aotActive[k] = idx
			currentTablePath = path
			_ = ensureContainer(root, path, aotActive)

		case unstable.KeyValue:
			rel := keyParts(e.Key())
			// Resolve the actual typed value from the fully-decoded structure.
			val, ok := lookupTyped(typed, currentTablePath, rel, aotActive)
			if !ok {
				continue // skip gracefully on weird edge cases
			}
			obj := ensureContainer(root, currentTablePath, aotActive)
			setNested(obj, rel, toJ(val))
		}
	}
	if err := p.Error(); err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := writeJSON(&buf, root); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func keyParts(it unstable.Iterator) []string {
	var out []string
	for it.Next() {
		out = append(out, string(it.Node().Data))
	}
	return out
}
func dot(parts []string) string { return strings.Join(parts, ".") }

func ensureContainer(root *jobject, tablePath []string, aotActive map[string]int) *jobject {
	cur := root
	if len(tablePath) == 0 {
		return cur
	}

	prefix := []string{}
	for i, seg := range tablePath {
		prefix = append(prefix, seg)
		pkey := dot(prefix)

		// If this prefix is an active AOT, ensure an array & select the element.
		if idx, ok := aotActive[pkey]; ok {
			// Ensure field seg is an array
			f, exists := getField(cur, seg)
			if !exists {
				f = jfield{key: seg, val: &jarray{}}
				cur.fields = append(cur.fields, f)
			}
			arr, ok := f.val.(*jarray)
			if !ok {
				// Convert/replace if necessary
				arr = &jarray{}
				replaceField(cur, seg, arr)
			}
			// Ensure element at idx is an object
			for len(arr.elems) <= idx {
				arr.elems = append(arr.elems, &jobject{})
			}
			elemObj, _ := arr.elems[idx].(*jobject)
			if elemObj == nil {
				elemObj = &jobject{}
				arr.elems[idx] = elemObj
			}
			cur = elemObj
			continue
		}

		// Regular table segment: ensure an object at seg
		child, ok := getField(cur, seg)
		if !ok {
			obj := &jobject{}
			cur.fields = append(cur.fields, jfield{key: seg, val: obj})
			cur = obj
			continue
		}
		if obj, ok := child.val.(*jobject); ok {
			cur = obj
		} else {
			newObj := &jobject{}
			replaceField(cur, seg, newObj)
			cur = newObj
		}

		// If this isn't the last segment and we just ensured parent exists,
		// loop continues to drill down.
		if i == len(tablePath)-1 {
			// current table container reached
		}
	}
	return cur
}

func setNested(obj *jobject, parts []string, val jnode) {
	if len(parts) == 0 {
		return
	}
	cur := obj
	for i := 0; i < len(parts)-1; i++ {
		k := parts[i]
		f, ok := getField(cur, k)
		if !ok {
			n := &jobject{}
			cur.fields = append(cur.fields, jfield{key: k, val: n})
			cur = n
			continue
		}
		if as, ok := f.val.(*jobject); ok {
			cur = as
			continue
		}
		// Replace if something else is there
		n := &jobject{}
		replaceField(cur, k, n)
		cur = n
	}
	// final key
	last := parts[len(parts)-1]
	if _, ok := getField(cur, last); ok {
		replaceField(cur, last, val)
	} else {
		cur.fields = append(cur.fields, jfield{key: last, val: val})
	}
}

func getField(obj *jobject, key string) (jfield, bool) {
	for _, f := range obj.fields {
		if f.key == key {
			return f, true
		}
	}
	return jfield{}, false
}
func replaceField(obj *jobject, key string, val jnode) {
	for i := range obj.fields {
		if obj.fields[i].key == key {
			obj.fields[i].val = val
			return
		}
	}
	obj.fields = append(obj.fields, jfield{key: key, val: val})
}

// Convert typed TOML -> jnode. Note: order inside *inline tables* cannot be
// recovered from the typed map; if you need that too, we’d have to walk the
// value expression via unstable APIs as well.
func toJ(v any) jnode {
	switch x := v.(type) {
	case map[string]any:
		obj := &jobject{}
		// Map iteration order is undefined; this only affects inline tables.
		for k, vv := range x {
			obj.fields = append(obj.fields, jfield{key: k, val: toJ(vv)})
		}
		return obj
	case []any:
		arr := &jarray{elems: make([]jnode, len(x))}
		for i := range x {
			arr.elems[i] = toJ(x[i])
		}
		return arr
	default:
		return x // primitives, time.Time, etc. json.Marshal will handle them.
	}
}

func lookupTyped(typed any, tablePath, rel []string, aotActive map[string]int) (any, bool) {
	x := typed
	prefix := []string{}
	// Walk table path, applying AOT indices when present.
	for _, k := range tablePath {
		mp, ok := asMap(x)
		if !ok {
			return nil, false
		}
		var ok2 bool
		x, ok2 = mp[k]
		if !ok2 {
			return nil, false
		}
		prefix = append(prefix, k)
		if idx, ok := aotActive[dot(prefix)]; ok {
			arr, ok := x.([]any)
			if !ok || idx >= len(arr) {
				return nil, false
			}
			x = arr[idx]
		}
	}
	// Then the dotted relative key
	for _, k := range rel {
		mp, ok := asMap(x)
		if !ok {
			return nil, false
		}
		var ok2 bool
		x, ok2 = mp[k]
		if !ok2 {
			return nil, false
		}
	}
	return x, true
}

func asMap(x any) (map[string]any, bool) {
	if m, ok := x.(map[string]any); ok {
		return m, true
	}
	if m, ok := x.(map[string]interface{}); ok {
		return m, true
	}
	return nil, false
}

func writeJSON(w io.Writer, n jnode) error {
	switch v := n.(type) {
	case *jobject:
		if _, err := io.WriteString(w, "{"); err != nil {
			return err
		}
		for i, f := range v.fields {
			if i > 0 {
				if _, err := io.WriteString(w, ","); err != nil {
					return err
				}
			}
			// key
			kb, _ := json.Marshal(f.key)
			if _, err := w.Write(kb); err != nil {
				return err
			}
			if _, err := io.WriteString(w, ":"); err != nil {
				return err
			}
			if err := writeJSON(w, f.val); err != nil {
				return err
			}
		}
		_, err := io.WriteString(w, "}")
		return err
	case *jarray:
		if _, err := io.WriteString(w, "["); err != nil {
			return err
		}
		for i, e := range v.elems {
			if i > 0 {
				if _, err := io.WriteString(w, ","); err != nil {
					return err
				}
			}
			if err := writeJSON(w, e); err != nil {
				return err
			}
		}
		_, err := io.WriteString(w, "]")
		return err
	default:
		if str, ok := v.(string); ok {
			quoted := engine.Quote(str)
			_, err := w.Write([]byte(quoted))
			return err
		}
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		_, err = w.Write(b)
		return err
	}
}
