package jsonx

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParser_DirectoryIsError(t *testing.T) {
	dir, err := os.Open(t.TempDir())
	require.NoError(t, err)
	defer dir.Close()
	var p *JsonParser
	require.NotPanics(t, func() { p = NewJsonParser(dir, false) })
	_, err = p.Parse()
	require.Error(t, err)
	assert.NotEqual(t, io.EOF, err)
	assert.Nil(t, p.Recover(), "nothing to recover past a read error")
	_, err2 := p.Parse()
	assert.Equal(t, err, err2, "the read error stays")
	more, err3 := p.More()
	assert.False(t, more)
	assert.Equal(t, err, err3)
}

func TestParser_ReadErrorMidway(t *testing.T) {
	boom := errors.New("boom")
	r := io.MultiReader(strings.NewReader(`{"a": [1, 2`), iotest.ErrReader(boom))
	p := NewJsonParser(r, false)
	_, err := p.Parse()
	assert.Equal(t, boom, err)
	assert.Nil(t, p.Recover())
}

func TestParser_BytesWithEOF(t *testing.T) {
	p := NewJsonParser(iotest.DataErrReader(strings.NewReader(`{"a":[1,2,3]}`)), false)
	node, err := p.Parse()
	require.NoError(t, err)
	assert.Equal(t, `{"a":[1,2,3]}`, node.String())
}

// zeroReader returns no bytes and no error, forever.
type zeroReader struct{}

func (zeroReader) Read([]byte) (int, error) { return 0, nil }

func TestParser_NoProgressIsError(t *testing.T) {
	p := NewJsonParser(zeroReader{}, false)
	_, err := p.Parse()
	assert.Equal(t, io.ErrNoProgress, err)
}

func TestParser_MaxNesting(t *testing.T) {
	ok := strings.Repeat("[", MaxNesting) + strings.Repeat("]", MaxNesting)
	_, err := Parse([]byte(ok))
	require.NoError(t, err)

	deep := strings.Repeat("[", 5_000_000)
	_, err = Parse([]byte(deep))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Exceeded max depth of 10000")

	deep = strings.Repeat(`{"a":`, MaxNesting+1) + "1" + strings.Repeat("}", MaxNesting+1)
	_, err = Parse([]byte(deep))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Exceeded max depth")
}

func TestParser_NestingResetsPerValue(t *testing.T) {
	half := strings.Repeat("[", MaxNesting) + strings.Repeat("]", MaxNesting)
	p := NewJsonParser(strings.NewReader(half+"\n"+half), false)
	for range 2 {
		_, err := p.Parse()
		require.NoError(t, err)
	}
}
