package complete

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/antonmedv/fx/internal/format"
	"github.com/antonmedv/fx/internal/jsonx"
)

func TestKeysComplete(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		args     []string
		compWord string
		want     []string
	}{
		{
			name:     "simple object keys with empty compWord",
			json:     `{"foo": 1, "bar": 2, "baz": 3}`,
			args:     []string{"fx", "file.json"},
			compWord: "",
			want:     []string{".foo", ".bar", ".baz"},
		},
		{
			name:     "nested object keys with trailing dot",
			json:     `{"outer": {"inner1": 1, "inner2": 2}}`,
			args:     []string{"fx", "file.json", ".outer."},
			compWord: ".outer.",
			want:     []string{".outer.inner1", ".outer.inner2"},
		},
		{
			name:     "nested object without trailing dot returns root keys",
			json:     `{"outer": {"inner1": 1, "inner2": 2}}`,
			args:     []string{"fx", "file.json", ".outer"},
			compWord: ".outer",
			want:     []string{".outer"},
		},
		{
			name:     "nested object with partial compWord",
			json:     `{"data": {"name": "test", "value": 42}}`,
			args:     []string{"fx", "file.json", ".data."},
			compWord: ".data.",
			want:     []string{".data.name", ".data.value"},
		},
		{
			name:     "empty object",
			json:     `{}`,
			args:     []string{"fx", "file.json"},
			compWord: "",
			want:     nil,
		},
		{
			name:     "key with special characters",
			json:     `{"normal": 1, "with-dash": 2, "with space": 3}`,
			args:     []string{"fx", "file.json"},
			compWord: "",
			want:     []string{".normal", ".[\"with-dash\"]", ".[\"with space\"]"},
		},
		{
			name:     "deeply nested with trailing dot",
			json:     `{"a": {"b": {"c": {"d": 1}}}}`,
			args:     []string{"fx", "file.json", ".a.b.c."},
			compWord: ".a.b.c.",
			want:     []string{".a.b.c.d"},
		},
		{
			name:     "deeply nested without trailing dot returns parent keys",
			json:     `{"a": {"b": {"c": {"d": 1}}}}`,
			args:     []string{"fx", "file.json", ".a.b.c"},
			compWord: ".a.b.c",
			want:     []string{".a.b.c"},
		},
		{
			name:     "array without objects completes array methods",
			json:     `[1, 2, 3]`,
			args:     []string{"fx", "file.json", ".fla"},
			compWord: ".fla",
			want:     []string{".flat", ".flatMap"},
		},
		{
			name:     "string completes string methods",
			json:     `"hello"`,
			args:     []string{"fx", "file.json", ".toUp"},
			compWord: ".toUp",
			want:     []string{".toUpperCase"},
		},
		{
			name:     "boolean has no completions",
			json:     `true`,
			args:     []string{"fx", "file.json"},
			compWord: "",
			want:     nil,
		},
		{
			name:     "numeric keys use bracket notation",
			json:     `{"123": "numeric", "abc": "alpha"}`,
			args:     []string{"fx", "file.json"},
			compWord: "",
			want:     []string{".[\"123\"]", ".abc"},
		},
		{
			name:     "key starting with underscore",
			json:     `{"_private": 1, "public": 2}`,
			args:     []string{"fx", "file.json"},
			compWord: "",
			want:     []string{"._private", ".public"},
		},
		{
			name:     "key starting with dollar",
			json:     `{"$ref": "#/defs", "name": "test"}`,
			args:     []string{"fx", "file.json"},
			compWord: "",
			want:     []string{".$ref", ".name"},
		},
		{
			name:     "object inside array via bracket access with trailing dot",
			json:     `{"items": [{"x": 1, "y": 2}]}`,
			args:     []string{"fx", "file.json", ".items[0]."},
			compWord: ".items[0].",
			want:     []string{".items[0].x", ".items[0].y"},
		},
		{
			name:     "multiple args combines path",
			json:     `{"a": {"b": {"c": 1}}}`,
			args:     []string{"fx", "file.json", ".a", ".b."},
			compWord: ".b.",
			want:     []string{".b.c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := jsonx.Parse([]byte(tt.json))
			if err != nil {
				t.Fatalf("failed to parse JSON: %v", err)
			}

			got := KeysComplete(node, tt.args, tt.compWord)

			if len(got) != len(tt.want) {
				t.Errorf("KeysComplete() returned %d replies, want %d", len(got), len(tt.want))
				t.Errorf("got: %v", replyValues(got))
				t.Errorf("want: %v", tt.want)
				return
			}

			gotValues := replyValues(got)
			for i, want := range tt.want {
				if gotValues[i] != want {
					t.Errorf("KeysComplete()[%d].Value = %q, want %q", i, gotValues[i], want)
				}
			}
		})
	}
}

func replyValues(replies []Reply) []string {
	values := make([]string, len(replies))
	for i, r := range replies {
		values[i] = r.Value
	}
	return values
}

func TestKeysComplete_Display(t *testing.T) {
	json := `{"foo": 1, "bar": 2}`
	node, err := jsonx.Parse([]byte(json))
	if err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	got := KeysComplete(node, []string{"fx", "file.json"}, "")

	for _, r := range got {
		if r.Type != "key" {
			t.Errorf("expected Type to be 'key', got %q", r.Type)
		}
		if r.Display == "" {
			t.Error("expected Display to be non-empty")
		}
	}
}

func TestKeysComplete_DisplayVsValue(t *testing.T) {
	tests := []struct {
		name        string
		json        string
		args        []string
		compWord    string
		wantDisplay string
		wantValue   string
	}{
		{
			name:        "display shows key with dot prefix, value shows full path",
			json:        `{"outer": {"inner": 1}}`,
			args:        []string{"fx", "file.json", ".outer."},
			compWord:    ".outer.",
			wantDisplay: ".inner",
			wantValue:   ".outer.inner",
		},
		{
			name:        "special key display and value both use bracket notation",
			json:        `{"key-dash": 1}`,
			args:        []string{"fx", "file.json"},
			compWord:    "",
			wantDisplay: ".[\"key-dash\"]",
			wantValue:   ".[\"key-dash\"]",
		},
		{
			name:        "regular key at root",
			json:        `{"foo": 1}`,
			args:        []string{"fx", "file.json"},
			compWord:    "",
			wantDisplay: ".foo",
			wantValue:   ".foo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := jsonx.Parse([]byte(tt.json))
			if err != nil {
				t.Fatalf("failed to parse JSON: %v", err)
			}

			got := KeysComplete(node, tt.args, tt.compWord)
			if len(got) == 0 {
				t.Fatal("expected at least one reply")
			}

			if got[0].Display != tt.wantDisplay {
				t.Errorf("Display = %q, want %q", got[0].Display, tt.wantDisplay)
			}
			if got[0].Value != tt.wantValue {
				t.Errorf("Value = %q, want %q", got[0].Value, tt.wantValue)
			}
		})
	}
}

// doCompleteOutput runs doComplete on a file holding json and returns what
// it prints, one reply per line.
func doCompleteOutput(t *testing.T, json, words string) []string {
	return doCompleteFile(t, "file.json", json, "", words)
}

// doCompleteFile runs doComplete on `fx <flags> <file> <words>`, with a file
// called name holding content, and returns what it prints, one reply per
// line.
func doCompleteFile(t *testing.T, name, content, flags, words string) []string {
	file := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(file, []byte(content), 0644))

	r, w, err := os.Pipe()
	require.NoError(t, err)
	stdout := os.Stdout
	os.Stdout = w
	line := "fx " + flags + file + " " + words
	doComplete(line, lastWord(line), false)
	os.Stdout = stdout
	require.NoError(t, w.Close())
	out, err := io.ReadAll(r)
	require.NoError(t, err)
	if len(out) == 0 {
		return nil
	}
	return strings.Split(string(out), "\n")
}

func TestDoComplete_ArrayRewrite(t *testing.T) {
	json := `{"users": [{"name": 1}, {"age": 2}], "user": {"id": 1}}`
	// `.users[].name` does not start with the typed `.users.`, but replaces it.
	require.Equal(t, []string{".users[].name", ".users[].age"}, doCompleteOutput(t, json, ".users."))
	require.Equal(t, []string{".users", ".user"}, doCompleteOutput(t, json, ".us"))
	require.Equal(t, []string{"map"}, doCompleteOutput(t, json, "ma"))
}

func TestDoComplete_Formats(t *testing.T) {
	// main registers the format flags in Flags; filterArgs needs them.
	saved := Flags
	for _, f := range format.All {
		Flags = append(Flags, Reply{Display: f.Flag, Value: f.Flag, Type: "flag"})
	}
	t.Cleanup(func() { Flags = saved })

	want := []string{".zebra", ".alpha"}
	require.Equal(t, want, doCompleteFile(t, "file.yaml", "zebra: 1\nalpha: 2\n", "", "."))
	require.Equal(t, want, doCompleteFile(t, "file.toml", "zebra = 1\nalpha = 2\n", "", "."))
	require.Equal(t, want, doCompleteFile(t, "file.edn", "{:zebra 1 :alpha 2}", "", "."))
	// A flag wins over the extension.
	require.Equal(t, want, doCompleteFile(t, "file.txt", "{:zebra 1 :alpha 2}", "--edn ", "."))
	require.Equal(t, want, doCompleteFile(t, "file.yaml", "zebra = 1\nalpha = 2\n", "--toml ", "."))
	// Malformed input gives no candidates and does not panic.
	require.Nil(t, doCompleteFile(t, "file.toml", "a = [1\n", "", "."))
	require.Nil(t, doCompleteFile(t, "file.edn", "{:a", "", "."))
	require.Nil(t, doCompleteFile(t, "file.yaml", "a: [1", "", "."))
}
