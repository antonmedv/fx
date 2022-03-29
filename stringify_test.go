package main

import "testing"

func Test_stringify(t *testing.T) {
	t.Run("dict", func(t *testing.T) {
		arg := newDict()
		arg.set("a", number("1"))
		arg.set("b", number("2"))
		want := `{"a": 1, "b": 2}`
		if got := stringify(arg); got != want {
			t.Errorf("stringify() = %v, want %v", got, want)
		}
	})
	t.Run("array", func(t *testing.T) {
		arg := array{number("1"), number("2")}
		want := `[1, 2]`
		if got := stringify(arg); got != want {
			t.Errorf("stringify() = %v, want %v", got, want)
		}
	})
	t.Run("array_with_dict", func(t *testing.T) {
		arg := array{newDict(), array{}}
		want := `[{}, []]`
		if got := stringify(arg); got != want {
			t.Errorf("stringify() = %v, want %v", got, want)
		}
	})
}
