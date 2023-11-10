package main

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

type jsonParser struct {
	data           []byte
	end            int
	lastChar       byte
	lineNumber     uint
	sourceTail     *ring
	depth          uint8
	skipFirstIdent bool
}

func parse(data []byte) (head *node, err error) {
	p := &jsonParser{
		data:       data,
		lineNumber: 1,
		sourceTail: &ring{},
	}
	defer func() {
		if r := recover(); r != nil {
			err = p.errorSnippet(fmt.Sprintf("%v", r))
		}
	}()
	p.next()
	var next *node
	for p.lastChar != 0 {
		value := p.parseValue()
		if head == nil {
			head = value
			next = head
		} else {
			value.index = -1
			next.adjacent(value)
			next = value
		}
	}
	return
}

func (p *jsonParser) next() {
	if p.end < len(p.data) {
		p.lastChar = p.data[p.end]
		p.end++
	} else {
		p.lastChar = 0
	}
	if p.lastChar == '\n' {
		p.lineNumber++
	}
	p.sourceTail.writeByte(p.lastChar)
}

func (p *jsonParser) parseValue() *node {
	p.skipWhitespace()

	var l *node
	switch p.lastChar {
	case '"':
		l = p.parseString()
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '-':
		l = p.parseNumber()
	case '{':
		l = p.parseObject()
	case '[':
		l = p.parseArray()
	case 't':
		l = p.parseKeyword("true")
	case 'f':
		l = p.parseKeyword("false")
	case 'n':
		l = p.parseKeyword("null")
	default:
		panic(fmt.Sprintf("Unexpected character %q", p.lastChar))
	}

	p.skipWhitespace()
	return l
}

func (p *jsonParser) parseString() *node {
	str := &node{depth: p.depth}
	start := p.end - 1
	p.next()
	escaped := false
	for {
		if escaped {
			switch p.lastChar {
			case 'u':
				var unicode string
				for i := 0; i < 4; i++ {
					p.next()
					if !isHexDigit(p.lastChar) {
						panic(fmt.Sprintf("Invalid Unicode escape sequence '\\u%s%c'", unicode, p.lastChar))
					}
					unicode += string(p.lastChar)
				}
				_, err := strconv.ParseInt(unicode, 16, 32)
				if err != nil {
					panic(fmt.Sprintf("Invalid Unicode escape sequence '\\u%s'", unicode))
				}
			case '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
			default:
				panic(fmt.Sprintf("Invalid escape sequence '\\%c'", p.lastChar))
			}
			escaped = false
		} else if p.lastChar == '\\' {
			escaped = true
		} else if p.lastChar == '"' {
			break
		} else if p.lastChar == 0 {
			panic("Unexpected end of input in string")
		} else if p.lastChar < 0x1F {
			panic(fmt.Sprintf("Invalid character %q in string", p.lastChar))
		}
		p.next()
	}

	str.value = p.data[start:p.end]
	p.next()
	return str
}

func (p *jsonParser) parseNumber() *node {
	num := &node{depth: p.depth}
	start := p.end - 1

	// Handle negative numbers
	if p.lastChar == '-' {
		p.next()
		if !isDigit(p.lastChar) {
			panic(fmt.Sprintf("Invalid character %q in number", p.lastChar))
		}
	}

	// Leading zero
	if p.lastChar == '0' {
		p.next()
	} else {
		for isDigit(p.lastChar) {
			p.next()
		}
	}

	// Decimal portion
	if p.lastChar == '.' {
		p.next()
		if !isDigit(p.lastChar) {
			panic(fmt.Sprintf("Invalid character %q in number", p.lastChar))
		}
		for isDigit(p.lastChar) {
			p.next()
		}
	}

	// Exponent
	if p.lastChar == 'e' || p.lastChar == 'E' {
		p.next()
		if p.lastChar == '+' || p.lastChar == '-' {
			p.next()
		}
		if !isDigit(p.lastChar) {
			panic(fmt.Sprintf("Invalid character %q in number", p.lastChar))
		}
		for isDigit(p.lastChar) {
			p.next()
		}
	}

	num.value = p.data[start : p.end-1]
	return num
}

func (p *jsonParser) parseObject() *node {
	object := &node{depth: p.depth}
	object.value = []byte{'{'}

	p.next()
	p.skipWhitespace()

	// Empty object
	if p.lastChar == '}' {
		object.value = append(object.value, '}')
		p.next()
		return object
	}

	for {
		// Expecting a key which should be a string
		if p.lastChar != '"' {
			panic(fmt.Sprintf("Expected object key to be a string, got %q", p.lastChar))
		}

		p.depth++
		key := p.parseString()
		key.key, key.value = key.value, nil
		key.directParent = object

		p.skipWhitespace()

		// Expecting colon after key
		if p.lastChar != ':' {
			panic(fmt.Sprintf("Expected colon after object key, got %q", p.lastChar))
		}

		p.next()

		p.skipFirstIdent = true
		value := p.parseValue()
		p.depth--

		key.value = value.value
		key.next = value.next
		if key.next != nil {
			key.next.prev = key
		}
		key.end = value.end
		value.indirectParent = key
		object.append(key)

		p.skipWhitespace()

		if p.lastChar == ',' {
			object.end.comma = true
			p.next()
			p.skipWhitespace()
			if p.lastChar == '}' {
				object.end.comma = false
			} else {
				continue
			}
		}

		if p.lastChar == '}' {
			closeBracket := &node{depth: p.depth}
			closeBracket.value = []byte{'}'}
			closeBracket.directParent = object
			closeBracket.index = -1
			object.append(closeBracket)
			p.next()
			return object
		}

		panic(fmt.Sprintf("Unexpected character %q in object", p.lastChar))
	}
}

func (p *jsonParser) parseArray() *node {
	arr := &node{depth: p.depth}
	arr.value = []byte{'['}

	p.next()
	p.skipWhitespace()

	if p.lastChar == ']' {
		arr.value = append(arr.value, ']')
		p.next()
		return arr
	}

	for i := 0; ; i++ {
		p.depth++
		value := p.parseValue()
		value.directParent = arr
		value.index = i
		p.depth--

		arr.append(value)
		p.skipWhitespace()

		if p.lastChar == ',' {
			arr.end.comma = true
			p.next()
			p.skipWhitespace()
			if p.lastChar == ']' {
				arr.end.comma = false
			} else {
				continue
			}
		}

		if p.lastChar == ']' {
			closeBracket := &node{depth: p.depth}
			closeBracket.value = []byte{']'}
			closeBracket.directParent = arr
			closeBracket.index = -1
			arr.append(closeBracket)
			p.next()
			return arr
		}

		panic(fmt.Sprintf("Invalid character %q in array", p.lastChar))
	}
}

func (p *jsonParser) parseKeyword(name string) *node {
	for i := 1; i < len(name); i++ {
		p.next()
		if p.lastChar != name[i] {
			panic(fmt.Sprintf("Unexpected character %q in keyword", p.lastChar))
		}
	}
	p.next()

	nextCharIsSpecial := isWhitespace(p.lastChar) || p.lastChar == ',' || p.lastChar == '}' || p.lastChar == ']' || p.lastChar == 0
	if nextCharIsSpecial {
		keyword := &node{depth: p.depth}
		keyword.value = []byte(name)
		return keyword
	}

	panic(fmt.Sprintf("Unexpected character %q in keyword", p.lastChar))
}

func isWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}

func (p *jsonParser) skipWhitespace() {
	for {
		switch p.lastChar {
		case ' ', '\t', '\n', '\r':
			p.next()
		case '/':
			p.skipComment()
		default:
			return
		}
	}
}

func (p *jsonParser) skipComment() {
	p.next()
	switch p.lastChar {
	case '/':
		for p.lastChar != '\n' && p.lastChar != 0 {
			p.next()
		}
	case '*':
		for {
			p.next()
			if p.lastChar == '*' {
				p.next()
				if p.lastChar == '/' {
					p.next()
					return
				}
			}
			if p.lastChar == 0 {
				panic("Unexpected end of input in comment")
			}
		}
	default:
		panic(fmt.Sprintf("Invalid comment: '/%c'", p.lastChar))
	}
}

func (p *jsonParser) errorSnippet(message string) error {
	lines := strings.Split(p.sourceTail.string(), "\n")
	lastLineLen := utf8.RuneCountInString(lines[len(lines)-1])
	pointer := strings.Repeat(".", lastLineLen-1) + "^"
	lines = append(lines, pointer)

	paddedLines := make([]string, len(lines))
	for i, line := range lines {
		paddedLines[i] = "   " + line
	}

	return fmt.Errorf(
		"%s on line %d.\n\n%s\n\n",
		message,
		p.lineNumber,
		strings.Join(paddedLines, "\n"),
	)
}
