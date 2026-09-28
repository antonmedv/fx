// Package edn converts EDN (Extensible Data Notation) to JSON.
package edn

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/antonmedv/fx/internal/engine"
)

// ToJSON converts EDN to a stream of JSON documents, one per top-level
// form, separated by newlines. Map keys keep their order. EDN has more
// types than JSON; they are mapped as follows:
//
//	nil, true, false            null, true, false
//	"string", \c                string
//	:keyword, :ns/keyword       "keyword", "ns/keyword"
//	symbol                      "symbol"
//	42, 42N, 4.2, 4.2M, 1e3     number, N and M suffixes dropped
//	(list), [vector], #{set}    array
//	{map}                       object; a key whose JSON form is not
//	                            a string keeps its EDN text
//	#tag value                  value, tag dropped
//	##Inf, ##-Inf, ##NaN        Infinity, -Infinity, NaN
//	#_ form, ; comment          dropped
func ToJSON(in []byte) ([]byte, error) {
	p := &parser{src: bytes.TrimPrefix(in, []byte("\xEF\xBB\xBF"))}
	for {
		if err := p.skipDiscards(); err != nil {
			return nil, err
		}
		if p.pos >= len(p.src) {
			return p.out.Bytes(), nil
		}
		if p.out.Len() > 0 {
			p.out.WriteByte('\n')
		}
		if err := p.readForm(); err != nil {
			return nil, err
		}
	}
}

type parser struct {
	src []byte
	pos int
	out bytes.Buffer
}

func (p *parser) errorf(at int, format string, args ...any) error {
	line := 1 + bytes.Count(p.src[:at], []byte{'\n'})
	lineStart := bytes.LastIndexByte(p.src[:at], '\n') + 1
	column := 1 + utf8.RuneCount(p.src[lineStart:at])
	return fmt.Errorf("edn: line %d, column %d: %s", line, column, fmt.Sprintf(format, args...))
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == ',' || c == '\f' || c == '\v'
}

// isDelimiter reports whether c ends a token.
func isDelimiter(c byte) bool {
	if isSpace(c) {
		return true
	}
	switch c {
	case '(', ')', '[', ']', '{', '}', '"', ';', '\\', '@', '^', '`', '~':
		return true
	}
	return false
}

func isDigit(c byte) bool {
	return '0' <= c && c <= '9'
}

func isLetter(c byte) bool {
	return 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z'
}

// skipSpace skips whitespace, comments and a leading #! line.
func (p *parser) skipSpace() {
	for p.pos < len(p.src) {
		c := p.src[p.pos]
		switch {
		case isSpace(c):
			p.pos++
		case c == ';' || c == '#' && p.pos == 0 && bytes.HasPrefix(p.src, []byte("#!")):
			for p.pos < len(p.src) && p.src[p.pos] != '\n' {
				p.pos++
			}
		default:
			return
		}
	}
}

// skipDiscards skips whitespace, comments and #_ forms.
func (p *parser) skipDiscards() error {
	for {
		p.skipSpace()
		if !bytes.HasPrefix(p.src[p.pos:], []byte("#_")) {
			return nil
		}
		p.pos += 2
		mark := p.out.Len()
		if err := p.readForm(); err != nil {
			return err
		}
		p.out.Truncate(mark)
	}
}

// readForm writes the next form to out as JSON.
func (p *parser) readForm() error {
	if err := p.skipDiscards(); err != nil {
		return err
	}
	if p.pos >= len(p.src) {
		return p.errorf(p.pos, "unexpected end of input")
	}
	start := p.pos
	c := p.src[p.pos]
	switch c {
	case '(':
		p.pos++
		return p.readSeq(start, ')', "list")
	case '[':
		p.pos++
		return p.readSeq(start, ']', "vector")
	case '{':
		p.pos++
		return p.readMap(start)
	case '"':
		return p.readString()
	case '\\':
		return p.readChar()
	case '#':
		return p.readDispatch()
	}
	if isDelimiter(c) {
		return p.errorf(start, "unexpected %q", c)
	}
	return p.readToken()
}

// readSeq reads the elements of a list, vector or set up to close as a
// JSON array. The opening delimiter, at start, is already consumed.
func (p *parser) readSeq(start int, close byte, what string) error {
	p.out.WriteByte('[')
	first := true
	for {
		if err := p.skipDiscards(); err != nil {
			return err
		}
		if p.pos >= len(p.src) {
			return p.errorf(start, "unclosed %s", what)
		}
		if p.src[p.pos] == close {
			p.pos++
			break
		}
		if !first {
			p.out.WriteByte(',')
		}
		first = false
		if err := p.readForm(); err != nil {
			return err
		}
	}
	p.out.WriteByte(']')
	return nil
}

// readMap reads a map as a JSON object. The opening brace, at start, is
// already consumed.
func (p *parser) readMap(start int) error {
	p.out.WriteByte('{')
	first := true
	for {
		if err := p.skipDiscards(); err != nil {
			return err
		}
		if p.pos >= len(p.src) {
			return p.errorf(start, "unclosed map")
		}
		if p.src[p.pos] == '}' {
			p.pos++
			break
		}
		if !first {
			p.out.WriteByte(',')
		}
		first = false
		if err := p.readKey(); err != nil {
			return err
		}
		p.out.WriteByte(':')
		if err := p.skipDiscards(); err != nil {
			return err
		}
		if p.pos >= len(p.src) {
			return p.errorf(start, "unclosed map")
		}
		if p.src[p.pos] == '}' {
			return p.errorf(p.pos, "map key without value")
		}
		if err := p.readForm(); err != nil {
			return err
		}
	}
	p.out.WriteByte('}')
	return nil
}

// readKey reads a map key as a JSON string. A key whose JSON form is not
// a string (a keyword, symbol or character is one) keeps its EDN text.
func (p *parser) readKey() error {
	start := p.pos
	mark := p.out.Len()
	if err := p.readForm(); err != nil {
		return err
	}
	if p.out.Bytes()[mark] == '"' {
		return nil
	}
	text := string(p.src[start:p.pos])
	p.out.Truncate(mark)
	p.out.WriteString(engine.Quote(text))
	return nil
}

func (p *parser) readString() error {
	start := p.pos
	p.pos++ // opening quote
	var b strings.Builder
	for {
		if p.pos >= len(p.src) {
			return p.errorf(start, "unclosed string")
		}
		c := p.src[p.pos]
		switch c {
		case '"':
			p.pos++
			p.out.WriteString(engine.Quote(b.String()))
			return nil
		case '\\':
			r, err := p.readEscape(start)
			if err != nil {
				return err
			}
			b.WriteRune(r)
		default:
			b.WriteByte(c)
			p.pos++
		}
	}
}

// readEscape reads an escape sequence in a string, at a backslash. The
// string starts at strStart.
func (p *parser) readEscape(strStart int) (rune, error) {
	start := p.pos
	p.pos++ // backslash
	if p.pos >= len(p.src) {
		return 0, p.errorf(strStart, "unclosed string")
	}
	c := p.src[p.pos]
	p.pos++
	switch c {
	case 't':
		return '\t', nil
	case 'r':
		return '\r', nil
	case 'n':
		return '\n', nil
	case '\\':
		return '\\', nil
	case '"':
		return '"', nil
	case 'b':
		return '\b', nil
	case 'f':
		return '\f', nil
	case 'u':
		r, ok := p.readHex4()
		if !ok {
			return 0, p.errorf(start, "invalid escape %q", p.text(start, 6))
		}
		if isHighSurrogate(r) && bytes.HasPrefix(p.src[p.pos:], []byte(`\u`)) {
			// A surrogate pair is two \u escapes.
			p.pos += 2
			low, ok := p.readHex4()
			if !ok {
				return 0, p.errorf(start, "invalid escape %q", p.text(start, 12))
			}
			if isLowSurrogate(low) {
				return utf16.DecodeRune(r, low), nil
			}
			p.pos -= 6 // Not a pair: the second escape stands on its own.
		}
		return r, nil
	}
	return 0, p.errorf(start, "invalid escape %q", p.text(start, 2))
}

func isHighSurrogate(r rune) bool { return 0xD800 <= r && r < 0xDC00 }
func isLowSurrogate(r rune) bool  { return 0xDC00 <= r && r < 0xE000 }

// readHex4 reads four hex digits.
func (p *parser) readHex4() (rune, bool) {
	if p.pos+4 > len(p.src) {
		return 0, false
	}
	n, err := strconv.ParseUint(string(p.src[p.pos:p.pos+4]), 16, 32)
	if err != nil {
		return 0, false
	}
	p.pos += 4
	return rune(n), true
}

// text returns up to n bytes of the source from start, for errors.
func (p *parser) text(start, n int) string {
	return string(p.src[start:min(start+n, len(p.src))])
}

// readChar reads a character literal, at a backslash.
func (p *parser) readChar() error {
	start := p.pos
	p.pos++ // backslash
	if p.pos >= len(p.src) {
		return p.errorf(start, "unexpected end of input")
	}
	// The first character is taken as is, so \( and \; are valid.
	_, size := utf8.DecodeRune(p.src[p.pos:])
	p.pos += size
	for p.pos < len(p.src) && !isDelimiter(p.src[p.pos]) {
		p.pos++
	}
	name := string(p.src[start+1 : p.pos])
	var s string
	switch {
	case utf8.RuneCountInString(name) == 1:
		s = name
	case name == "newline":
		s = "\n"
	case name == "space":
		s = " "
	case name == "tab":
		s = "\t"
	case name == "return":
		s = "\r"
	case name == "backspace":
		s = "\b"
	case name == "formfeed":
		s = "\f"
	case len(name) == 5 && name[0] == 'u':
		n, err := strconv.ParseUint(name[1:], 16, 32)
		if err != nil {
			return p.errorf(start, "invalid character %q", "\\"+name)
		}
		s = string(rune(n))
	case len(name) >= 2 && len(name) <= 4 && name[0] == 'o':
		n, err := strconv.ParseUint(name[1:], 8, 32)
		if err != nil || n > 0377 {
			return p.errorf(start, "invalid character %q", "\\"+name)
		}
		s = string(rune(n))
	default:
		return p.errorf(start, "invalid character %q", "\\"+name)
	}
	p.out.WriteString(engine.Quote(s))
	return nil
}

// readDispatch reads a form starting with #: a set, a tagged element or a
// symbolic value. #_ is handled by skipDiscards.
func (p *parser) readDispatch() error {
	start := p.pos
	p.pos++ // #
	if p.pos >= len(p.src) {
		return p.errorf(start, "unexpected end of input")
	}
	c := p.src[p.pos]
	switch {
	case c == '{':
		p.pos++
		return p.readSeq(start, '}', "set")
	case c == '#':
		p.pos++
		for p.pos < len(p.src) && !isDelimiter(p.src[p.pos]) {
			p.pos++
		}
		switch string(p.src[start+2 : p.pos]) {
		case "Inf":
			p.out.WriteString("Infinity")
		case "-Inf":
			p.out.WriteString("-Infinity")
		case "NaN":
			p.out.WriteString("NaN")
		default:
			return p.errorf(start, "unknown symbolic value %q", p.text(start, p.pos-start))
		}
		return nil
	case isLetter(c):
		// A tagged element: the tag is dropped, the value kept.
		for p.pos < len(p.src) && !isDelimiter(p.src[p.pos]) {
			p.pos++
		}
		return p.readForm()
	}
	r, _ := utf8.DecodeRune(p.src[p.pos:])
	return p.errorf(start, "unexpected %q", "#"+string(r))
}

// readToken reads nil, a boolean, a number, a keyword or a symbol.
func (p *parser) readToken() error {
	start := p.pos
	for p.pos < len(p.src) && !isDelimiter(p.src[p.pos]) {
		p.pos++
	}
	tok := string(p.src[start:p.pos])
	switch tok {
	case "nil":
		p.out.WriteString("null")
		return nil
	case "true", "false":
		p.out.WriteString(tok)
		return nil
	}
	if tok[0] == '\'' {
		return p.errorf(start, "unexpected %q", tok[0])
	}
	if tok[0] == ':' {
		if len(tok) == 1 || tok[1] == ':' {
			return p.errorf(start, "invalid keyword %q", tok)
		}
		p.out.WriteString(engine.Quote(tok[1:]))
		return nil
	}
	if looksNumeric(tok) {
		n, ok := number(tok)
		if !ok {
			return p.errorf(start, "invalid number %q", tok)
		}
		p.out.WriteString(n)
		return nil
	}
	p.out.WriteString(engine.Quote(tok))
	return nil
}

// looksNumeric reports whether tok must be a number: it starts with a
// digit, or with a sign or dot followed by a digit (.5, -.5), which
// symbols may not.
func looksNumeric(tok string) bool {
	i := 0
	if tok[i] == '+' || tok[i] == '-' {
		i++
	}
	if i < len(tok) && tok[i] == '.' {
		i++
	}
	return i < len(tok) && isDigit(tok[i])
}

// number converts an EDN number to a JSON number.
func number(tok string) (string, bool) {
	s := strings.TrimPrefix(tok, "+")
	if n := len(s); n > 1 {
		switch s[n-1] {
		case 'N':
			// Arbitrary precision integers only.
			if strings.ContainsAny(s, ".eE") {
				return "", false
			}
			s = s[:n-1]
		case 'M':
			s = s[:n-1]
		}
	}
	i := 0
	if i < len(s) && s[i] == '-' {
		i++
	}
	if i >= len(s) {
		return "", false
	}
	if s[i] == '0' {
		i++
	} else {
		if !isDigit(s[i]) {
			return "", false
		}
		for i < len(s) && isDigit(s[i]) {
			i++
		}
	}
	if i < len(s) && s[i] == '.' {
		i++
		digits := i
		for i < len(s) && isDigit(s[i]) {
			i++
		}
		if i == digits {
			return "", false
		}
	}
	if i < len(s) && (s[i] == 'e' || s[i] == 'E') {
		i++
		if i < len(s) && (s[i] == '+' || s[i] == '-') {
			i++
		}
		digits := i
		for i < len(s) && isDigit(s[i]) {
			i++
		}
		if i == digits {
			return "", false
		}
	}
	return s, i == len(s)
}
