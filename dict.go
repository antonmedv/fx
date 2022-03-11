package main

type dict struct {
	keys    []string
	indexes map[string]int
	values  map[string]interface{}
}

func newDict() *dict {
	d := dict{}
	d.keys = []string{}
	d.indexes = map[string]int{}
	d.values = map[string]interface{}{}
	return &d
}

func (d *dict) get(key string) (interface{}, bool) {
	val, exists := d.values[key]
	return val, exists
}

func (d *dict) index(key string) (int, bool) {
	index, exists := d.indexes[key]
	return index, exists
}

func (d *dict) set(key string, value interface{}) {
	_, exists := d.values[key]
	if !exists {
		d.indexes[key] = len(d.keys)
		d.keys = append(d.keys, key)
	}
	d.values[key] = value
}
