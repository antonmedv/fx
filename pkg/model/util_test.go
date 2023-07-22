package model

import "testing"

func Test_clamp(t *testing.T) {
	got := clamp(1, 2, 3)
	if got != 2 {
		t.Errorf("clamp() = %v, want 2", got)
	}
}

func Test_max(t *testing.T) {
	got := max(1, 2)
	if got != 2 {
		t.Errorf("max() = %v, want 2", got)
	}
}

func Test_min(t *testing.T) {
	got := min(1, 2)
	if got != 1 {
		t.Errorf("min() = %v, want 1", got)
	}
}
