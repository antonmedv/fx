package jsonx

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/antonmedv/fx/internal/utils"
)

type JsonParser struct {
	strict     bool
	rd         io.Reader
	buf        []byte
	data       []byte
	end        int
	eof        bool
	char       byte
	lineNumber int
	depth      uint8
}

func Parse(b []byte) (*Node, error) {
	p := NewJsonParser(bytes.NewReader(b), false)
	node, err := p.Parse()
	if err == io.EOF {
		err = nil
	}
	return node, err
}

func NewJsonParser(rd io.Reader, strict bool) *JsonParser {
	p := &JsonParser{
		strict:     strict,
		rd:         rd,
		buf:        make([]byte, 4096),
		lineNumber: 1,
	}
	p.next() // Should be called here, to support streaming.
	return p
}

func (p *JsonParser) Parse() (node *Node, err error) {
	if p.eof {
		return nil, io.EOF
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
	p.eof = false
	p.depth = 0

	start := p.end - 1
	for {
		p.next()
		if p.eof {
			break
		}
		if p.char == '{' || p.char == '[' {
			break
		}
	}

	end := p.end - 1
	if p.data[end-1] == '\n' {
		end-- // Trim trailing newline.
	}

	start = min(start, end)
	text := string(p.data[start:end])
	text = strings.ReplaceAll(text, "\t", "    ")
	text = strings.ReplaceAll(text, "\r", "")
	lines := strings.Split(text, "\n")

	textNode := &Node{
		Kind:       Err,
		Value:      lines[0],
		Index:      -1,
		LineNumber: p.lineNumberPlusPlus(),
	}
	for i := 1; i < len(lines); i++ {
		textNode.Append(&Node{
			Kind:       Err,
			Value:      lines[i],
			Index:      -1,
			Parent:     textNode,
			LineNumber: p.lineNumberPlusPlus(),
		})
	}
	return textNode
}

func (p *JsonParser) refill() {
	n, err := p.rd.Read(p.buf)
	if err != nil {
		if err == io.EOF {
			p.eof = true
			return
		} else {
			panic(err)
		}
	}
	p.data = append(p.data, p.buf[:n]...)
}

func (p *JsonParser) next() {
	if p.end >= len(p.data) {
		p.refill()
	}
	if p.eof {
		p.char = 0
		p.end = len(p.data) + 1
		return
	}
	p.char = p.data[p.end]
	p.end++
}

func (p *JsonParser) back() {
	p.end--
	p.char = p.data[p.end]
}

func (p *JsonParser) lineNumberPlusPlus() int {
	n := p.lineNumber
	p.lineNumber++
	return n
}

func (p *JsonParser) parseValue() *Node {
	p.skipWhitespace()

	var l *Node
	switch p.char {
	case '"':
		l = p.parseString()
	case '-':
		l = p.parseMinus()
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		l = p.parseNumber(p.end - 1)
	case '{':
		l = p.parseObject()
	case '[':
		l = p.parseArray()
	case 't':
		l = p.parseKeyword("true", Bool)
	case 'f':
		l = p.parseKeyword("false", Bool)
	case 'n':
		l = p.parseNullOrNan()
	case 'N':
		l = p.parseNan(p.end - 1)
	case 'i', 'I':
		l = p.parseInfinity(p.end - 1)
	default:
		panic(fmt.Sprintf("Unexpected character %q", p.char))
	}

	p.skipWhitespace()
	return l
}

func (p *JsonParser) parseString() *Node {
	return &Node{
		Kind:       String,
		Depth:      p.depth,
		Value:      p.scanString(),
		LineNumber: p.lineNumberPlusPlus(),
	}
}

func (p *JsonParser) scanString() string {
	start := p.end - 1
	p.next()
	escaped := false
	for {
		if escaped {
			switch p.char {
			case 'u':
				var unicode string
				for i := 0; i < 4; i++ {
					p.next()
					if !utils.IsHexDigit(p.char) {
						panic(fmt.Sprintf("Invalid Unicode escape sequence '\\u%s%c'", unicode, p.char))
					}
					unicode += string(p.char)
				}
				_, err := strconv.ParseInt(unicode, 16, 32)
				if err != nil {
					panic(fmt.Sprintf("Invalid Unicode escape sequence '\\u%s'", unicode))
				}
			case '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
			default:
				panic(fmt.Sprintf("Invalid escape sequence '\\%c'", p.char))
			}
			escaped = false
		} else if p.char == '\\' {
			escaped = true
		} else if p.char == '"' {
			break
		} else if p.char == 0 {
			panic("Unexpected end of input in string")
		} else if p.char < 0x1F {
			panic(fmt.Sprintf("Invalid character %q in string", p.char))
		}
		p.next()
	}

	str := string(p.data[start:p.end])
	p.next()

	return str
}

func (p *JsonParser) parseMinus() *Node {
	start := p.end - 1
	p.next()
	switch p.char {
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return p.parseNumber(start)
	case 'n', 'N':
		return p.parseNan(start)
	case 'i', 'I':
		return p.parseInfinity(start)
	}
	panic(fmt.Sprintf("Invalid character %q in number", p.char))
}

func (p *JsonParser) parseNumber(start int) *Node {
	num := &Node{
		Kind:       Number,
		Depth:      p.depth,
		LineNumber: p.lineNumberPlusPlus(),
	}

	// Leading zero
	if p.char == '0' {
		p.next()
	} else {
		for utils.IsDigit(p.char) {
			p.next()
		}
	}

	// Decimal portion
	if p.char == '.' {
		p.next()
		if !utils.IsDigit(p.char) {
			panic(fmt.Sprintf("Invalid character %q in number", p.char))
		}
		for utils.IsDigit(p.char) {
			p.next()
		}
	}

	// Exponent
	if p.char == 'e' || p.char == 'E' {
		p.next()
		if p.char == '+' || p.char == '-' {
			p.next()
		}
		if !utils.IsDigit(p.char) {
			panic(fmt.Sprintf("Invalid character %q in number", p.char))
		}
		for utils.IsDigit(p.char) {
			p.next()
		}
	}

	num.Value = string(p.data[start : p.end-1])
	return num
}

func (p *JsonParser) parseObject() *Node {
	object := &Node{
		Kind:       Object,
		Depth:      p.depth,
		LineNumber: p.lineNumberPlusPlus(),
	}
	object.Value = curlyBracketOpen

	p.next()
	p.skipWhitespace()

	// Empty object
	if p.char == '}' {
		object.Value = curlyBracketPair
		p.next()
		return object
	}

	for {
		// Expecting a key which should be a string
		if p.char != '"' {
			panic(fmt.Sprintf("Expected object key to be a string, got %q", p.char))
		}

		keyBytes := p.scanString()

		p.skipWhitespace()

		// Expecting colon after key
		if p.char != ':' {
			panic(fmt.Sprintf("Expected colon after object key, got %q", p.char))
		}

		p.next()

		p.depth++
		value := p.parseValue()
		value.Key = keyBytes
		value.Parent = object
		p.depth--

		object.Append(value)
		object.Size += 1

		p.skipWhitespace()

		if p.char == ',' {
			object.End.Comma = true
			p.next()
			p.skipWhitespace()
			if p.char == '}' {
				if p.strict {
					panic("Trailing comma is not allowed in strict mode")
				}
				object.End.Comma = false
			} else {
				continue
			}
		}

		if p.char == '}' {
			closeBracket := &Node{
				Kind:       Object,
				Depth:      p.depth,
				LineNumber: p.lineNumberPlusPlus(),
			}
			closeBracket.Value = curlyBracketClose
			closeBracket.Parent = object
			closeBracket.Index = -1
			object.Append(closeBracket)
			p.next()
			return object
		}

		panic(fmt.Sprintf("Unexpected character %q in object", p.char))
	}
}

func (p *JsonParser) parseArray() *Node {
	arr := &Node{
		Kind:       Array,
		Depth:      p.depth,
		LineNumber: p.lineNumberPlusPlus(),
	}
	arr.Value = squareBracketOpen

	p.next()
	p.skipWhitespace()

	if p.char == ']' {
		arr.Value = squareBracketPair
		p.next()
		return arr
	}

	for i := 0; ; i++ {
		p.depth++
		value := p.parseValue()
		value.Parent = arr
		arr.Size += 1
		value.Index = i
		p.depth--

		arr.Append(value)
		p.skipWhitespace()

		if p.char == ',' {
			arr.End.Comma = true
			p.next()
			p.skipWhitespace()
			if p.char == ']' {
				if p.strict {
					panic("Trailing comma is not allowed in strict mode")
				}
				arr.End.Comma = false
			} else {
				continue
			}
		}

		if p.char == ']' {
			closeBracket := &Node{
				Kind:       Array,
				Depth:      p.depth,
				LineNumber: p.lineNumberPlusPlus(),
			}
			closeBracket.Value = squareBracketClose
			closeBracket.Parent = arr
			closeBracket.Index = -1
			arr.Append(closeBracket)
			p.next()
			return arr
		}

		panic(fmt.Sprintf("Invalid character %q in array", p.char))
	}
}

func (p *JsonParser) parseKeyword(name string, kind Kind) *Node {
	start := p.end - 1
	for i := 1; i < len(name); i++ {
		p.next()
		if p.char != name[i] {
			panic(fmt.Sprintf("Unexpected character %q in keyword", p.char))
		}
	}
	p.next()
	if isEndOfValue(p.char) {
		keyword := &Node{
			Kind:       kind,
			Depth:      p.depth,
			Value:      string(p.data[start : p.end-1]),
			LineNumber: p.lineNumberPlusPlus(),
		}
		return keyword
	}

	panic(fmt.Sprintf("Unexpected character %q in keyword", p.char))
}

func (p *JsonParser) parseNullOrNan() *Node {
	p.next()
	if p.char == 'u' {
		p.next()
		if p.char == 'l' {
			p.next()
			if p.char == 'l' {
				p.next()
				if isEndOfValue(p.char) {
					return &Node{
						Kind:       Null,
						Depth:      p.depth,
						Value:      "null",
						LineNumber: p.lineNumberPlusPlus(),
					}
				}
			}
		}
	} else if p.char == 'a' {
		p.back() // Put back the 'a'.
		return p.parseNan(p.end - 1)
	}
	panic(fmt.Sprintf("Unexpected character %q", p.char))
}

func (p *JsonParser) parseNan(start int) *Node {
	p.next()
	if !p.strict {
		if p.char == 'a' || p.char == 'A' {
			p.next()
			if p.char == 'n' || p.char == 'N' {
				p.next()
				if isEndOfValue(p.char) {
					return &Node{
						Kind:       NaN,
						Depth:      p.depth,
						Value:      string(p.data[start : p.end-1]),
						LineNumber: p.lineNumberPlusPlus(),
					}
				}
			}
		}
	}
	panic(fmt.Sprintf("Unexpected character %q", p.char))
}

func (p *JsonParser) parseInfinity(start int) *Node {
	p.next()
	if !p.strict {
		if p.char == 'n' || p.char == 'N' {
			p.next()
			if p.char == 'f' || p.char == 'F' {
				p.next()
				if isEndOfValue(p.char) {
					return &Node{
						Kind:       Infinity,
						Depth:      p.depth,
						Value:      string(p.data[start : p.end-1]),
						LineNumber: p.lineNumberPlusPlus(),
					}
				}
				if p.char == 'i' {
					p.next()
					if p.char == 'n' {
						p.next()
						if p.char == 'i' {
							p.next()
							if p.char == 't' {
								p.next()
								if p.char == 'y' {
									p.next()
									if isEndOfValue(p.char) {
										return &Node{
											Kind:       Infinity,
											Depth:      p.depth,
											Value:      string(p.data[start : p.end-1]),
											LineNumber: p.lineNumberPlusPlus(),
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
	panic(fmt.Sprintf("Unexpected character %q", p.char))
}

func isEndOfValue(ch byte) bool {
	return isWhitespace(ch) || ch == ',' || ch == '}' || ch == ']' || ch == 0 // 0 is EOF
}

func isWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}

func (p *JsonParser) skipWhitespace() {
	for {
		switch p.char {
		case ' ', '\t', '\n', '\r':
			p.next()
		case '/':
			if p.strict {
				panic("Comments are not allowed in strict mode")
			}
			p.skipComment()
		default:
			return
		}
	}
}

func (p *JsonParser) skipComment() {
	p.next()
	switch p.char {
	case '/':
		for p.char != '\n' && p.char != 0 {
			p.next()
		}
	case '*':
		for {
			p.next()
			if p.char == '*' {
				p.next()
				if p.char == '/' {
					p.next()
					return
				}
			}
			if p.char == 0 {
				panic("Unexpected end of input in comment")
			}
		}
	default:
		panic(fmt.Sprintf("Invalid comment: '/%c'", p.char))
	}
}

func (p *JsonParser) errorSnippet(message string) error {
	var buf []byte
	br := 0
	start := max(0, p.end-70)
	end := min(p.end, len(p.data))
	for i := end - 1; i >= start; i-- {
		if p.data[i] == '\n' {
			br++
			if br == 2 {
				break
			}
		}
		buf = append(buf, p.data[i])
	}
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}

	tail := strings.TrimRight(string(buf), "\t \n")
	lines := strings.Split(tail, "\n")
	lastLineLen := max(1, utf8.RuneCountInString(lines[len(lines)-1]))
	pointer := strings.Repeat(".", lastLineLen-1) + "^"
	lines = append(lines, pointer)

	paddedLines := make([]string, len(lines))
	for i, line := range lines {
		paddedLines[i] = "   " + line
	}

	return fmt.Errorf(
		"%s on line %d.\n\n%s\n",
		message,
		p.lineNumber,
		strings.Join(paddedLines, "\n"),
	)
}
