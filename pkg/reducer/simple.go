package reducer

import (
	"strconv"
	"unicode"

	. "github.com/antonmedv/fx/pkg/dict"
	. "github.com/antonmedv/fx/pkg/json"
)

type state int

const (
	start state = iota
	unknown
	propOrIndex
	prop
	index
	indexEnd
	number
	doubleQuote
	doubleQuoteEscape
	singleQuote
	singleQuoteEscape
)

func SplitSimplePath(args []string) ([]interface{}, bool) {
	path := make([]interface{}, 0)
	for _, arg := range args {
		s := ""
		state := start
		for _, ch := range arg {
			switch state {

			case start:
				switch {
				case ch == 'x':
					state = unknown
				case ch == '.':
					state = propOrIndex
				default:
					return path, false
				}

			case unknown:
				switch {
				case ch == '.':
					state = prop
					s = ""
				case ch == '[':
					state = index
					s = ""
				default:
					return path, false
				}

			case propOrIndex:
				switch {
				case isProp(ch):
					state = prop
					s = string(ch)
				case ch == '[':
					state = index
				default:
					return path, false
				}

			case prop:
				switch {
				case isProp(ch):
					s += string(ch)
				case ch == '.':
					state = prop
					path = append(path, s)
					s = ""
				case ch == '[':
					state = index
					path = append(path, s)
					s = ""
				default:
					return path, false
				}

			case index:
				switch {
				case unicode.IsDigit(ch):
					state = number
					s = string(ch)
				case ch == '"':
					state = doubleQuote
					s = ""
				case ch == '\'':
					state = singleQuote
					s = ""
				default:
					return path, false
				}

			case indexEnd:
				switch {
				case ch == ']':
					state = unknown
				default:
					return path, false
				}

			case number:
				switch {
				case unicode.IsDigit(ch):
					s += string(ch)
				case ch == ']':
					state = unknown
					n, err := strconv.Atoi(s)
					if err != nil {
						return path, false
					}
					path = append(path, n)
					s = ""
				default:
					return path, false
				}

			case doubleQuote:
				switch ch {
				case '"':
					state = indexEnd
					path = append(path, s)
					s = ""
				case '\\':
					state = doubleQuoteEscape
				default:
					s += string(ch)
				}

			case doubleQuoteEscape:
				switch ch {
				case '"':
					state = doubleQuote
					s += string(ch)
				default:
					return path, false
				}

			case singleQuote:
				switch ch {
				case '\'':
					state = indexEnd
					path = append(path, s)
					s = ""
				case '\\':
					state = singleQuoteEscape
					s += string(ch)
				default:
					s += string(ch)
				}

			case singleQuoteEscape:
				switch ch {
				case '\'':
					state = singleQuote
					s += string(ch)
				default:
					return path, false
				}
			}
		}
		if len(s) > 0 {
			if state == prop {
				path = append(path, s)
			} else {
				return path, false
			}

		}
	}
	return path, true
}

func isProp(ch rune) bool {
	return unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_' || ch == '$'
}

func GetBySimplePath(object interface{}, path []interface{}) interface{} {
	for _, get := range path {
		switch get := get.(type) {
		case string:
			switch o := object.(type) {
			case *Dict:
				object = o.Values[get]
			case string:
				if get == "length" {
					object = Number(strconv.Itoa(len([]rune(o))))
				} else {
					object = nil
				}
			case Array:
				if get == "length" {
					object = Number(strconv.Itoa(len(o)))
				} else {
					object = nil
				}
			default:
				object = nil
			}
		case int:
			switch o := object.(type) {
			case Array:
				object = o[get]
			default:
				object = nil
			}
		}
	}
	return object
}
