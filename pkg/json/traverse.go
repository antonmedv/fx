package json

import (
	"fmt"
	. "github.com/antonmedv/fx/pkg/dict"
)

type Iterator struct {
	Object       interface{}
	Path, Parent string
}

func Dfs(object interface{}, f func(it Iterator)) {
	sub(Iterator{Object: object}, f)
}

func sub(it Iterator, f func(it Iterator)) {
	f(it)
	switch it.Object.(type) {
	case *Dict:
		keys := it.Object.(*Dict).Keys
		for _, k := range keys {
			subpath := it.Path + "." + k
			value, _ := it.Object.(*Dict).Get(k)
			sub(Iterator{
				Object: value,
				Path:   subpath,
				Parent: it.Path,
			}, f)
		}

	case Array:
		slice := it.Object.(Array)
		for i, value := range slice {
			subpath := accessor(it.Path, i)
			sub(Iterator{
				Object: value,
				Path:   subpath,
				Parent: it.Path,
			}, f)
		}
	}
}

func accessor(path string, to interface{}) string {
	return fmt.Sprintf("%v[%v]", path, to)
}
