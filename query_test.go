package main

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/antonmedv/fx/internal/jsonx"
)

// docList links documents the same way nodeMsg does.
type docList struct {
	head, bottom *jsonx.Node
}

func (l *docList) add(t *testing.T, json string) *jsonx.Node {
	node, err := jsonx.Parse([]byte(json))
	require.NoError(t, err)
	if l.head == nil {
		l.head = node
	} else {
		l.bottom.Adjacent(node)
	}
	l.bottom = node
	return node
}

func parseAll(t *testing.T, p *nodesParser) []string {
	var got []string
	for {
		node, err := p.Parse()
		if err == io.EOF {
			return got
		}
		require.NoError(t, err)
		got = append(got, node.Value)
	}
}

func TestNodesParser_Replay(t *testing.T) {
	var l docList
	l.add(t, `{"a": 1}`)
	l.add(t, `[1, 2]`)
	l.add(t, `"s"`)
	l.add(t, `3`)

	p := newNodesParser(l.head, l.bottom, true)
	require.Equal(t, []string{"{", "[", `"s"`, "3"}, parseAll(t, p))
}

func TestNodesParser_WrappedAndCollapsed(t *testing.T) {
	var l docList
	first := l.add(t, `{"a": {"b": 1}}`)
	first.Collapse() // Collapsed before the next doc is attached.
	l.add(t, `"`+strings.Repeat("long ", 20)+`"`)
	third := l.add(t, `[1, [2]]`)
	l.add(t, `null`)
	third.Collapse() // Collapsed after the next doc is attached.
	jsonx.Wrap(l.head, 10)

	p := newNodesParser(l.head, l.bottom, true)
	got := parseAll(t, p)
	require.Len(t, got, 4)
	require.Equal(t, "{", got[0])
	require.True(t, strings.HasPrefix(got[1], `"long`))
	require.Equal(t, "[", got[2])
	require.Equal(t, "null", got[3])
}

func TestNodesParser_BlocksUntilPublish(t *testing.T) {
	var l docList
	l.add(t, `1`)
	p := newNodesParser(l.head, l.bottom, false)

	results := make(chan string)
	go func() {
		for {
			node, err := p.Parse()
			if err == io.EOF {
				close(results)
				return
			}
			results <- node.Value
		}
	}()

	require.Equal(t, "1", <-results)
	select {
	case v := <-results:
		t.Fatalf("expected Parse to block, got %q", v)
	case <-time.After(50 * time.Millisecond):
	}

	p.publish(l.add(t, `2`))
	require.Equal(t, "2", <-results)

	p.setEOF()
	_, ok := <-results
	require.False(t, ok, "expected EOF after setEOF")
}

func TestNodesParser_EmptyUntilFirstPublish(t *testing.T) {
	var l docList
	p := newNodesParser(nil, nil, false)

	done := make(chan string)
	go func() {
		node, err := p.Parse()
		require.NoError(t, err)
		done <- node.Value
	}()

	p.publish(l.add(t, `"first"`))
	require.Equal(t, `"first"`, <-done)
}

func TestNodesParser_StopUnblocks(t *testing.T) {
	var l docList
	l.add(t, `1`)
	p := newNodesParser(l.head, l.bottom, false)
	_, err := p.Parse()
	require.NoError(t, err)

	done := make(chan error)
	go func() {
		_, err := p.Parse()
		done <- err
	}()

	p.stop()
	select {
	case err := <-done:
		require.Equal(t, io.EOF, err)
	case <-time.After(2 * time.Second):
		t.Fatal("Parse did not return after stop")
	}
}
