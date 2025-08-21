package engine

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

func Quote(s string) string {
	var err error
	var b strings.Builder
	b.WriteByte('"')

	for i := 0; i < len(s); {
		r, width := utf8.DecodeRuneInString(s[i:])

		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if r < 0x20 || r == 0x7F {
				// Control characters must be escaped as \uXXXX
				_, err = fmt.Fprintf(&b, `\u%04x`, r)
				if err != nil {
					panic(err)
				}
			} else if r == utf8.RuneError && width == 1 {
				// Invalid UTF-8 sequence - escape the byte
				_, err = fmt.Fprintf(&b, `\u%04x`, s[i])
				if err != nil {
					panic(err)
				}
			} else {
				// Regular character - write as-is
				b.WriteRune(r)
			}
		}

		i += width
	}

	b.WriteByte('"')
	return b.String()
}
