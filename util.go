package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func clamp(v, low, high int) int {
	if high < low {
		low, high = high, low
	}
	return min(high, max(low, v))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func width(s string) int {
	return lipgloss.Width(s)
}

func accessor(path string, to interface{}) string {
	return fmt.Sprintf("%v[%v]", path, to)
}

func toLowerNumber(s string, style int) string {
	var out strings.Builder
    //  Subscript            ₀₁₂₃₄₅₆₇₈₉
    //  Math Monospace       𝟶𝟷𝟸𝟹𝟺𝟻𝟼𝟽𝟾𝟿
    //  Math bold digit      𝟎𝟏𝟐𝟑𝟒𝟓𝟔𝟕𝟖𝟗 
    //  Math bold sans serif 𝟬𝟭𝟮𝟯𝟰𝟱𝟲𝟳𝟴𝟵
    //  Math double struck   𝟘𝟙𝟚𝟛𝟜𝟝𝟞𝟠𝟡
    //  Ascii                0123456789
    //


	for _, r := range s {
		switch {
		case '0' <= r && r <= '9':
            switch style {
            case 2:
                out.WriteRune('\u2080' + (r - '\u0030')) // Subscript
            case 3:
                out.WriteRune('\U0001D7F6' + (r - '\u0030')) // Math Monospace
            case 4:
                out.WriteRune('\U0001D7CE' + (r - '\u0030')) // Math bold
            case 5:
                out.WriteRune('\U0001D7EC' + (r - '\u0030')) // Math bold - Sans serif
            case 6:
                out.WriteRune('\U0001D7D8' + (r - '\u0030')) // Math bold - Double struck
            default:
                out.WriteRune(r)
            }
		default:
			out.WriteRune(r)
		}
	}
    return out.String()
}
