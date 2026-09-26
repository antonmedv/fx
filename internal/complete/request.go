package complete

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/antonmedv/fx/internal/engine"
)

type mode uint8

const (
	modeNone    mode = iota
	modeKey          // `.partial`
	modeBracket      // `[` or `["partial`
	modeGlobal       // an identifier not after a dot, like `le` for len
)

// Request is the word being completed in an fx query, with the arguments
// before it.
type Request struct {
	Args    []string // arguments before Word
	Word    string   // the argument being completed, possibly empty
	base    string   // Word before the accessor being completed
	partial string   // key (or global) prefix typed so far, unescaped
	mode    mode
	quote   byte // bracket mode: the quote typed after `[`, 0 if none yet
}

// NewRequest parses the word being completed.
func NewRequest(args []string, word string) *Request {
	r := &Request{Args: args, Word: word}
	r.parseWord()
	return r
}

// ParseQuery splits an interactive query, as typed up to the cursor, into a
// request. start is the byte offset of the word in the query.
func ParseQuery(query string) (r *Request, start int) {
	args := engine.SplitArgs(query)
	if query == "" || isSpace(query[len(query)-1]) || len(args) == 0 {
		return NewRequest(args, ""), len(query)
	}
	word := args[len(args)-1]
	return NewRequest(args[:len(args)-1], word), len(query) - len(word)
}

func (r *Request) parseWord() {
	w := r.Word
	if w == "" {
		r.mode = modeKey
		return
	}

	// `["partial` or `['partial`: an unterminated string right after `[`.
	for i := strings.LastIndexByte(w, '['); i >= 0; i = strings.LastIndexByte(w[:i], '[') {
		if i+1 < len(w) && (w[i+1] == '"' || w[i+1] == '\'') {
			if s, ok := unterminated(w[i+2:], w[i+1]); ok {
				r.mode, r.base, r.partial, r.quote = modeBracket, w[:i], s, w[i+1]
			}
			break
		}
	}
	if r.mode != modeNone {
		return
	}

	if w[len(w)-1] == '[' {
		r.mode, r.base = modeBracket, w[:len(w)-1]
		return
	}

	j := len(w)
	for j > 0 && isIdentChar(w[j-1]) {
		j--
	}
	switch {
	case j > 0 && w[j-1] == '.':
		r.mode, r.base, r.partial = modeKey, w[:j-1], w[j:]
	case j < len(w) && !isDigit(w[j]) && (j == 0 || !isQuote(w[j-1])):
		r.mode, r.base, r.partial = modeGlobal, w[:j], w[j:]
	}
}

// unterminated returns the content of a string literal missing its closing
// quote, unescaped. ok is false if the string is closed.
func unterminated(s string, quote byte) (string, bool) {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case quote:
			return "", false
		case '\\':
			if i+1 < len(s) {
				i++
				b.WriteByte(s[i])
			}
		default:
			b.WriteByte(s[i])
		}
	}
	return b.String(), true
}

// path is an fx argument made of property accesses only, walkable without JS.
type path struct {
	maps  int    // leading @, each maps over an array
	steps []step // accesses, in order
	each  bool   // fx accepts `[]` after this path
	empty bool   // no accessor yet: a bracket needs a leading dot, as in `.["key"]`
}

type stepKind uint8

const (
	stepKey   stepKind = iota // .key or ["key"]
	stepIndex                 // [0]
	stepEach                  // [], flatMap over an array
)

type step struct {
	kind  stepKind
	key   string
	index int
}

var (
	// Same as reBracket in engine's transpile: `[]` is only accepted after
	// plain dotted identifiers.
	reBracket  = regexp.MustCompile(`^(\.\w*)+\[]`)
	reEachHead = regexp.MustCompile(`^(\.\w*)+$`)
)

// parsePath parses arg as transpile would, if it is a plain path: `.`,
// `.a.b`, `.a[0]["k"]`, `.a[].b`, `@.a`, `x.a`. A base (the part of a word
// before the key being completed) may also be empty, meaning `.`.
func parsePath(arg string, base bool) (path, bool) {
	var p path
	for strings.HasPrefix(arg, "@") {
		p.maps++
		arg = arg[1:]
	}
	var ok bool
	switch {
	case arg == "" && base:
		// The word `.key` or `@.key`: the key is accessed on x itself.
		p.each, p.empty, ok = true, true, true
	case arg == ".":
		p.each, p.empty, ok = true, true, true
	case arg == "x" || arg == "this":
		ok = true
	case strings.HasPrefix(arg, "x.") || strings.HasPrefix(arg, "x["):
		p.steps, ok = chain(arg[1:], nil)
	case strings.HasPrefix(arg, "this.") || strings.HasPrefix(arg, "this["):
		p.steps, ok = chain(arg[4:], nil)
	case reBracket.MatchString(arg):
		segs := strings.Split(arg, "[]")
		ok = true
		if segs[0] != "." {
			p.steps, ok = chain(segs[0], nil)
		}
		for _, seg := range segs[1:] {
			if !ok {
				break
			}
			p.steps = append(p.steps, step{kind: stepEach})
			p.steps, ok = chain(seg, p.steps)
		}
		p.each = true
	case strings.HasPrefix(arg, ".["):
		p.steps, ok = chain(arg[1:], nil)
	case strings.HasPrefix(arg, "."):
		p.steps, ok = chain(arg, nil)
		p.each = reEachHead.MatchString(arg)
	}
	return p, ok
}

// chain parses property accesses `.key`, `[0]` and `["key"]`, appending them
// to steps.
func chain(s string, steps []step) ([]step, bool) {
	for i := 0; i < len(s); {
		switch s[i] {
		case '.':
			j := i + 1
			for j < len(s) && isIdentChar(s[j]) {
				j++
			}
			key := s[i+1 : j]
			if key == "" || isDigit(key[0]) {
				return nil, false
			}
			steps = append(steps, step{kind: stepKey, key: key})
			i = j
		case '[':
			if i+1 >= len(s) {
				return nil, false
			}
			if q := s[i+1]; q == '"' || q == '\'' {
				key, n, ok := stringLiteral(s[i+1:])
				if !ok || i+1+n >= len(s) || s[i+1+n] != ']' {
					return nil, false
				}
				steps = append(steps, step{kind: stepKey, key: key})
				i += n + 2
				continue
			}
			j := i + 1
			for j < len(s) && isDigit(s[j]) {
				j++
			}
			digits := s[i+1 : j]
			if digits == "" || j >= len(s) || s[j] != ']' || len(digits) > 1 && digits[0] == '0' {
				return nil, false
			}
			index, err := strconv.Atoi(digits)
			if err != nil {
				return nil, false
			}
			steps = append(steps, step{kind: stepIndex, index: index})
			i = j + 1
		default:
			return nil, false
		}
	}
	return steps, true
}

// stringLiteral decodes the JS string literal at the start of s, returning
// its length. Escapes other than simple ones are left to the engine.
func stringLiteral(s string) (string, int, bool) {
	quote := s[0]
	var b strings.Builder
	for i := 1; i < len(s); i++ {
		c := s[i]
		switch {
		case c == quote:
			return b.String(), i + 1, true
		case c == '\\':
			if i+1 >= len(s) {
				return "", 0, false
			}
			i++
			switch s[i] {
			case '\\', '"', '\'', '/':
				b.WriteByte(s[i])
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case 'r':
				b.WriteByte('\r')
			case 'b':
				b.WriteByte('\b')
			case 'f':
				b.WriteByte('\f')
			default:
				return "", 0, false
			}
		case c == '\n':
			return "", 0, false
		default:
			b.WriteByte(c)
		}
	}
	return "", 0, false
}

func isIdentChar(c byte) bool {
	return c == '_' || c == '$' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || isDigit(c)
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func isQuote(c byte) bool {
	return c == '"' || c == '\'' || c == '`'
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

var identRe = regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*$`)

// quoteSingle is engine.Quote with single quotes: control characters are
// escaped too, as a string literal cannot hold a raw newline.
func quoteSingle(key string) string {
	var b strings.Builder
	b.WriteByte('\'')
	for _, r := range key {
		switch r {
		case '\'':
			b.WriteString(`\'`)
		case '"':
			b.WriteByte('"')
		default:
			q := engine.Quote(string(r))
			b.WriteString(q[1 : len(q)-1])
		}
	}
	b.WriteByte('\'')
	return b.String()
}

// accessor returns the JS to access key: `.key`, or `["key"]` if key is not
// an identifier. dot is prepended to a bracket, for `.["key"]` at the start.
func accessor(key string, dot bool, quote byte) string {
	if quote == 0 && identRe.MatchString(key) {
		return "." + key
	}
	var s string
	if quote == '\'' {
		s = quoteSingle(key)
	} else {
		s = engine.Quote(key)
	}
	if dot {
		return ".[" + s + "]"
	}
	return "[" + s + "]"
}
