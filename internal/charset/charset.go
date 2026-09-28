// Package charset decodes the text encodings an XML declaration or an HTML
// meta tag commonly names. It maps labels by hand to the single-byte and
// UTF-16 encodings of golang.org/x/text, which keeps the CJK tables that
// golang.org/x/net/html/charset would add out of the binary.
package charset

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
)

var (
	utf16le = unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)
	utf16be = unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM)
)

// encodings maps a lowercased label to its encoding. UTF-16 labels map to
// UTF-8 on purpose: a declaration or meta tag naming UTF-16 can only be
// read after Sniff transcoded the input, so the bytes are UTF-8 by then.
var encodings = map[string]encoding.Encoding{
	"utf-8":    unicode.UTF8,
	"utf8":     unicode.UTF8,
	"us-ascii": unicode.UTF8,
	"ascii":    unicode.UTF8,
	"utf-16":   unicode.UTF8,
	"utf-16le": unicode.UTF8,
	"utf-16be": unicode.UTF8,

	"iso-8859-1":  charmap.ISO8859_1,
	"iso-8859-2":  charmap.ISO8859_2,
	"iso-8859-3":  charmap.ISO8859_3,
	"iso-8859-4":  charmap.ISO8859_4,
	"iso-8859-5":  charmap.ISO8859_5,
	"iso-8859-6":  charmap.ISO8859_6,
	"iso-8859-7":  charmap.ISO8859_7,
	"iso-8859-8":  charmap.ISO8859_8,
	"iso-8859-9":  charmap.ISO8859_9,
	"iso-8859-10": charmap.ISO8859_10,
	"iso-8859-13": charmap.ISO8859_13,
	"iso-8859-14": charmap.ISO8859_14,
	"iso-8859-15": charmap.ISO8859_15,
	"iso-8859-16": charmap.ISO8859_16,
	"latin1":      charmap.ISO8859_1,
	"latin2":      charmap.ISO8859_2,
	"latin3":      charmap.ISO8859_3,
	"latin4":      charmap.ISO8859_4,
	"latin5":      charmap.ISO8859_9,
	"latin6":      charmap.ISO8859_10,
	"latin7":      charmap.ISO8859_13,
	"latin8":      charmap.ISO8859_14,
	"latin9":      charmap.ISO8859_15,
	"latin10":     charmap.ISO8859_16,

	"windows-874":  charmap.Windows874,
	"windows-1250": charmap.Windows1250,
	"windows-1251": charmap.Windows1251,
	"windows-1252": charmap.Windows1252,
	"windows-1253": charmap.Windows1253,
	"windows-1254": charmap.Windows1254,
	"windows-1255": charmap.Windows1255,
	"windows-1256": charmap.Windows1256,
	"windows-1257": charmap.Windows1257,
	"windows-1258": charmap.Windows1258,
	"cp874":        charmap.Windows874,
	"cp1250":       charmap.Windows1250,
	"cp1251":       charmap.Windows1251,
	"cp1252":       charmap.Windows1252,
	"cp1253":       charmap.Windows1253,
	"cp1254":       charmap.Windows1254,
	"cp1255":       charmap.Windows1255,
	"cp1256":       charmap.Windows1256,
	"cp1257":       charmap.Windows1257,
	"cp1258":       charmap.Windows1258,

	"koi8-r":      charmap.KOI8R,
	"koi8-u":      charmap.KOI8U,
	"macintosh":   charmap.Macintosh,
	"x-mac-roman": charmap.Macintosh,
	"ibm437":      charmap.CodePage437,
	"cp437":       charmap.CodePage437,
	"ibm850":      charmap.CodePage850,
	"cp850":       charmap.CodePage850,
	"ibm852":      charmap.CodePage852,
	"cp852":       charmap.CodePage852,
	"ibm866":      charmap.CodePage866,
	"cp866":       charmap.CodePage866,
}

// Lookup returns the encoding a label names, or nil. The comparison
// ignores case, surrounding space and the underscore in "iso_8859-1".
func Lookup(label string) encoding.Encoding {
	label = strings.ToLower(strings.TrimSpace(label))
	label = strings.Replace(label, "iso_8859", "iso-8859", 1)
	label = strings.Replace(label, "iso8859", "iso-8859", 1)
	label = strings.Replace(label, "latin-", "latin", 1)
	return encodings[label]
}

// Reader wraps r so it yields UTF-8. It has the signature of
// encoding/xml's Decoder.CharsetReader and fails on an unknown label.
func Reader(label string, r io.Reader) (io.Reader, error) {
	enc := Lookup(label)
	if enc == nil {
		return nil, fmt.Errorf("unsupported encoding %q", label)
	}
	if enc == unicode.UTF8 {
		return r, nil
	}
	return enc.NewDecoder().Reader(r), nil
}

// Sniff reports the encoding a byte order mark or a UTF-16 XML declaration
// reveals, with the mark removed from the returned bytes. It returns nil
// and the input as is when nothing is found.
func Sniff(in []byte) (encoding.Encoding, []byte) {
	switch {
	case bytes.HasPrefix(in, []byte("\xEF\xBB\xBF")):
		return unicode.UTF8, in[3:]
	case bytes.HasPrefix(in, []byte("\xFF\xFE")):
		return utf16le, in[2:]
	case bytes.HasPrefix(in, []byte("\xFE\xFF")):
		return utf16be, in[2:]
	case bytes.HasPrefix(in, []byte("<\x00?\x00")):
		return utf16le, in
	case bytes.HasPrefix(in, []byte("\x00<\x00?")):
		return utf16be, in
	}
	return nil, in
}

// Decode converts in from enc to UTF-8.
func Decode(enc encoding.Encoding, in []byte) ([]byte, error) {
	if enc == unicode.UTF8 {
		return in, nil
	}
	return enc.NewDecoder().Bytes(in)
}
