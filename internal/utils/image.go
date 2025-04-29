package utils

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"

	"github.com/charmbracelet/lipgloss"
	"github.com/nfnt/resize"
)

func DrawImage(r io.Reader, width, height int) (string, error) {
	img, _, err := image.Decode(r)
	if err != nil {
		return "", err
	}

	img = resize.Resize(uint(width), uint(height)*2, img, resize.Lanczos3)
	bounds := img.Bounds()

	var buffer bytes.Buffer
	for y := bounds.Min.Y + 1; y < bounds.Max.Y-1; y += 2 {
		for x := bounds.Min.X + 1; x < bounds.Max.X-1; x++ {
			r1, g1, b1, a1 := img.At(x, y+1).RGBA()
			r2, g2, b2, a2 := img.At(x, y).RGBA()

			// If both pixels are transparent, print a space.
			if a1 < 6553 && a2 < 6553 {
				buffer.WriteString(" ")
				continue
			}

			colorStr1 := fmt.Sprintf("#%02X%02X%02X", r1>>8, g1>>8, b1>>8)
			colorStr2 := fmt.Sprintf("#%02X%02X%02X", r2>>8, g2>>8, b2>>8)

			block := lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorStr1)).
				Background(lipgloss.Color(colorStr2)).
				Render("▄")

			buffer.WriteString(block)
		}
		buffer.WriteString("\n")
	}
	return buffer.String(), nil
}
