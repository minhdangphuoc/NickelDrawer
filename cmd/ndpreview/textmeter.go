package main

import (
	"fmt"
	"io"

	xfont "golang.org/x/image/font"
)

// width measures text with the face the PNG renderer draws it with. Bold is drawn as a second
// pass one pixel to the right (drawLine), so it is one pixel wider.
func (f *fontSet) width(text string, face textFace) int {
	fontFace, err := f.face(face.family, face.size)
	if err != nil {
		return estimatedMeter{}.width(text, face)
	}
	width := xfont.MeasureString(fontFace, text).Ceil()
	if face.isBold {
		width++
	}
	return width
}

// useFonts makes layout measure with fonts; without a usable font it keeps the estimate, which
// is told to the user because lines then break where the real glyphs will not.
func useFonts(fonts *fontSet, stderr io.Writer) {
	if fonts.hasText() {
		textMeter = fonts
		return
	}
	textMeter = estimatedMeter{}
	fmt.Fprintln(stderr, "ndpreview: warning: no font found, line breaks are estimated; pass --font /path/to/font.ttf")
}
