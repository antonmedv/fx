package jsonx

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/antonmedv/fx/internal/utils"
)

type JsonParser struct {
	buf        *bufio.Reader
	data       []byte
	err        error
	end        int
	lastChar   byte
	lineNumber uint
	sourceTail *ring
	depth      uint8
}

func Parse(b []byte) (*Node, error) {
	p := NewParser(bytes.NewReader(b))
	node, err := p.Parse()
	if err == io.EOF {
		err = nil
	}
	return node, err
}

func NewParser(in io.Reader) *JsonParser {
	p := &JsonParser{
		buf:        bufio.NewReader(in),
		lineNumber: 1,
		sourceTail: &ring{},
	}
	p.next()
	return p
}

func (p *JsonParser) Parse() (node *Node, err error) {
	if p.err != nil {
		return nil, p.err
	}
	defer func() {
		if r := recover(); r != nil {
			err = p.errorSnippet(fmt.Sprintf("%v", r))
		}
	}()
	node = p.parseValue()
	return
}

func (p *JsonParser) Recover() *Node {
	p.err = nil
	p.depth = 0

	start := p.end - 1
	for {
		p.next()
		if p.lastChar == '{' || p.lastChar == '[' {
			break
		}
	}
	nonJson := p.data[start : p.end-1]
	lines := strings.Split(string(nonJson), "\n")

	textNode := &Node{Err: []byte(lines[0])}
	for i := 1; i < len(lines); i++ {
		textNode.Append(&Node{Err: []byte(lines[i])})
	}
	return textNode
}

func (p *JsonParser) next() {
	ch, err := p.buf.ReadByte()
	if err != nil {
		if err == io.EOF {
			p.err = err
		} else {
			panic(err)
		}
	}
	p.lastChar = ch
	p.data = append(p.data, ch)
	p.end++
	if p.lastChar == '\n' {
		p.lineNumber++
	}
	p.sourceTail.writeByte(p.lastChar)
}

func (p *JsonParser) parseValue() *Node {
	p.skipWhitespace()

	var l *Node
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

func (p *JsonParser) parseString() *Node {
	str := &Node{Depth: p.depth}
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
					if !utils.IsHexDigit(p.lastChar) {
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

	str.Value = p.data[start:p.end]
	p.next()
	return str
}

func (p *JsonParser) parseNumber() *Node {
	num := &Node{Depth: p.depth}
	start := p.end - 1

	// Handle negative numbers
	if p.lastChar == '-' {
		p.next()
		if !utils.IsDigit(p.lastChar) {
			panic(fmt.Sprintf("Invalid character %q in number", p.lastChar))
		}
	}

	// Leading zero
	if p.lastChar == '0' {
		p.next()
	} else {
		for utils.IsDigit(p.lastChar) {
			p.next()
		}
	}

	// Decimal portion
	if p.lastChar == '.' {
		p.next()
		if !utils.IsDigit(p.lastChar) {
			panic(fmt.Sprintf("Invalid character %q in number", p.lastChar))
		}
		for utils.IsDigit(p.lastChar) {
			p.next()
		}
	}

	// Exponent
	if p.lastChar == 'e' || p.lastChar == 'E' {
		p.next()
		if p.lastChar == '+' || p.lastChar == '-' {
			p.next()
		}
		if !utils.IsDigit(p.lastChar) {
			panic(fmt.Sprintf("Invalid character %q in number", p.lastChar))
		}
		for utils.IsDigit(p.lastChar) {
			p.next()
		}
	}

	num.Value = p.data[start : p.end-1]
	return num
}

func (p *JsonParser) parseObject() *Node {
	object := &Node{Depth: p.depth}
	object.Value = []byte{'{'}

	p.next()
	p.skipWhitespace()

	// Empty object
	if p.lastChar == '}' {
		object.Value = append(object.Value, '}')
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
		key.Key, key.Value = key.Value, nil
		object.Size += 1
		key.directParent = object

		p.skipWhitespace()

		// Expecting colon after key
		if p.lastChar != ':' {
			panic(fmt.Sprintf("Expected colon after object key, got %q", p.lastChar))
		}

		p.next()

		value := p.parseValue()
		p.depth--

		key.Value = value.Value
		key.Size = value.Size
		key.Next = value.Next
		if key.Next != nil {
			key.Next.Prev = key
		}
		key.End = value.End
		value.indirectParent = key
		object.Append(key)

		p.skipWhitespace()

		if p.lastChar == ',' {
			object.End.Comma = true
			p.next()
			p.skipWhitespace()
			if p.lastChar == '}' {
				object.End.Comma = false
			} else {
				continue
			}
		}

		if p.lastChar == '}' {
			closeBracket := &Node{Depth: p.depth}
			closeBracket.Value = []byte{'}'}
			closeBracket.directParent = object
			closeBracket.Index = -1
			object.Append(closeBracket)
			p.next()
			return object
		}

		panic(fmt.Sprintf("Unexpected character %q in object", p.lastChar))
	}
}

func (p *JsonParser) parseArray() *Node {
	arr := &Node{Depth: p.depth}
	arr.Value = []byte{'['}

	p.next()
	p.skipWhitespace()

	if p.lastChar == ']' {
		arr.Value = append(arr.Value, ']')
		p.next()
		return arr
	}

	for i := 0; ; i++ {
		p.depth++
		value := p.parseValue()
		value.directParent = arr
		arr.Size += 1
		value.Index = i
		p.depth--

		arr.Append(value)
		p.skipWhitespace()

		if p.lastChar == ',' {
			arr.End.Comma = true
			p.next()
			p.skipWhitespace()
			if p.lastChar == ']' {
				arr.End.Comma = false
			} else {
				continue
			}
		}

		if p.lastChar == ']' {
			closeBracket := &Node{Depth: p.depth}
			closeBracket.Value = []byte{']'}
			closeBracket.directParent = arr
			closeBracket.Index = -1
			arr.Append(closeBracket)
			p.next()
			return arr
		}

		panic(fmt.Sprintf("Invalid character %q in array", p.lastChar))
	}
}

func (p *JsonParser) parseKeyword(name string) *Node {
	for i := 1; i < len(name); i++ {
		p.next()
		if p.lastChar != name[i] {
			panic(fmt.Sprintf("Unexpected character %q in keyword", p.lastChar))
		}
	}
	p.next()

	nextCharIsSpecial := isWhitespace(p.lastChar) || p.lastChar == ',' || p.lastChar == '}' || p.lastChar == ']' || p.lastChar == 0
	if nextCharIsSpecial {
		keyword := &Node{Depth: p.depth}
		keyword.Value = []byte(name)
		return keyword
	}

	panic(fmt.Sprintf("Unexpected character %q in keyword", p.lastChar))
}

func isWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}

func (p *JsonParser) skipWhitespace() {
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

func (p *JsonParser) skipComment() {
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

func (p *JsonParser) errorSnippet(message string) error {
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
