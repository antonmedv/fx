package main

type dict struct {
	keys    []string
	indexes map[string]int
	values  map[string]interface{}
}

func newDict() *dict {
	return newDictOfCapacity(0)
}

func newDictOfCapacity(capacity int) *dict {
	return &dict{
		keys:    make([]string, 0, capacity),
		indexes: make(map[string]int, capacity),
		values:  make(map[string]interface{}, capacity),
	}
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
