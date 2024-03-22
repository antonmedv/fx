package http_response

import (
	"bytes"
	"regexp"
)

type HttpResponseHeader interface {
	Name() []byte
	Value() []byte
	NormalizedValue() []byte
}

type HttpResponse interface {
	Status() []byte
	Headers() []HttpResponseHeader
	Body() []byte
}

type httpResponseHeader struct {
	name            []byte
	value           []byte
	normalizedValue []byte
}

func (h httpResponseHeader) Name() []byte {
	return h.name
}

func (h httpResponseHeader) Value() []byte {
	return h.value
}

func (h httpResponseHeader) NormalizedValue() []byte {
	if h.normalizedValue == nil {
		h.normalizedValue = normalizeHeaderValue(h.value)
	}
	return h.normalizedValue
}

var normalizeValueRe *regexp.Regexp

func normalizeHeaderValue(value []byte) []byte {
	if normalizeValueRe == nil {
		str := `\s*\n\s+`
		normalizeValueRe = regexp.MustCompile(str)
	}

	return normalizeValueRe.ReplaceAllLiteral(bytes.TrimSpace(value), []byte{' '})
}

type FailedParsingHttpResponseError struct {
	error string
}

func (e FailedParsingHttpResponseError) Error() string {
	return e.error
}

var splitHttpResponseRe *regexp.Regexp

func splitHttpResponse(text []byte) (status []byte, headers []byte, body []byte, ok bool) {
	if splitHttpResponseRe == nil {
		str := `(?ms)\A(?P<status>(?:HTTP/\d\.\d \d{3} \w+|HTTP/2 [^\n]+))\n(?P<headers>.*?\n)\n(?P<body>.*)\z`
		splitHttpResponseRe = regexp.MustCompile(str)
	}

	match := splitHttpResponseRe.FindSubmatch(text)
	if match == nil {
		return
	}

	status = match[1]
	headers = match[2]
	body = match[3]
	ok = true

	return
}

var splitHeadersRe *regexp.Regexp

func splitHeaders(text []byte) (headers []HttpResponseHeader, ok bool) {
	if splitHeadersRe == nil {
		str := `(?m)\A(?:^(?P<field>[^:\s]+):\s+(?P<value>.*$\n(?:^\s+.*$\n)*))`
		splitHeadersRe = regexp.MustCompile(str)
	}

	for p := 0; p < len(text); {
		match := splitHeadersRe.FindSubmatch(text[p:])
		if match == nil {
			return nil, false
		}
		name := match[1]
		value := bytes.TrimRight(match[2], "\n")
		headers = append(headers, httpResponseHeader{name, value, nil})
		p += len(match[0])
	}

	ok = true

	return
}

func normalizeEols(text []byte) []byte {
	return bytes.ReplaceAll(text, []byte{'\r', '\n'}, []byte{'\n'})
}

type httpResponse struct {
	status  []byte
	headers []HttpResponseHeader
	body    []byte
}

func NewHttpResponseFromText(text []byte) (HttpResponse, error) {
	status, combinedHeaders, body, ok := splitHttpResponse(normalizeEols(text))
	if !ok {
		return nil, FailedParsingHttpResponseError{"failed extracting status, headers or body"}
	}

	headers, ok := splitHeaders(combinedHeaders)
	if !ok {
		return nil, FailedParsingHttpResponseError{"failed parsing headers"}
	}

	return httpResponse{status, headers, body}, nil
}

func (r httpResponse) Status() []byte {
	return r.status
}

func (r httpResponse) Headers() []HttpResponseHeader {
	return r.headers
}

func (r httpResponse) Body() []byte {
	return r.body
}
