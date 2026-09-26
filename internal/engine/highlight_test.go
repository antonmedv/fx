package engine_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/antonmedv/fx/internal/engine"
)

// marked renders code with every token wrapped in a marker of its kind:
// s(tring), n(umber), k(eyword), b(oolean), 0 (null), p(roperty), c(omment),
// punctuation unmarked.
func marked(code string) string {
	letters := map[engine.TokenKind]string{
		engine.TokenString: "s", engine.TokenNumber: "n", engine.TokenKeyword: "k",
		engine.TokenBoolean: "b", engine.TokenNull: "0", engine.TokenProperty: "p",
		engine.TokenComment: "c",
	}
	var out strings.Builder
	last := 0
	for _, t := range engine.Tokenize(code) {
		out.WriteString(code[last:t.Start])
		if letter, ok := letters[t.Kind]; ok {
			out.WriteString(letter + "(" + code[t.Start:t.End] + ")")
		} else {
			out.WriteString(code[t.Start:t.End])
		}
		last = t.End
	}
	out.WriteString(code[last:])
	return out.String()
}

func TestTokenize(t *testing.T) {
	tests := []struct{ code, want string }{
		{``, ``},
		{`.name`, `.p(name)`},
		{`@.name len`, `@.p(name) len`},
		{`?.active`, `?.p(active)`},
		{`x?.a.b`, `x?.p(a).p(b)`},
		{`{...rest}`, `{...rest}`},
		{`"a b" 'c\' d'`, `s("a b") s('c\' d')`},
		{`"unterminated x`, `s("unterminated x)`},
		{`1 0.5 1e-3 0xFF 10n`, `n(1) n(0.5) n(1e-3) n(0xFF) n(10n)`},
		{`0xE-1`, `n(0xE)-n(1)`},
		{`true false null undefined`, `b(true) b(false) 0(null) 0(undefined)`},
		{`typeof x === "a"`, `k(typeof) x === s("a")`},
		{`x => { return this }`, `x => { k(return) k(this) }`},
		{`.a.true`, `.p(a).p(true)`},
		{"`a ${x.b + 1} c`", "s(`a )${x.p(b) + n(1)}s( c`)"},
		{"`${`in ${1}`}`", "s(`)${s(`in )${n(1)}s(`)}s(`)"},
		{"`a ${x", "s(`a )${x"},
		{"`a ${ {b: 1}", "s(`a )${ {b: n(1)}"},
		{`/a b/gi.test(x)`, `s(/a b/gi).p(test)(x)`},
		{`.a / 2 / 3`, `.p(a) / n(2) / n(3)`},
		{`x.filter(y => /[/]/.test(y))`, `x.p(filter)(y => s(/[/]/).p(test)(y))`},
		{`x // note`, `x c(// note)`},
		{`x /* a */ + 1`, `x c(/* a */) + n(1)`},
		{`.ключ "日本"`, `.p(ключ) s("日本")`},
	}
	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			assert.Equal(t, tt.want, marked(tt.code))
		})
	}
}
