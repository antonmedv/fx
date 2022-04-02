package main

type dict struct {
	keys   []string
	values map[string]interface{}
}

func newDict() *dict {
	return &dict{
		keys:   make([]string, 0),
		values: make(map[string]interface{}),
	}
}

func (d *dict) get(key string) (interface{}, bool) {
	val, exists := d.values[key]
	return val, exists
}

func (d *dict) set(key string, value interface{}) {
	_, exists := d.values[key]
	if !exists {
		d.keys = append(d.keys, key)
	}
	d.values[key] = value
}
