package charset

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
)

func TestLookup(t *testing.T) {
	require.Equal(t, unicode.UTF8, Lookup("UTF-8"))
	require.Equal(t, unicode.UTF8, Lookup(" utf8 "))
	require.Equal(t, unicode.UTF8, Lookup("UTF-16"))
	require.Equal(t, charmap.ISO8859_1, Lookup("ISO-8859-1"))
	require.Equal(t, charmap.ISO8859_1, Lookup("iso8859-1"))
	require.Equal(t, charmap.ISO8859_1, Lookup("ISO_8859-1"))
	require.Equal(t, charmap.ISO8859_1, Lookup("Latin-1"))
	require.Equal(t, charmap.ISO8859_15, Lookup("latin9"))
	require.Equal(t, charmap.Windows1252, Lookup("Windows-1252"))
	require.Equal(t, charmap.Windows1252, Lookup("cp1252"))
	require.Equal(t, charmap.KOI8R, Lookup("KOI8-R"))
	require.Nil(t, Lookup("shift_jis"))
	require.Nil(t, Lookup(""))
}

func TestReader(t *testing.T) {
	r, err := Reader("windows-1252", strings.NewReader("a\x93b\x94"))
	require.NoError(t, err)
	got, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, "a“b”", string(got))

	in := strings.NewReader("x")
	r, err = Reader("UTF-8", in)
	require.NoError(t, err)
	require.Same(t, in, r, "UTF-8 passes the reader through")

	_, err = Reader("shift_jis", in)
	require.EqualError(t, err, `unsupported encoding "shift_jis"`)
}

func TestSniffAndDecode(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"utf-8 bom", "\xEF\xBB\xBF<a/>", "<a/>"},
		{"utf-16le bom", "\xFF\xFE<\x00a\x00/\x00>\x00", "<a/>"},
		{"utf-16be bom", "\xFE\xFF\x00<\x00a\x00/\x00>", "<a/>"},
		{"utf-16le declaration", "<\x00?\x00x\x00", "<?x"},
		{"utf-16be declaration", "\x00<\x00?\x00x", "<?x"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc, rest := Sniff([]byte(tt.in))
			require.NotNil(t, enc)
			got, err := Decode(enc, rest)
			require.NoError(t, err)
			require.Equal(t, tt.want, string(got))
		})
	}

	enc, rest := Sniff([]byte("<a/>"))
	require.Nil(t, enc)
	require.Equal(t, "<a/>", string(rest))
}
