package main

import "testing"

func TestPreviewTransform(t *testing.T) {
	out, err := previewTransform("tr a-z A-Z", "hello world")
	if err != nil {
		t.Fatal(err)
	}
	if out != "HELLO WORLD" {
		t.Errorf("got %q, want %q", out, "HELLO WORLD")
	}
}

func TestPreviewTransformTrimsTrailingNewline(t *testing.T) {
	// jq and most filters end their output with a newline; the preview pane
	// shouldn't carry it.
	out, err := previewTransform("cat", "line\n")
	if err != nil {
		t.Fatal(err)
	}
	if out != "line" {
		t.Errorf("got %q, want %q", out, "line")
	}
}

func TestPreviewTransformError(t *testing.T) {
	if _, err := previewTransform("exit 1", "x"); err == nil {
		t.Error("expected error from failing command")
	}
}

// When FX_PREVIEW is unset the preview path never calls previewTransform, so
// the value is shown verbatim.
func TestPreviewNoTransformWhenUnset(t *testing.T) {
	t.Setenv("FX_PREVIEW", "")
	if got := lookup([]string{"FX_PREVIEW"}, ""); got != "" {
		t.Errorf("expected empty transform, got %q", got)
	}
}
