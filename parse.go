package main

import (
	"encoding/json"
)

func parse(dec *json.Decoder) (interface{}, error) {
	token, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if delim, ok := token.(json.Delim); ok {
		switch delim {
		case '{':
			return decodeDict(dec)
		case '[':
			return decodeArray(dec)
		}
	}
	return token, nil
}

func decodeDict(dec *json.Decoder) (*dict, error) {
	d := newDict()
	for {
		token, err := dec.Token()
		if err != nil {
			return nil, err
		}
		if delim, ok := token.(json.Delim); ok && delim == '}' {
			return d, nil
		}
		key := token.(string)
		token, err = dec.Token()
		if err != nil {
			return nil, err
		}
		var value interface{} = token
		if delim, ok := token.(json.Delim); ok {
			switch delim {
			case '{':
				value, err = decodeDict(dec)
				if err != nil {
					return nil, err
				}
			case '[':
				value, err = decodeArray(dec)
				if err != nil {
					return nil, err
				}
			}
		}
		d.set(key, value)
	}
}

type array = []interface{}

func decodeArray(dec *json.Decoder) ([]interface{}, error) {
	slice := make(array, 0)
	for index := 0; ; index++ {
		token, err := dec.Token()
		if err != nil {
			return nil, err
		}
		if delim, ok := token.(json.Delim); ok {
			switch delim {
			case '{':
				value, err := decodeDict(dec)
				if err != nil {
					return nil, err
				}
				slice = append(slice, value)
			case '[':
				value, err := decodeArray(dec)
				if err != nil {
					return nil, err
				}
				slice = append(slice, value)
			case ']':
				return slice, nil
			}
			continue
		}
		slice = append(slice, token)
	}
}
