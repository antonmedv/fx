package dict

type Dict struct {
	Keys   []string
	Values map[string]interface{}
}

func NewDict() *Dict {
	return &Dict{
		Keys:   make([]string, 0),
		Values: make(map[string]interface{}),
	}
}

func (d *Dict) Get(key string) (interface{}, bool) {
	val, exists := d.Values[key]
	return val, exists
}

func (d *Dict) Set(key string, value interface{}) {
	_, exists := d.Values[key]
	if !exists {
		d.Keys = append(d.Keys, key)
	}
	d.Values[key] = value
}
