package complete

import (
	"errors"
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
		return r.replies(k, path{}), false
	}
	k.walk(docs, args, base)
	return r.replies(k, base), false
}

// PutEngine stores the keys EngineKeys returned for r.
func (c *Cache) PutEngine(r *Request, names []string) {
	k := c.get(r.cacheKey())
	*k = baseKeys{engine: true}
	k.obj.addAll(names)
}

func (r *Request) replies(k *baseKeys, base path) []Reply {
	var replies []Reply
	add := func(value string) {
		replies = append(replies, Reply{Display: value[len(r.base):], Value: value, Type: "key"})
	}
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
func (r *Request) EngineKeys(doc *jsonx.Node, cancel <-chan struct{}) []string {
	last := balanceBrackets(strings.TrimSuffix(r.base, ".") + ".__keys()")
	return engineKeys([]*jsonx.Node{doc}, append(r.Args[:len(r.Args):len(r.Args)], last), cancel)
}

func engineKeys(docs []*jsonx.Node, args []string, cancel <-chan struct{}) []string {
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
		return nil
	}
	main, ok := goja.AssertFunction(vm.Get("__main__"))
	if !ok {
		return nil
	}
	for _, doc := range docs {
		if _, err := callMain(main, doc.ToValue(vm)); err != nil {
			if _, ok := err.(*goja.InterruptedError); ok {
				return nil
			}
			// Keys collected before the error are still completions.
		}
	}
	value, err := vm.RunString("Array.from(__keys)")
	if err != nil {
		return nil
	}
	var keys []string
	if array, ok := value.Export().([]any); ok {
		for _, key := range array {
			if s, ok := key.(string); ok {
				keys = append(keys, s)
			}
		}
	}
	return keys
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

func (r *Request) globals() []Reply {
	var replies []Reply
	for _, name := range globals() {
		if strings.HasPrefix(name, r.partial) {
			replies = append(replies, Reply{Display: name, Value: r.base + name, Type: "global"})
		}
	}
	return replies
}

var globals = sync.OnceValue(func() []string {
	var code strings.Builder
	code.WriteString(prelude)
	code.WriteString(engine.Stdlib)
	code.WriteString("\n__autocomplete()\n")

	value, err := goja.New().RunString(code.String())
	if err != nil {
		return nil
	}
	var names []string
	if array, ok := value.Export().([]any); ok {
		for _, key := range array {
			names = append(names, key.(string))
		}
	}
	return names
})
