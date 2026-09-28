package complete

import (
	"bytes"
	"io"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/require"

	"github.com/antonmedv/fx/internal/jsonx"
)

const walkData = `
{
  "a": {"b": 1, "c": [{"d": 1, "e": {"f": 1}}, {"d": 2, "g": 3}]},
  "list": [{"x": 1}, {"y": 2, "x": 3}],
  "k-ey": {"z": 1},
  "0": {"zero": 1},
  "nested": [[{"p": 1}], [{"q": 2}]],
  "dup": {"u": 1},
  "dup": {"v": 2},
  "esc\"aped": {"in\u0041": 1},
  "$dollar": [{"m": 1}],
  "str": "text",
  "num": 1
}
{
  "a": {"b": 2, "h": 4, "c": []},
  "list": [{"w": 1}],
  "k-ey": {"z2": 1},
  "0": {"zero": 1},
  "nested": [],
  "dup": {"v": 3},
  "esc\"aped": {"B": 1},
  "$dollar": [],
  "str": "more",
  "num": 2
}
`

func parseDocs(t testing.TB, data string) []*jsonx.Node {
	p := jsonx.NewJsonParser(bytes.NewReader([]byte(data)), false)
	var docs []*jsonx.Node
	for {
		node, err := p.Parse()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		docs = append(docs, node)
	}
	return docs
}

// sliceDocs links docs like the viewer does, with Next reading the slice so
// that tests can append documents while streaming.
func sliceDocs(docs *[]*jsonx.Node) Docs {
	return Docs{
		First: (*docs)[0],
		Next: func(doc *jsonx.Node) *jsonx.Node {
			i := slices.Index(*docs, doc)
			if i+1 < len(*docs) {
				return (*docs)[i+1]
			}
			return nil
		},
	}
}

func walkKeys(t *testing.T, docs []*jsonx.Node, args []string, base string) (*baseKeys, path) {
	b, ok := parsePath(base, true)
	require.True(t, ok, "base %q is not a path", base)
	var paths []path
	for _, arg := range args {
		p, ok := parsePath(arg, false)
		require.True(t, ok, "arg %q is not a path", arg)
		paths = append(paths, p)
	}
	k := &baseKeys{}
	k.walk(sliceDocs(&docs), paths, b)
	return k, b
}

func sorted(s []string) []string {
	s = slices.Clone(s)
	slices.Sort(s)
	return s
}

// TestWalkMatchesEngine checks that walking without JS finds the keys the
// engine would, over all documents, and offers `[]` only where fx accepts it.
func TestWalkMatchesEngine(t *testing.T) {
	docs := parseDocs(t, walkData)
	tests := []struct {
		args []string
		base string
	}{
		{nil, ""},
		{nil, ".a"},
		{nil, ".a.c"},
		{nil, ".a.c[]"},
		{nil, ".a.c[].e"},
		{nil, ".a.c[0]"},
		{nil, ".a.c[1]"},
		{nil, `.a["c"][0]`},
		{nil, `.a['c'][0]`},
		{nil, ".list"},
		{nil, ".list[]"},
		{nil, `.["k-ey"]`},
		{nil, `.["0"]`},
		{nil, ".[0]"},
		{nil, `.["esc\"aped"]`},
		{nil, ".nested"},
		{nil, ".nested[]"},
		{nil, ".nested[][]"},
		{nil, ".nested[0]"},
		{nil, ".dup"},
		{nil, ".$dollar"},
		{nil, ".str"},
		{nil, ".missing"},
		{nil, "@.a"},
		{nil, "x.a"},
		{nil, "this.a"},
		{nil, "x"},
		{[]string{".a"}, ""},
		{[]string{".a"}, ".c[]"},
		{[]string{".a", ".c"}, "@"},
		{[]string{".list[]"}, "@"},
		{[]string{".list"}, "@"},
		{[]string{".list"}, ".[]"},
		{[]string{".list"}, "."},
		{[]string{"@.a"}, ""},
		{[]string{".nested"}, "@@"},
		{[]string{".nested[]"}, "@"},
		{[]string{".nested[][]"}, ".[]"},
		{[]string{".a.c[].e", ".[0]"}, ""},
		{[]string{".a.c[].d"}, ".[]"},
		{[]string{"x.a", "this"}, ".c[0]"},
		{[]string{".", "x"}, ".a"},
	}
	for _, tt := range tests {
		t.Run(tt.base+" after "+slices.Concat(tt.args, []string{""})[0], func(t *testing.T) {
			k, b := walkKeys(t, docs, tt.args, tt.base)

			base := tt.base
			if base == "." {
				base = ""
			}
			want := engineKeys(docs, append(slices.Clone(tt.args), base+".__keys()"), nil).Keys
			require.Equal(t, sorted(want), sorted(k.obj.keys), "object keys")

			each := base + "[]"
			if base == "" || base == "@" || base == "@@" {
				each = base + ".[]"
			}
			wantElem := engineKeys(docs, append(slices.Clone(tt.args), each+".__keys()"), nil).Keys
			if b.each {
				require.Equal(t, sorted(wantElem), sorted(k.elem.keys), "element keys")
			} else {
				require.Empty(t, wantElem, "fx accepts %s, but it is not offered", each)
			}
		})
	}
}

func TestNotAPath(t *testing.T) {
	for _, arg := range []string{
		"@", "len", ".a.map(x => x)", ".a?.b", `.a["b" + 1]`, ".a[-1]", ".a[01]", ".a.1",
		`.["x"][]`, ".a[0][]", ".a.[0]", "?.a", `.a["\x41"]`,
	} {
		_, ok := parsePath(arg, false)
		require.False(t, ok, arg)
	}
}

func values(replies []Reply) []string {
	var out []string
	for _, r := range replies {
		out = append(out, r.Value)
	}
	return out
}

func TestReplies(t *testing.T) {
	docs := parseDocs(t, walkData)
	tests := []struct {
		query string
		want  []string
	}{
		{".a.", []string{".a.b", ".a.c", ".a.h"}},
		{".a.c.", []string{".a.c[].d", ".a.c[].e", ".a.c[].g"}},
		{".a.c.g", []string{".a.c[].g"}},
		{".a.c[", []string{".a.c[]"}},
		{".a.c[].", []string{".a.c[].d", ".a.c[].e", ".a.c[].g"}},
		{".a.c[].e.", []string{".a.c[].e.f"}},
		{".a.c[0].", []string{".a.c[0].d", ".a.c[0].e"}},
		{".k", []string{`.["k-ey"]`}},
		{".[", []string{`.["a"]`, `.["list"]`, `.["k-ey"]`, `.["0"]`, `.["nested"]`, `.["dup"]`, `.["esc\"aped"]`, `.["$dollar"]`, `.["str"]`, `.["num"]`}},
		{`.["k`, []string{`.["k-ey"]`}},
		{`.['k`, []string{`.['k-ey']`}},
		{`.["esc\"`, []string{`.["esc\"aped"]`}},
		{`.["esc\"aped"].`, []string{`.["esc\"aped"].inA`, `.["esc\"aped"].B`}},
		{".list.", []string{".list[].x", ".list[].y", ".list[].w"}},
		{".nested.le", []string{".nested.length"}}, // methods, no keys
		{".nested[].", []string{".nested[][].p", ".nested[][].q"}},
		{".nested[][].", []string{".nested[][].p", ".nested[][].q"}},
		{`.["k-ey"].`, []string{`.["k-ey"].z`, `.["k-ey"].z2`}},
		{`.$dollar.le`, []string{".$dollar.length"}}, // fx does not accept .$dollar[]
		{".dup.", []string{".dup.v"}},
		{".str.len", []string{".str.length"}},
		{".list @.", []string{"@.x", "@.y", "@.w"}},
		{".list @", nil},
		{".list .", []string{".[].x", ".[].y", ".[].w"}},
		{".list .[", []string{".[]"}},
		{".a .c[].", []string{".c[].d", ".c[].e", ".c[].g"}},
		{".a.c.map(x => x.", []string{".a.c.map(x => x.d", ".a.c.map(x => x.e", ".a.c.map(x => x.g"}},
		{".list.filter(x => x.y) @.", []string{"@.y", "@.x"}},
		{"le", []string{"len"}},
		{".a.c.map(le", []string{".a.c.map(len"}},
		{".a.c.map(x => x.d == 'a.", nil},
		{"", []string{".a", `.["k-ey"]`, `.["0"]`, ".nested", ".dup", `.["esc\"aped"]`, ".$dollar", ".str", ".num", ".list"}},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			r, start := ParseQuery(tt.query)
			require.Equal(t, len(tt.query)-len(r.Word), start)
			got := Replies(r, sliceDocs(&docs))
			if tt.query == "" {
				require.ElementsMatch(t, tt.want, values(got))
				return
			}
			require.Equal(t, tt.want, values(got))
		})
	}
}

func TestCacheStreaming(t *testing.T) {
	all := parseDocs(t, `{"a": {"x": 1}} {"b": 1} {"a": {"y": 2}}`)
	docs := all[:1]
	var cache Cache
	r, _ := ParseQuery(".a.")
	got, needEngine := cache.Complete(r, sliceDocs(&docs))
	require.False(t, needEngine)
	require.Equal(t, []string{".a.x"}, values(got))

	docs = all
	got, _ = cache.Complete(r, sliceDocs(&docs))
	require.Equal(t, []string{".a.x", ".a.y"}, values(got))

	r, _ = ParseQuery(".a.y")
	got, _ = cache.Complete(r, sliceDocs(&docs))
	require.Equal(t, []string{".a.y"}, values(got))
}

func TestCacheEngine(t *testing.T) {
	docs := parseDocs(t, `{"a": [{"x": 1}]}`)
	var cache Cache
	r, _ := ParseQuery(".a.map(x => x.")
	got, needEngine := cache.Complete(r, sliceDocs(&docs))
	require.True(t, needEngine)
	require.Nil(t, got)

	cache.PutEngine(r, r.EngineKeys(docs[0], nil))
	r, _ = ParseQuery(".a.map(x => x.x")
	got, needEngine = cache.Complete(r, sliceDocs(&docs))
	require.False(t, needEngine)
	require.Equal(t, []string{".a.map(x => x.x"}, values(got))
}

func TestEngineCancel(t *testing.T) {
	docs := parseDocs(t, `{"a": 1}`)
	cancel := make(chan struct{})
	close(cancel)
	r, _ := ParseQuery("(() => { while (true) {} })() .")
	require.Zero(t, r.EngineKeys(docs[0], cancel))

	r, _ = ParseQuery(`save(x) .`)
	require.Zero(t, r.EngineKeys(docs[0], nil)) // save is disabled
}

func TestReplies_MethodsIfNoPropertyMatches(t *testing.T) {
	docs := parseDocs(t, `{"s": "text", "n": 1, "list": [1, 2], "items": [{"map": 1}], "o": {"toUpper": 1}, "b": true}`)
	tests := []struct {
		query string
		want  []string
	}{
		{".s.toUpperC", []string{".s.toUpperCase"}},
		{".s.len", []string{".s.length"}},
		{".n.toF", []string{".n.toFixed"}},
		{".list.fla", []string{".list.flat", ".list.flatMap"}},
		{".list.le", []string{".list.length"}},
		{".items.ma", []string{".items[].map"}}, // an element key wins
		{".items.fil", []string{".items.fill", ".items.filter"}},
		{".o.toUpper", []string{".o.toUpper"}},
		{".o.toSt", nil}, // no methods of objects
		{".b.", nil},
		{".list[", []string{".list[]"}}, // not after a bracket
		{"Object.ke", []string{"Object.keys"}},
		{"JSON.", []string{"JSON.parse", "JSON.isRawJSON", "JSON.rawJSON", "JSON.stringify"}},
		{"console.", []string{"console.log"}},
		{".s.split(' ').jo", []string{".s.split(' ').join"}},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			r, _ := ParseQuery(tt.query)
			got := values(Replies(r, sliceDocs(&docs)))
			if strings.HasPrefix(tt.query, "JSON") {
				require.ElementsMatch(t, tt.want, got)
				return
			}
			require.Equal(t, tt.want, got)
		})
	}
}

func TestReplies_MethodsListedWhenNothingTyped(t *testing.T) {
	docs := parseDocs(t, `{"s": "text"}`)
	r, _ := ParseQuery(".s.")
	got := values(Replies(r, sliceDocs(&docs)))
	require.Contains(t, got, ".s.toUpperCase")
	require.Contains(t, got, ".s.length")
	require.NotContains(t, got, ".s.constructor")
	require.True(t, slices.IsSorted(got))
}

func TestReplies_BuiltinsIfNoStdlibMatches(t *testing.T) {
	docs := parseDocs(t, `{}`)
	for query, want := range map[string][]string{
		"ma":  {"map"},
		"Ma":  {"Math", "Map"},
		"Obj": {"Object"},
		"con": {"console"},
		"YA":  {"YAML"},
		"MA":  {"MAML"},
		"sk":  {"skip"},
		"esc": nil, // not a constructor or namespace
	} {
		r, _ := ParseQuery(query)
		require.ElementsMatch(t, want, values(Replies(r, sliceDocs(&docs))), query)
	}
}

// TestAccessorRoundTrip checks that the JS of every accessor evaluates back
// to its key, in both quote styles.
func TestAccessorRoundTrip(t *testing.T) {
	keys := []string{"hello\nworld", "tab\there", "cr\rbs\bff\f", "nul\x00del\x7f", "esc\x1b[0m",
		`back\slash`, `dq"`, `sq'`, `both'"\`, "日本 語", "line sep"}
	for _, key := range keys {
		for _, quote := range []byte{'"', '\''} {
			a := accessor(key, false, quote)
			for _, c := range a {
				require.False(t, c < 0x20 || c == 0x7f, "raw control character in %q", a)
			}
			v, err := goja.New().RunString("({[" + strconv.Quote(key) + "]: 1})" + a)
			require.NoError(t, err, a)
			require.Equal(t, int64(1), v.Export(), "%s does not access %q", a, key)
		}
	}
}

func TestReplies_SingleQuotedControlCharacters(t *testing.T) {
	docs := parseDocs(t, `{"hello\nworld": 1}`)
	r, _ := ParseQuery(`.['h`)
	require.Equal(t, []string{`.['hello\nworld']`}, values(Replies(r, sliceDocs(&docs))))
}
