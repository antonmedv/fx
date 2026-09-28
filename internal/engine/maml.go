package engine

import (
	"fmt"
	"math"
	"math/big"
	"reflect"
	"time"

	"github.com/dop251/goja"
	"github.com/maml-dev/go-maml"
)

// mamlParse converts MAML source to a JS value. Object keys keep their
// order. Integers come back as numbers, or as BigInts when a number would
// lose precision.
func mamlParse(vm *goja.Runtime, src string) (goja.Value, error) {
	v, err := maml.Parse(src)
	if err != nil {
		return nil, err
	}
	return mamlToJS(vm, v), nil
}

// maxSafeInteger is Number.MAX_SAFE_INTEGER.
const maxSafeInteger = 1<<53 - 1

func mamlToJS(vm *goja.Runtime, v maml.Value) goja.Value {
	switch {
	case v.IsNull():
		return goja.Null()
	case v.IsObject():
		obj := vm.NewObject()
		for _, kv := range v.AsObject().Entries() {
			_ = obj.Set(kv.Key, mamlToJS(vm, kv.Value))
		}
		return obj
	case v.IsArray():
		arr := v.AsArray()
		items := make([]any, len(arr))
		for i, item := range arr {
			items[i] = mamlToJS(vm, item)
		}
		return vm.NewArray(items...)
	case v.IsInt():
		n := v.AsInt()
		if n > maxSafeInteger || n < -maxSafeInteger {
			return vm.ToValue(big.NewInt(n))
		}
		return vm.ToValue(n)
	default:
		return vm.ToValue(v.Raw())
	}
}

// mamlStringify converts a JS value to MAML. NaN and Infinity become null
// as in JSON.stringify; undefined becomes null and dates become strings
// as in Stringify. A circular structure is an error.
func mamlStringify(vm *goja.Runtime, value goja.Value) (string, error) {
	e := mamlEncoder{vm: vm, parents: map[*goja.Object]bool{}}
	v, err := e.encode(value)
	if err != nil {
		return "", err
	}
	return maml.Stringify(v), nil
}

type mamlEncoder struct {
	vm *goja.Runtime
	// parents holds the objects and arrays being encoded, so a value that
	// contains itself is caught instead of recursing forever.
	parents map[*goja.Object]bool
}

// enter marks obj as being encoded. The returned func unmarks it.
func (e *mamlEncoder) enter(obj *goja.Object) (func(), error) {
	if e.parents[obj] {
		return nil, fmt.Errorf("MAML.stringify: converting circular structure")
	}
	e.parents[obj] = true
	return func() { delete(e.parents, obj) }, nil
}

func (e *mamlEncoder) encode(value goja.Value) (maml.Value, error) {
	vm := e.vm
	rtype := value.ExportType()
	if rtype == nil {
		return maml.NewNull(), nil
	}

	switch rtype {
	case bigIntType:
		bi := value.Export().(*big.Int)
		if !bi.IsInt64() {
			return maml.Value{}, fmt.Errorf("MAML.stringify: %s is outside the 64-bit integer range", bi)
		}
		return maml.NewInt(bi.Int64()), nil
	case timeTimeType:
		return maml.NewString(value.Export().(time.Time).String()), nil
	}

	switch rtype.Kind() {
	case reflect.Bool:
		return maml.NewBool(value.ToBoolean()), nil
	case reflect.Int64:
		return maml.NewInt(value.ToInteger()), nil
	case reflect.Float64:
		f := value.ToFloat()
		if math.IsInf(f, 0) || math.IsNaN(f) {
			return maml.NewNull(), nil
		}
		// Whole numbers print as integers, as JSON.stringify does.
		if f == math.Trunc(f) && math.Abs(f) < 1<<63 && !(f == 0 && math.Signbit(f)) {
			return maml.NewInt(int64(f)), nil
		}
		return maml.NewFloat(f), nil
	case reflect.String:
		return maml.NewString(value.String()), nil
	case reflect.Map:
		obj := value.ToObject(vm)
		leave, err := e.enter(obj)
		if err != nil {
			return maml.Value{}, err
		}
		defer leave()
		m := maml.NewOrderedMap()
		for _, key := range obj.Keys() {
			v, err := e.encode(obj.Get(key))
			if err != nil {
				return maml.Value{}, err
			}
			m.Set(key, v)
		}
		return maml.NewObject(m), nil
	case reflect.Slice:
		arr := value.ToObject(vm)
		leave, err := e.enter(arr)
		if err != nil {
			return maml.Value{}, err
		}
		defer leave()
		keys := arr.Keys()
		items := make([]maml.Value, len(keys))
		for i, key := range keys {
			v, err := e.encode(arr.Get(key))
			if err != nil {
				return maml.Value{}, err
			}
			items[i] = v
		}
		return maml.NewArray(items), nil
	}
	return maml.Value{}, fmt.Errorf("MAML.stringify: unsupported value type %v", rtype.Kind())
}
