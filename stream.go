package main

import (
	"encoding/json"
	"fmt"
	"io"

	. "github.com/antonmedv/fx/pkg/json"
	. "github.com/antonmedv/fx/pkg/reducer"
	. "github.com/antonmedv/fx/pkg/theme"
)

func stream(dec *json.Decoder, jsonObject interface{}, lang string, args []string, theme Theme) int {
	var err error
	for {
		if jsonObject != nil {
			Reduce(jsonObject, lang, args, theme)
		}
		jsonObject, err = Parse(dec)
		if err == io.EOF {
			return 0
		}
		if err != nil {
			fmt.Println("JSON Parse Error:", err.Error())
			return 1
		}
	}
}
