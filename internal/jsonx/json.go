package jsonx

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"

	"github.com/antonmedv/fx/internal/utils"
)

type JsonParser struct {
	strict         bool
	rd             io.Reader
	buf            []byte
	data           []byte
	end            int
	eof            bool
	char           byte
	lineNumber     int
	realLineNumber int
	depth          uint8
	count          int
	err            error // error found by More, until Recover
	readFailed     bool  // err is a read error: no more input
	nesting        int   // Depth without wrapping, to limit recursion.
	valueStart     int   // Offset in data of the value being parsed.
}

// MaxNesting is the deepest nesting of arrays and objects parsed, as in
// encoding/json: the parser recurses, and a stack overflow is fatal.
const MaxNesting = 10_000

// enter descends into a value of an array or object.
func (p *JsonParser) enter() {
	p.nesting++
	if p.nesting > MaxNesting {
		panic(fmt.Sprintf("Exceeded max depth of %d nested arrays and objects", MaxNesting))
	}
	p.depth++
}

func (p *JsonParser) leave() {
	p.nesting--
	p.depth--
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
		strict:         strict,
		rd:             rd,
		buf:            make([]byte, 4096),
		lineNumber:     1,
		realLineNumber: 1,
	}
	// Read here, to support streaming. A read error, as on a directory, is
	// returned by Parse.
	func() {
		defer p.recoverReadError()
		p.next()
		p.skipBOM()
	}()
	return p
}

// readError is the panic of a failed read of the input.
type readError struct {
	err error
}

// recoverReadError keeps a read error, which every later call returns: the
// input can't be read past it.
func (p *JsonParser) recoverReadError() {
	if r := recover(); r != nil {
		e, ok := r.(readError)
		if !ok {
			panic(r)
		}
		p.failRead(e.err)
	}
}

// failRead keeps err, the input can't be read past it.
func (p *JsonParser) failRead(err error) {
	p.err, p.eof, p.readFailed = err, true, true
}

// skipBOM skips a UTF-8 byte order mark at the start of the input.
func (p *JsonParser) skipBOM() {
	if p.char != 0xEF {
		return
	}
	for _, b := range []byte{0xEF, 0xBB, 0xBF} {
		if p.char != b {
			// Not a BOM: rewind to the first byte.
			p.end, p.char, p.eof, p.realLineNumber = 1, p.data[0], false, 1
			return
		}
		p.next()
	}
}

func (p *JsonParser) Parse() (node *Node, err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(readError); ok {
				p.failRead(e.err)
				node, err = nil, e.err
				return
			}
			err = p.errorSnippet(fmt.Sprintf("%v", r))
		}
	}()
	if p.err != nil {
		return nil, p.err
	}
	p.nesting = 0
	p.markValueStart()
	p.skipWhitespace()
	if p.eof {
		return nil, io.EOF
	}
	p.markValueStart()
	node = p.parseValue(true)
	p.count++
	p.markValueStart() // Parsed: nothing to recover before here.
	return
}

// More reports whether input remains after the values parsed so far. It
// reads ahead, so on a stream it blocks until more input or EOF arrives.
func (p *JsonParser) More() (more bool, err error) {
	if p.err != nil {
		return false, p.err
	}
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(readError); ok {
				p.failRead(e.err)
			} else {
				p.err = p.errorSnippet(fmt.Sprintf("%v", r))
			}
			more, err = false, p.err
		}
	}()
	p.markValueStart()
	p.skipWhitespace()
	p.markValueStart()
	return !p.eof, nil
}

// markValueStart records the current character as where the next value
// starts: Recover returns the text of a value that failed from there. It is
// marked before skipping whitespace too, as a bad comment fails in it.
func (p *JsonParser) markValueStart() {
	p.valueStart = p.end - 1
}

// Recover returns the text from the last error to the next value, or nil
// if the input can't be read further.
func (p *JsonParser) Recover() (node *Node) {
	if p.readFailed {
		return nil
	}
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(readError); ok {
				p.failRead(e.err)
				node = nil
				return
			}
			panic(r)
		}
	}()
	p.err = nil
	p.eof = false
	p.depth = 0
	p.nesting = 0

	// The text of the value that failed, from its start: the error is
	// somewhere in it, as in INFO, read as the start of Infinity.
	start := p.valueStart
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

	start = max(0, min(start, end))
	// A value failing in a comment starts before the whitespace skipped.
	for start < end && (p.data[start] == '\n' || p.data[start] == '\r') {
		start++
	}
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
	// A reader may return no bytes and no error; give up after many such
	// reads, as bufio does, instead of looping forever.
	for range 100 {
		n, err := p.rd.Read(p.buf)
		// Bytes come first: a reader may return them with io.EOF.
		p.data = append(p.data, p.buf[:n]...)
		switch {
		case err == io.EOF && n == 0:
			p.eof = true
			return
		case err != nil && err != io.EOF:
			panic(readError{err})
		case n > 0:
			return
		}
	}
	panic(readError{io.ErrNoProgress})
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
	if p.char == '\n' {
		p.realLineNumber++
	}
	p.end++
}

func (p *JsonParser) back() {
	p.end--
	p.char = p.data[p.end]
}

func (p *JsonParser) set(pos int) {
	p.end = pos
	p.char = p.data[p.end]
}

func (p *JsonParser) lineNumberPlusPlus() int {
	n := p.lineNumber
	p.lineNumber++
	return n
}

func (p *JsonParser) parseValue(root bool) *Node {
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
	case 'u':
		if p.strict {
			panic(fmt.Sprintf("Unexpected character %q", p.char))
		}
		l = p.parseKeyword("undefined", Undefined)
	default:
		panic(fmt.Sprintf("Unexpected character %q", p.char))
	}

	// Skip whitespace will block parseValue (with io.Read in refill func),
	// as soon as we parsed the root value, return and ignore remining whitespaces.
	if !root {
		p.skipWhitespace()
	}

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
			escaped = false
			if p.strict {
				switch p.char {
				case 'u':
					var s string
					for i := 0; i < 4; i++ {
						p.next()
						if !utils.IsHexDigit(p.char) {
							panic(fmt.Sprintf("Invalid Unicode escape sequence '\\u%s%c'", s, p.char))
						}
						s += string(p.char)
					}
					_, err := strconv.ParseInt(s, 16, 32)
					if err != nil {
						panic(fmt.Sprintf("Invalid Unicode escape sequence '\\u%s'", s))
					}
				case '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
				default:
					panic(fmt.Sprintf("Invalid escape sequence '\\%c'", p.char))
				}
			}
		} else if p.char == '\\' {
			escaped = true
		} else if p.char == '"' {
			break
		} else if p.char == 0 {
			panic("Unexpected end of input in string")
		} else if rune(p.char) > unicode.MaxRune {
			panic(fmt.Sprintf("Invalid character code point %d in string", p.char))
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
	}
	if !p.strict {
		switch p.char {
		case 'n', 'N':
			return p.parseNan(start)
		case 'i', 'I':
			return p.parseInfinity(start)
		}
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

		p.enter()
		value := p.parseValue(false)
		value.Key = keyBytes
		value.Parent = object
		p.leave()

		object.Append(value)
		object.Size += 1

		p.skipWhitespace()

		commaPos := p.end
		if p.char == ',' {
			object.End.Comma = true
			p.next()
			p.skipWhitespace()
			if p.char == '}' {
				if p.strict {
					p.set(commaPos)
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
		p.enter()
		value := p.parseValue(false)
		value.Parent = arr
		arr.Size += 1
		value.Index = i
		p.leave()

		arr.Append(value)
		p.skipWhitespace()

		commaPos := p.end
		if p.char == ',' {
			arr.End.Comma = true
			p.next()
			p.skipWhitespace()
			if p.char == ']' {
				if p.strict {
					p.set(commaPos)
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
	if p.strict {
		panic(fmt.Sprintf("Unexpected character %q", p.char))
	}
	p.next()
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

	panic(fmt.Sprintf("Unexpected character %q", p.char))
}

func (p *JsonParser) parseInfinity(start int) *Node {
	if p.strict {
		panic(fmt.Sprintf("Unexpected character %q", p.char))
	}
	p.next()
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
