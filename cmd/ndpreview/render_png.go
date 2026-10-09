package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"math"
	"os"

	xfont "golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

const (
	greyStep       = 17 // 255 / 15: the spacing of 16 grey levels
	pixelsPerPoint = 72 // with DPI 72 a font "point" is exactly one pixel
	collectionTag  = "ttcf"
)

// fontPaths lists candidate system fonts per family, first match wins. Vietnamese needs a font
// with Latin Extended Additional, which these cover.
var fontPaths = map[string][]string{
	familySans: {
		"/Library/Fonts/Arial Unicode.ttf",
		"/System/Library/Fonts/Supplemental/Arial.ttf",
		"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
		"/usr/share/fonts/truetype/noto/NotoSans-Regular.ttf",
		"/usr/share/fonts/dejavu/DejaVuSans.ttf",
		"C:\\Windows\\Fonts\\arial.ttf",
	},
	familySerif: {
		"/System/Library/Fonts/Supplemental/Georgia.ttf",
		"/System/Library/Fonts/Supplemental/Times New Roman.ttf",
		"/usr/share/fonts/truetype/dejavu/DejaVuSerif.ttf",
		"/usr/share/fonts/dejavu/DejaVuSerif.ttf",
	},
	familyMono: {
		"/System/Library/Fonts/Supplemental/Courier New.ttf",
		"/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf",
		"/usr/share/fonts/dejavu/DejaVuSansMono.ttf",
	},
}

// fontSet holds the parsed fonts and a cache of faces per family and size.
type fontSet struct {
	fonts map[string]*opentype.Font
	faces map[string]xfont.Face
}

// loadFonts parses the given font files; an empty path falls back to the system candidates.
// Serif and mono fall back to the sans font. With no usable font at all the set is empty and
// text is skipped (the caller reports it).
func loadFonts(sansPath, serifPath, monoPath string) (*fontSet, error) {
	set := &fontSet{fonts: map[string]*opentype.Font{}, faces: map[string]xfont.Face{}}
	explicit := map[string]string{familySans: sansPath, familySerif: serifPath, familyMono: monoPath}
	for _, family := range []string{familySans, familySerif, familyMono} {
		loaded, err := loadFamily(explicit[family], fontPaths[family])
		if err != nil {
			return nil, err
		}
		if loaded != nil {
			set.fonts[family] = loaded
		}
	}
	for _, family := range []string{familySerif, familyMono} {
		if set.fonts[family] == nil {
			set.fonts[family] = set.fonts[familySans]
		}
	}
	return set, nil
}

func loadFamily(explicit string, candidates []string) (*opentype.Font, error) {
	if explicit != "" {
		return parseFontFile(explicit)
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			if parsed, parseErr := parseFontFile(path); parseErr == nil {
				return parsed, nil
			}
		}
	}
	return nil, nil
}

func parseFontFile(path string) (*opentype.Font, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read the font file %s: %w", path, err)
	}
	if len(data) > 4 && string(data[:4]) == collectionTag {
		collection, collErr := opentype.ParseCollection(data)
		if collErr != nil {
			return nil, fmt.Errorf("cannot parse the font collection %s: %w", path, collErr)
		}
		return collection.Font(0)
	}
	parsed, err := opentype.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("cannot parse the font %s (use a .ttf or .otf file): %w", path, err)
	}
	return parsed, nil
}

func (f *fontSet) hasText() bool { return f != nil && f.fonts[familySans] != nil }

func (f *fontSet) face(family string, size int) (xfont.Face, error) {
	key := fmt.Sprintf("%s/%d", family, size)
	if face, cached := f.faces[key]; cached {
		return face, nil
	}
	face, err := opentype.NewFace(f.fonts[family], &opentype.FaceOptions{Size: float64(size), DPI: pixelsPerPoint, Hinting: xfont.HintingNone})
	if err != nil {
		return nil, fmt.Errorf("cannot create a %d px %s face: %w", size, family, err)
	}
	f.faces[key] = face
	return face, nil
}

// renderPNG draws a laid-out page as an 8-bit grey image of the full screen.
func renderPNG(page Page, fonts *fontSet, greyLevels int) (*image.Gray, error) {
	canvas := image.NewGray(image.Rect(0, 0, screenWidth, screenHeight))
	draw.Draw(canvas, canvas.Bounds(), image.NewUniform(color.Gray{Y: paperFull}), image.Point{}, draw.Src)
	for _, item := range page.Items {
		drawShape(canvas, item)
		if item.Shape == "text" && fonts.hasText() {
			if err := drawText(canvas, item, fonts); err != nil {
				return nil, err
			}
		}
	}
	if greyLevels == greyLevelsMax {
		quantize(canvas)
	}
	return canvas, nil
}

func drawShape(canvas *image.Gray, item Item) {
	switch item.Shape {
	case "rect":
		drawRect(canvas, item)
	case "ellipse":
		drawEllipse(canvas, item)
	}
}

func fillRect(canvas *image.Gray, rect image.Rectangle, level int) {
	draw.Draw(canvas, rect.Intersect(canvas.Bounds()), image.NewUniform(color.Gray{Y: uint8(level)}), image.Point{}, draw.Src)
}

func drawRect(canvas *image.Gray, item Item) {
	bounds := image.Rect(item.X, item.Y, item.X+item.W, item.Y+item.H)
	if item.Fill != noFill {
		fillRect(canvas, bounds, item.Fill)
	}
	border := item.Border
	if border == 0 {
		return
	}
	fillRect(canvas, image.Rect(bounds.Min.X, bounds.Min.Y, bounds.Max.X, bounds.Min.Y+border), item.Ink)
	fillRect(canvas, image.Rect(bounds.Min.X, bounds.Max.Y-border, bounds.Max.X, bounds.Max.Y), item.Ink)
	fillRect(canvas, image.Rect(bounds.Min.X, bounds.Min.Y, bounds.Min.X+border, bounds.Max.Y), item.Ink)
	fillRect(canvas, image.Rect(bounds.Max.X-border, bounds.Min.Y, bounds.Max.X, bounds.Max.Y), item.Ink)
}

func drawEllipse(canvas *image.Gray, item Item) {
	centerX, centerY := float64(item.X)+float64(item.W)/2, float64(item.Y)+float64(item.H)/2
	radiusX, radiusY := float64(item.W)/2, float64(item.H)/2
	for y := item.Y; y < item.Y+item.H; y++ {
		for x := item.X; x < item.X+item.W; x++ {
			distance := math.Hypot((float64(x)+0.5-centerX)/radiusX, (float64(y)+0.5-centerY)/radiusY)
			inner := 1 - float64(item.Border)/radiusX
			switch {
			case distance > 1 || !image.Pt(x, y).In(canvas.Bounds()):
			case item.Border > 0 && distance >= inner:
				canvas.SetGray(x, y, color.Gray{Y: uint8(item.Ink)})
			case item.Fill != noFill:
				canvas.SetGray(x, y, color.Gray{Y: uint8(item.Fill)})
			}
		}
	}
}

func drawText(canvas *image.Gray, item Item, fonts *fontSet) error {
	face, err := fonts.face(item.Family, item.Size)
	if err != nil {
		return err
	}
	metrics := face.Metrics()
	ascent, descent := metrics.Ascent.Ceil(), metrics.Descent.Ceil()
	clip := canvas.SubImage(image.Rect(item.X, item.Y, item.X+item.W, item.Y+item.H).Intersect(canvas.Bounds())).(*image.Gray)
	drawer := &xfont.Drawer{Dst: clip, Src: image.NewUniform(color.Gray{Y: uint8(item.Ink)}), Face: face}
	for index, line := range item.Lines {
		top := item.Y + index*item.LineH
		baseline := top + (item.LineH-(ascent+descent))/2 + ascent
		width := drawer.MeasureString(line).Ceil()
		x := item.X
		switch item.Align {
		case "center":
			x = item.X + (item.W-width)/2
		case "end":
			x = item.X + item.W - width
		}
		drawLine(drawer, line, x, baseline, item.Bold)
		if item.Underline {
			fillRect(canvas, image.Rect(x, baseline+4, x+width, baseline+6), item.Ink)
		}
	}
	return nil
}

// drawLine draws a line of text; bold is a second pass one pixel to the right.
func drawLine(drawer *xfont.Drawer, line string, x, baseline int, isBold bool) {
	passes := 1
	if isBold {
		passes = 2
	}
	for pass := 0; pass < passes; pass++ {
		drawer.Dot = fixed.P(x+pass, baseline)
		drawer.DrawString(line)
	}
}

// quantize snaps every pixel to one of 16 grey levels, as a 16-level e-ink panel shows it.
func quantize(canvas *image.Gray) {
	for index, value := range canvas.Pix {
		canvas.Pix[index] = uint8(math.Round(float64(value)/greyStep) * greyStep)
	}
}

func writePNG(out io.Writer, canvas *image.Gray) error {
	if err := png.Encode(out, canvas); err != nil {
		return fmt.Errorf("cannot encode the PNG: %w", err)
	}
	return nil
}
