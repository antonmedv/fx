package engine

import (
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"time"

	"github.com/dop251/goja"
)

// Stringify serializes value as indented JSON. A circular structure throws
// a TypeError, as JSON.stringify does, instead of recursing forever.
func Stringify(value goja.Value, vm *goja.Runtime, depth int) string {
	s := stringifier{vm: vm, indent: true}
	return s.run(value, depth)
}

// stringifyCompact serializes value as JSON without whitespace, which keeps
// deeply nested output linear in size. It stops with errAborted once abort
// is closed, as the work is Go code that interrupting the VM can't stop.
func stringifyCompact(value goja.Value, vm *goja.Runtime, abort <-chan struct{}) string {
	s := stringifier{vm: vm, abort: abort}
	return s.run(value, 0)
}

// errAborted is the panic of a stringifier stopped by its abort channel.
var errAborted = fmt.Errorf("aborted")

type stringifier struct {
	vm     *goja.Runtime
	indent bool
	abort  <-chan struct{}
	out    strings.Builder
	seen   map[*goja.Object]struct{} // The objects and arrays being serialized.
	steps  int
}

func (s *stringifier) run(value goja.Value, depth int) string {
	s.seen = make(map[*goja.Object]struct{})
	s.write(value, depth)
	return s.out.String()
}

// enter marks obj as being serialized, throwing if it already is.
func (s *stringifier) enter(obj *goja.Object) {
	if _, ok := s.seen[obj]; ok {
		panic(s.vm.NewTypeError("Converting circular structure to JSON"))
	}
	s.seen[obj] = struct{}{}
}

func (s *stringifier) leave(obj *goja.Object) {
	delete(s.seen, obj)
}

// check panics with errAborted once abort is closed. It looks every so many
// values, a select per value would slow serializing down.
func (s *stringifier) check() {
	s.steps++
	if s.abort == nil || s.steps%1024 != 0 {
		return
	}
	select {
	case <-s.abort:
		panic(errAborted)
	default:
	}
}

// newline starts a line indented to depth, in indented mode.
func (s *stringifier) newline(depth int) {
	if !s.indent {
		return
	}
	s.out.WriteByte('\n')
	for range depth {
		s.out.WriteString("  ")
	}
}

func (s *stringifier) write(value goja.Value, depth int) {
	s.check()
	vm := s.vm
	rtype := value.ExportType()
	if rtype == nil {
		// Convert both null and undefined to null (save as JSON.stringify)
		s.out.WriteString("null")
		return
	}

	switch rtype {
	case bigIntType:
		s.out.WriteString(value.Export().(*big.Int).String())
		return
	case timeTimeType:
		s.out.WriteString(Quote(value.Export().(time.Time).String()))
		return
	}

	switch rtype.Kind() {
	case reflect.Bool:
		if value.ToBoolean() {
			s.out.WriteString("true")
		} else {
			s.out.WriteString("false")
		}

	case reflect.Int64, reflect.Float64:
		// NaN and ±Infinity print as such: fx shows them as values.
		s.out.WriteString(value.String())

	case reflect.String:
		s.out.WriteString(Quote(value.String()))

	case reflect.Map:
		obj := value.ToObject(vm)
		keys := obj.Keys()
		if len(keys) == 0 {
			s.out.WriteString("{}")
			return
		}
		s.enter(obj)
		s.out.WriteByte('{')
		for i, key := range keys {
			if i > 0 {
				s.out.WriteByte(',')
			}
			s.newline(depth + 1)
			s.out.WriteString(Quote(key))
			s.out.WriteByte(':')
			if s.indent {
				s.out.WriteByte(' ')
			}
			s.write(obj.Get(key), depth+1)
		}
		s.newline(depth)
		s.out.WriteByte('}')
		s.leave(obj)

	case reflect.Slice:
		arr := value.ToObject(vm)
		keys := arr.Keys()
		if len(keys) == 0 {
			s.out.WriteString("[]")
			return
		}
		s.enter(arr)
		s.out.WriteByte('[')
		for i, key := range keys {
			if i > 0 {
				s.out.WriteByte(',')
			}
			s.newline(depth + 1)
			s.write(arr.Get(key), depth+1)
		}
		s.newline(depth)
		s.out.WriteByte(']')
		s.leave(arr)

	default:
		panic(fmt.Sprintf("Unsupported value type: %v", rtype.Kind()))
	}
}
