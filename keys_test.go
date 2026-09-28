package main

import (
	"reflect"
	"strings"
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"
)

// keyCodes maps the names bubbletea prints for special keys to their codes.
var keyCodes = map[string]rune{
	"enter":     tea.KeyEnter,
	"esc":       tea.KeyEscape,
	"tab":       tea.KeyTab,
	"backspace": tea.KeyBackspace,
	"space":     tea.KeySpace,
	"up":        tea.KeyUp,
	"down":      tea.KeyDown,
	"left":      tea.KeyLeft,
	"right":     tea.KeyRight,
	"pgup":      tea.KeyPgUp,
	"pgdown":    tea.KeyPgDown,
	"home":      tea.KeyHome,
	"end":       tea.KeyEnd,
}

// press returns the key press bubbletea delivers for name: a character as
// typed ("q", "%") or a key name as String() prints it ("enter", "shift+tab",
// "ctrl+u"). Unknown names panic, so a typo fails the test that uses it.
func press(name string) tea.KeyPressMsg {
	var k tea.Key
	if r := []rune(name); len(r) == 1 {
		k.Code, k.Text = r[0], name
		return tea.KeyPressMsg(k)
	}
	parts := strings.Split(name, "+")
	for _, mod := range parts[:len(parts)-1] {
		switch mod {
		case "ctrl":
			k.Mod |= tea.ModCtrl
		case "alt":
			k.Mod |= tea.ModAlt
		case "shift":
			k.Mod |= tea.ModShift
		default:
			panic("unknown modifier in key " + name)
		}
	}
	last := parts[len(parts)-1]
	if code, ok := keyCodes[last]; ok {
		k.Code = code
		if last == "space" {
			k.Text = " "
		}
	} else if r := []rune(last); len(r) == 1 {
		k.Code = r[0]
		if k.Mod&(tea.ModCtrl|tea.ModAlt) == 0 {
			k.Text = last
		}
	} else {
		panic("unknown key " + name)
	}
	return tea.KeyPressMsg(k)
}

// typed returns one key press per character of s, as typed.
func typed(s string) []tea.KeyPressMsg {
	var out []tea.KeyPressMsg
	for _, r := range s {
		out = append(out, press(string(r)))
	}
	return out
}

// view returns the rendered screen.
func view(m *model) string {
	return m.View().Content
}

// TestKeymapNames checks that every key name in the keymap is one bubbletea
// v2 produces, so a binding cannot silently stop matching (v1 named the space
// bar " ", v2 names it "space").
func TestKeymapNames(t *testing.T) {
	bindings := map[string]key.Binding{
		"ctrlC": ctrlC, "escKey": escKey,
		"yankPath": yankPath, "yankKey": yankKey, "yankValueY": yankValueY,
		"yankValueV": yankValueV, "yankKeyValue": yankKeyValue,
		"showSizes": showSizes, "showLineNumbers": showLineNumbers,
	}
	v := reflect.ValueOf(keyMap)
	for i := 0; i < v.NumField(); i++ {
		bindings[v.Type().Field(i).Name] = v.Field(i).Interface().(key.Binding)
	}
	for field, b := range bindings {
		for _, name := range b.Keys() {
			require.Truef(t, key.Matches(press(name), b), "%s: %q", field, name)
			require.Equalf(t, name, press(name).String(), "%s: %q", field, name)
		}
	}
}
