package edn

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestToJSON(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"nil", `nil`, `null`},
		{"booleans", `[true false]`, `[true,false]`},
		{"integers", `[0 42 -7 +5 123456789012345678901234567890N -0]`, `[0,42,-7,5,123456789012345678901234567890,-0]`},
		{"floats", `[1.5 -0.25 1e3 1.5E-3 2e+2 1.5M 12M]`, `[1.5,-0.25,1e3,1.5E-3,2e+2,1.5,12]`},
		{"symbolic values", `[##Inf ##-Inf ##NaN]`, `[Infinity,-Infinity,NaN]`},
		{"string", `"hello"`, `"hello"`},
		{"string escapes", `"a\tb\nc\"d\\e\rf\bg\fh"`, `"a\tb\nc\"d\\e\rf\bg\fh"`},
		{"string unicode escape", `"\u00e9\u0041"`, `"éA"`},
		{"string surrogate pair", `"\ud83d\ude00"`, `"😀"`},
		{"string lone surrogates", `"\ud83d\u0041\ude00\ud83d"`, "\"\uFFFDA\uFFFD\uFFFD\""},
		{"string multiline", "\"a\nb\"", `"a\nb"`},
		{"string unicode", `"привет 😀"`, `"привет 😀"`},
		{"characters", `[\a \newline \space \tab \return \backspace \formfeed \u00e9 \o101 \( \; \é]`, `["a","\n"," ","\t","\r","\b","\f","é","A","(",";","é"]`},
		{"keywords", `[:a :ns/name :a.b/c-d? :x_y]`, `["a","ns/name","a.b/c-d?","x_y"]`},
		{"symbols", `[foo foo/bar - + -> a.b/c* nil? true! *x* $ x']`, `["foo","foo/bar","-","+","->","a.b/c*","nil?","true!","*x*","$","x'"]`},
		{"list", `(1 2 3)`, `[1,2,3]`},
		{"vector", `[1 "a" :b c]`, `[1,"a","b","c"]`},
		{"set", `#{1 2 3}`, `[1,2,3]`},
		{"empty collections", `[() [] #{} {}]`, `[[],[],[],{}]`},
		{"map", `{:a 1 :b 2}`, `{"a":1,"b":2}`},
		{"map keeps order", `{:zebra 1 :alpha 2 :mid 3}`, `{"zebra":1,"alpha":2,"mid":3}`},
		{"map string keys", `{"a b" 1}`, `{"a b":1}`},
		{"map symbol keys", `{org.clojure/clojure {:mvn/version "1.11.1"}}`, `{"org.clojure/clojure":{"mvn/version":"1.11.1"}}`},
		{"map other keys", `{1 "int" 1.5 "float" nil "nil" true "bool" \c "char" [1 2] "vec" {:a 1} "map" #{1} "set" (1) "list" #t 1 "tagged" 2N "big"}`,
			`{"1":"int","1.5":"float","nil":"nil","true":"bool","c":"char","[1 2]":"vec","{:a 1}":"map","#{1}":"set","(1)":"list","#t 1":"tagged","2N":"big"}`},
		{"nested", `{:a [1 {:b (2 #{3})}]}`, `{"a":[1,{"b":[2,[3]]}]}`},
		{"commas", `{:a 1, :b [1, 2,, 3],}`, `{"a":1,"b":[1,2,3]}`},
		{"tagged inst", `#inst "2020-01-01T00:00:00Z"`, `"2020-01-01T00:00:00Z"`},
		{"tagged uuid", `#uuid "f81d4fae-7dec-11d0-a765-00a0c91e6bf6"`, `"f81d4fae-7dec-11d0-a765-00a0c91e6bf6"`},
		{"tagged custom", `#my/tag {:a 1}`, `{"a":1}`},
		{"tagged nested", `#a #b [1]`, `[1]`},
		{"tagged no space", `#foo[1]`, `[1]`},
		{"discard", `[1 #_ 2 3]`, `[1,3]`},
		{"discard form", `[1 #_{:a [1 2]} 3]`, `[1,3]`},
		{"discard stacked", `[#_ #_ 1 2 3]`, `[3]`},
		{"discard last", `[1 #_ 2]`, `[1]`},
		{"discard in map", `{:a #_ 1 2 #_ :x :b 3}`, `{"a":2,"b":3}`},
		{"discard top-level", `#_ 1 2`, `2`},
		{"discard only", `#_ 1`, ``},
		{"comment", "{:a 1 ; comment\n :b 2}", `{"a":1,"b":2}`},
		{"comment shebang", "#!/usr/bin/env fx\n{:a 1}", `{"a":1}`},
		{"comment shebang after bom", "\xEF\xBB\xBF#!fx\n1", `1`},
		{"comment only", "; nothing", ``},
		{"multiple forms", "{:a 1}\n[1 2] \"s\" :k 42", "{\"a\":1}\n[1,2]\n\"s\"\n\"k\"\n42"},
		{"empty", ``, ``},
		{"whitespace only", " \n\t,", ``},
		{"bom", "\xEF\xBB\xBF{:a 1}", `{"a":1}`},
		{"no space between forms", `[[1][2]{:a"b"}"c""d"#{1}#inst"x"]`, `[[1],[2],{"a":"b"},"c","d",[1],"x"]`},
		{"deps.edn", `{:paths ["src" "resources"]
 :deps {org.clojure/clojure {:mvn/version "1.11.1"}
        io.github.x/y {:git/tag "v1" :git/sha "abc"}}
 :aliases {:test {:extra-paths ["test"]
                  :main-opts ["-m" "cognitect.test-runner"]}}}`,
			`{"paths":["src","resources"],"deps":{"org.clojure/clojure":{"mvn/version":"1.11.1"},"io.github.x/y":{"git/tag":"v1","git/sha":"abc"}},"aliases":{"test":{"extra-paths":["test"],"main-opts":["-m","cognitect.test-runner"]}}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToJSON([]byte(tt.in))
			require.NoError(t, err)
			require.Equal(t, tt.want, string(got))
		})
	}
}

func TestToJSON_Errors(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"unclosed vector", "[1 2", `edn: line 1, column 1: unclosed vector`},
		{"unclosed list", "(1 2", `edn: line 1, column 1: unclosed list`},
		{"unclosed set", "#{1 2", `edn: line 1, column 1: unclosed set`},
		{"unclosed map", "{:a 1", `edn: line 1, column 1: unclosed map`},
		{"unclosed map after key", "{:a", `edn: line 1, column 1: unclosed map`},
		{"unclosed string", `"abc`, `edn: line 1, column 1: unclosed string`},
		{"unclosed string at escape", `"abc\`, `edn: line 1, column 1: unclosed string`},
		{"map key without value", "{:a 1 :b}", `edn: line 1, column 9: map key without value`},
		{"wrong close", "[1 2)", `edn: line 1, column 5: unexpected ')'`},
		{"stray close", "]", `edn: line 1, column 1: unexpected ']'`},
		{"position", "{:a 1\n :b [1 2\n", `edn: line 2, column 5: unclosed vector`},
		{"position close", "{:a 1\n :b [1 2\n}", `edn: line 3, column 1: unexpected '}'`},
		{"column counts runes", "{:ключ ]}", `edn: line 1, column 8: unexpected ']'`},
		{"invalid number", "[1/2]", `edn: line 1, column 2: invalid number "1/2"`},
		{"invalid number hex", "0x1F", `edn: line 1, column 1: invalid number "0x1F"`},
		{"invalid number leading zero", "007", `edn: line 1, column 1: invalid number "007"`},
		{"invalid number trailing dot", "1.", `edn: line 1, column 1: invalid number "1."`},
		{"invalid number N on float", "[1.5N]", `edn: line 1, column 2: invalid number "1.5N"`},
		{"invalid number N on exponent", "1e3N", `edn: line 1, column 1: invalid number "1e3N"`},
		{"invalid escape", `"a\qb"`, `edn: line 1, column 3: invalid escape "\\q"`},
		{"invalid unicode escape", `"\u12"`, `edn: line 1, column 2: invalid escape "\\u12\""`},
		{"invalid character", `\foo`, `edn: line 1, column 1: invalid character "\\foo"`},
		{"invalid character escape", `\u12`, `edn: line 1, column 1: invalid character "\\u12"`},
		{"lone backslash", `\`, `edn: line 1, column 1: unexpected end of input`},
		{"lone colon", `:`, `edn: line 1, column 1: invalid keyword ":"`},
		{"quote", `'foo`, `edn: line 1, column 1: unexpected '\''`},
		{"quote in map", `{:a 'foo}`, `edn: line 1, column 5: unexpected '\''`},
		{"deref", `@foo`, `edn: line 1, column 1: unexpected '@'`},
		{"fn literal", `#(inc %)`, `edn: line 1, column 1: unexpected "#("`},
		{"regex", `#"a"`, `edn: line 1, column 1: unexpected "#\""`},
		{"shebang not at start", "[1 #!x 2]", `edn: line 1, column 4: unexpected "#!"`},
		{"shebang after space", " #!x", `edn: line 1, column 2: unexpected "#!"`},
		{"namespaced map", `#:a{:b 1}`, `edn: line 1, column 1: unexpected "#:"`},
		{"unknown symbolic value", `##Foo`, `edn: line 1, column 1: unknown symbolic value "##Foo"`},
		{"lone hash", `#`, `edn: line 1, column 1: unexpected end of input`},
		{"tag without value", `#inst`, `edn: line 1, column 6: unexpected end of input`},
		{"discard without form", `[1 #_]`, `edn: line 1, column 6: unexpected ']'`},
		{"discard at end", `#_`, `edn: line 1, column 3: unexpected end of input`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ToJSON([]byte(tt.in))
			require.EqualError(t, err, tt.want)
		})
	}
}
