package main

type iterator struct {
	object       interface{}
	path, parent string
}

func dfs(object interface{}, f func(it iterator)) {
	sub(iterator{object: object}, f)
}

func sub(it iterator, f func(it iterator)) {
	f(it)
	switch it.object.(type) {
	case *dict:
		keys := it.object.(*dict).keys
		for _, k := range keys {
			subpath := it.path + "." + k
			value, _ := it.object.(*dict).get(k)
			sub(iterator{
				object: value,
				path:   subpath,
				parent: it.path,
			}, f)
		}

	case array:
		slice := it.object.(array)
		for i, value := range slice {
			subpath := accessor(it.path, i)
			sub(iterator{
				object: value,
				path:   subpath,
				parent: it.path,
			}, f)
		}
	}
}
