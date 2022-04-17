package json

import (
	. "github.com/antonmedv/fx/pkg/dict"
	"testing"
)

func Test_stringify(t *testing.T) {
	t.Run("dict", func(t *testing.T) {
		arg := NewDict()
		arg.Set("a", Number("1"))
		arg.Set("b", Number("2"))
		want := `{"a": 1,"b": 2}`
		if got := Stringify(arg); got != want {
			t.Errorf("stringify() = %v, want %v", got, want)
		}
	})
	t.Run("array", func(t *testing.T) {
		arg := Array{Number("1"), Number("2")}
		want := `[1,2]`
		if got := Stringify(arg); got != want {
			t.Errorf("stringify() = %v, want %v", got, want)
		}
	})
	t.Run("array_with_dict", func(t *testing.T) {
		arg := Array{NewDict(), Array{}}
		want := `[{},[]]`
		if got := Stringify(arg); got != want {
			t.Errorf("stringify() = %v, want %v", got, want)
		}
	})
}
