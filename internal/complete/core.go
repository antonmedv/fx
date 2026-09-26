package complete

import (
	"errors"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"

	"github.com/antonmedv/fx/internal/engine"
	"github.com/antonmedv/fx/internal/jsonx"
)

// Replies returns the completions of r, walking docs; the engine runs on
// the first document if the arguments are not plain paths.
func Replies(r *Request, docs Docs) []Reply {
	var cache Cache
	replies, needEngine := cache.Complete(r, docs)
	if needEngine && docs.First != nil {
		cache.PutEngine(r, r.EngineKeys(docs.First, nil))
		replies, _ = cache.Complete(r, docs)
	}
	return replies
}

// Complete returns the completions of r. Plain paths are walked in docs,
// without JS. needEngine is true if the keys can only come from EngineKeys,
// given to PutEngine.
func (c *Cache) Complete(r *Request, docs Docs) (replies []Reply, needEngine bool) {
	switch r.mode {
	case modeNone:
		return nil, false
	case modeGlobal:
		return r.globals(), false
	}
	base, ok := parsePath(r.base, true)
	args := make([]path, 0, len(r.Args))
	for _, arg := range r.Args {
		p, isPath := parsePath(arg, false)
		ok = ok && isPath
		args = append(args, p)
	}
	k := c.get(r.cacheKey())
	if !ok {
		if !k.engine {
			return nil, true
		}
		return r.replies(k, path{empty: base.empty}), false
	}
	k.walk(docs, args, base)
	return r.replies(k, base), false
}

// Names are what EngineKeys finds after a base: own properties, and the
// methods offered when no own property matches.
type Names struct {
	Keys, Methods []string
}

// PutEngine stores the names EngineKeys returned for r.
func (c *Cache) PutEngine(r *Request, names Names) {
	k := c.get(r.cacheKey())
	*k = baseKeys{engine: true}
	k.obj.addAll(names.Keys)
	k.methods = names.Methods
}

func (r *Request) replies(k *baseKeys, base path) (replies []Reply) {
	add := func(value string) {
		replies = append(replies, Reply{Display: value[len(r.base):], Value: value, Type: "key"})
	}
	defer func() {
		// Methods are many; list them only if no property matches.
		if r.mode != modeKey || len(replies) > 0 {
			return
		}
		for _, name := range k.methodNames() {
			if strings.HasPrefix(name, r.partial) {
				value := r.base + accessor(name, base.empty, 0)
				replies = append(replies, Reply{Display: value[len(r.base):], Value: value, Type: "method"})
			}
		}
	}()
	switch r.mode {
	case modeKey:
		for _, key := range k.obj.keys {
			if strings.HasPrefix(key, r.partial) {
				add(r.base + accessor(key, base.empty, 0))
			}
		}
		if base.each {
			each := r.base + "[]"
			if base.empty {
				each = r.base + ".[]"
			}
			for _, key := range k.elem.keys {
				if strings.HasPrefix(key, r.partial) {
					add(each + accessor(key, false, 0))
				}
			}
		}
	case modeBracket:
		dot := base.empty && !strings.HasSuffix(r.base, ".")
		if r.quote == 0 && base.each && k.arrays {
			if dot {
				add(r.base + ".[]")
			} else {
				add(r.base + "[]")
			}
		}
		quote := r.quote
		if quote == 0 {
			quote = '"'
		}
		for _, key := range k.obj.keys {
			if strings.HasPrefix(key, r.partial) {
				add(r.base + accessor(key, dot, quote))
			}
		}
	}
	return replies
}

// EngineTimeout bounds EngineKeys, as the query may never end.
const EngineTimeout = 2 * time.Second

// EngineKeys evaluates the arguments and the base of r on doc with the
// preview engine (save and exit disabled), returning the keys of the
// objects the completed accessor applies to. It stops on cancel or after
// EngineTimeout.
func (r *Request) EngineKeys(doc *jsonx.Node, cancel <-chan struct{}) Names {
	last := balanceBrackets(strings.TrimSuffix(r.base, ".") + ".__keys()")
	return engineKeys([]*jsonx.Node{doc}, append(r.Args[:len(r.Args):len(r.Args)], last), cancel)
}

func engineKeys(docs []*jsonx.Node, args []string, cancel <-chan struct{}) Names {
	var code strings.Builder
	code.WriteString(prelude)
	code.WriteString(engine.Stdlib)
	code.WriteString(engine.JS(args))

	vm := engine.NewVM(func(string) {}, true)
	finished := make(chan struct{})
	defer close(finished)
	go func() {
		timeout := time.NewTimer(EngineTimeout)
		defer timeout.Stop()
		select {
		case <-cancel:
		case <-timeout.C:
		case <-finished:
			return
		}
		vm.Interrupt("cancelled")
	}()

	if _, err := vm.RunString(code.String()); err != nil {
		return Names{}
	}
	main, ok := goja.AssertFunction(vm.Get("__main__"))
	if !ok {
		return Names{}
	}
	for _, doc := range docs {
		if _, err := callMain(main, doc.ToValue(vm)); err != nil {
			if _, ok := err.(*goja.InterruptedError); ok {
				return Names{}
			}
			// Keys collected before the error are still completions.
		}
	}
	keys, _ := vm.RunString("Array.from(__keys)")
	methods, _ := vm.RunString("Array.from(__methods)")
	return Names{Keys: exportStrings(keys), Methods: exportStrings(methods)}
}

// exportStrings returns the strings of a JS array.
func exportStrings(value goja.Value) []string {
	if value == nil {
		return nil
	}
	var out []string
	if array, ok := value.Export().([]any); ok {
		for _, v := range array {
			if s, ok := v.(string); ok {
				out = append(out, s)
			}
		}
	}
	return out
}

// callMain runs main, recovering from panics of Go functions called by JS.
func callMain(main goja.Callable, input goja.Value) (_ goja.Value, err error) {
	defer func() {
		if recover() != nil {
			err = errPanic
		}
	}()
	return main(goja.Undefined(), input)
}

var errPanic = errors.New("panic")

// globals completes a global: an fx function or value, or, if none matches,
// a JS built-in like Math.
func (r *Request) globals() []Reply {
	names := jsNames()
	replies := r.globalReplies(names.stdlib)
	if len(replies) == 0 {
		replies = r.globalReplies(names.builtins)
	}
	return replies
}

func (r *Request) globalReplies(names []string) []Reply {
	var replies []Reply
	for _, name := range names {
		if strings.HasPrefix(name, r.partial) {
			replies = append(replies, Reply{Display: name, Value: r.base + name, Type: "global"})
		}
	}
	return replies
}

// names are the JS names completion offers, read from the engine once.
type names struct {
	stdlib   []string // fx functions and values, in stdlib order
	builtins []string // JS built-ins like Math and JSON
	string   []string // methods of strings, sorted
	number   []string
	array    []string
}

// reConst finds stdlib values like YAML, which JS does not list in globalThis.
var reConst = regexp.MustCompile(`(?m)^const ([A-Za-z_$][\w$]*)`)

var jsNames = sync.OnceValue(func() names {
	var code strings.Builder
	code.WriteString(prelude)
	code.WriteString(engine.Stdlib)
	code.WriteString("\n;__autocomplete()\n")

	var n names
	value, err := goja.New().RunString(code.String())
	if err != nil {
		return n
	}
	obj, ok := value.Export().(map[string]any)
	if !ok {
		return n
	}
	list := func(key string) []string {
		var out []string
		if array, ok := obj[key].([]any); ok {
			for _, v := range array {
				if s, ok := v.(string); ok {
					out = append(out, s)
				}
			}
		}
		return out
	}
	n.stdlib = list("stdlib")
	for _, m := range reConst.FindAllStringSubmatch(engine.Stdlib, -1) {
		if !slices.Contains(n.stdlib, m[1]) {
			n.stdlib = append(n.stdlib, m[1])
		}
	}
	// A fresh runtime: goja lists built-ins only until a script declares
	// globals.
	builtins, err := goja.New().RunString("Object.getOwnPropertyNames(globalThis)")
	if err == nil {
		for _, name := range exportStrings(builtins) {
			// Constructors and namespaces, like Object and Math; not eval,
			// escape and such.
			if name[0] >= 'A' && name[0] <= 'Z' && !slices.Contains(n.stdlib, name) {
				n.builtins = append(n.builtins, name)
			}
		}
	}
	n.string, n.number, n.array = sortStrings(list("string")), sortStrings(list("number")), sortStrings(list("array"))
	return n
})

func sortStrings(s []string) []string {
	slices.Sort(s)
	return s
}
