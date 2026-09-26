package engine_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/antonmedv/fx/internal/engine"
)

func TestSplitArgs(t *testing.T) {
	tests := []struct {
		query string
		want  []string
	}{
		{``, nil},
		{`   `, nil},
		{`.`, []string{`.`}},
		{`@.name len`, []string{`@.name`, `len`}},
		{`  .a   .b  `, []string{`.a`, `.b`}},
		{`.users ?.active len`, []string{`.users`, `?.active`, `len`}},
		{`.users.map(x => x.name) uniq`, []string{`.users.map(x => x.name)`, `uniq`}},
		{`x => x.a + 1`, []string{`x => x.a + 1`}},
		{`.a ?? 0`, []string{`.a ?? 0`}},
		{`. .a`, []string{`.`, `.a`}},
		{`.a ? 1 : 2`, []string{`.a ? 1 : 2`}},
		{`.a != null`, []string{`.a != null`}},
		{`.a !== null .b`, []string{`.a !== null`, `.b`}},
		{`.a > 1 && .b`, []string{`.a > 1 && .b`}},
		{`typeof x`, []string{`typeof x`}},
		{`"a" in x`, []string{`"a" in x`}},
		{`new Date .a`, []string{`new Date`, `.a`}},
		{`.x.in .y`, []string{`.x.in`, `.y`}},
		{`{a: 1, b: [1, 2]} .b`, []string{`{a: 1, b: [1, 2]}`, `.b`}},
		{`"a b" 'c d'`, []string{`"a b"`, `'c d'`}},
		{`"a \" b" x`, []string{`"a \" b"`, `x`}},
		{"`a ${x + `b c`} d` x", []string{"`a ${x + `b c`} d`", "x"}},
		{`.filter(x => /[)] a/.test(x)) len`, []string{`.filter(x => /[)] a/.test(x))`, `len`}},
		{`/a b/.test(x) len`, []string{`/a b/.test(x)`, `len`}},
		{`.a / 2 .b`, []string{`.a / 2`, `.b`}},
		{`function (x) { return x.a } len`, []string{`function (x) { return x.a }`, `len`}},
		{`.a (x => x.b)`, []string{`.a`, `(x => x.b)`}},
		{`x => /a/ len`, []string{`x => /a/`, `len`}},
		{`/a+/ len`, []string{`/a+/`, `len`}},
		{`x => "a" + "b" len`, []string{`x => "a" + "b"`, `len`}},
		{`.filter(x => x.a) {n: 1}`, []string{`.filter(x => x.a)`, `{n: 1}`}},
		{`function f(x) { return x } len`, []string{`function f(x) { return x }`, `len`}},
		{`x => { if (x) { return 1 } } len`, []string{`x => { if (x) { return 1 } }`, `len`}},
		{`x++ .a`, []string{`x++`, `.a`}},
		{`x-- .a`, []string{`x--`, `.a`}},
		{`x + ++y .a`, []string{`x + ++y`, `.a`}},
		{`.map(x => (`, []string{`.map(x => (`}},
		{`"unterminated x`, []string{`"unterminated x`}},
		{`.a )) .b`, []string{`.a`, `))`, `.b`}},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			assert.Equal(t, tt.want, engine.SplitArgs(tt.query))
		})
	}
}
