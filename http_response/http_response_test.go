package http_response

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_normalizeHeaderValue(t *testing.T) {
	require.Equal(t, "a  b", string(normalizeHeaderValue([]byte("a  b"))))
	require.Equal(t, "a  b", string(normalizeHeaderValue([]byte("a  b "))))
	require.Equal(t, "a  b", string(normalizeHeaderValue([]byte(" a  b"))))
	require.Equal(t, "a b", string(normalizeHeaderValue([]byte(" a \n b "))))
}

func Test_splitHttpResponse(t *testing.T) {
	_, _, _, ok := splitHttpResponse([]byte("whatever"))
	require.False(t, ok)

	_, _, _, ok = splitHttpResponse([]byte("HTTP/1.1 200 OK\n\nbody"))
	require.False(t, ok)

	status, header, body, ok := splitHttpResponse([]byte("HTTP/1.1 200 OK\nheader1: value\nheader2: value\n with\n continuation\n\nbody"))
	require.True(t, ok)
	require.Equal(t, "HTTP/1.1 200 OK", string(status))
	require.Equal(t, "header1: value\nheader2: value\n with\n continuation\n", string(header))
	require.Equal(t, "body", string(body))

	status, header, body, ok = splitHttpResponse([]byte("HTTP/2 200 \ndate: Sun, 17 Mar 2024 21:58:03 GMT\n\n{}"))
	require.True(t, ok)
	require.Equal(t, "HTTP/2 200 ", string(status))
	require.Equal(t, "date: Sun, 17 Mar 2024 21:58:03 GMT\n", string(header))
	require.Equal(t, "{}", string(body))
}

func Test_splitHeaders(t *testing.T) {
	_, ok := splitHeaders([]byte("whatever"))
	require.False(t, ok)

	_, ok = splitHeaders([]byte("a: b\nwhatever\n"))
	require.False(t, ok)

	res, ok := splitHeaders([]byte("a: b\n"))
	require.True(t, ok)
	require.Len(t, res, 1)
	require.Equal(t, "a", string(res[0].Name()))
	require.Equal(t, "b", string(res[0].Value()))
	require.Equal(t, "b", string(res[0].NormalizedValue()))

	res, ok = splitHeaders([]byte("a: b1\n  b2\n  b3\n"))
	require.True(t, ok)
	require.Len(t, res, 1)
	require.Equal(t, "a", string(res[0].Name()))
	require.Equal(t, "b1\n  b2\n  b3", string(res[0].Value()))
	require.Equal(t, "b1 b2 b3", string(res[0].NormalizedValue()))

	res, ok = splitHeaders([]byte("a: b\nc: d\n"))
	require.True(t, ok)
	require.Len(t, res, 2)
	require.Equal(t, "a", string(res[0].Name()))
	require.Equal(t, "b", string(res[0].Value()))
	require.Equal(t, "c", string(res[1].Name()))
	require.Equal(t, "d", string(res[1].Value()))
}
