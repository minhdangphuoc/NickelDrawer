package wallpaper

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"math"

	"koboapps/internal/i18n"
	"koboapps/internal/storage"
)

const (
	grayLevels = 16
	// levelStep spreads 16 levels over 0..255 so white stays 255.
	levelStep = 255 / (grayLevels - 1)
)

// Convert decodes a PNG or JPEG, fits it to size and returns a 16-level dithered grayscale PNG.
// EXIF orientation is not applied: the picture is used as stored.
func Convert(content []byte, size Size, mode Mode) ([]byte, error) {
	if _, isImage := storage.DetectKind(content); !isImage {
		return nil, i18n.NewError("msg.err_not_image")
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(content))
	if err != nil {
		return nil, i18n.NewError("msg.err_decode").WithCause(err)
	}
	if config.Width <= 0 || config.Height <= 0 || config.Width*config.Height > maxPixels {
		return nil, i18n.NewError("msg.err_too_large", "width", itoa(config.Width), "height", itoa(config.Height))
	}
	source, err := decode(content)
	if err != nil {
		return nil, err
	}
	gray := toGray(source)
	source = nil // let the decoded picture go before scaling
	fitted := fitToScreen(gray, size, mode)
	gray = nil
	ditherFloydSteinberg(fitted)
	return encodePNG(fitted)
}

func decode(content []byte) (image.Image, error) {
	reader := bytes.NewReader(content)
	var decoded image.Image
	var err error
	if kind, _ := storage.DetectKind(content); kind == storage.KindPNG {
		decoded, err = png.Decode(reader)
	} else {
		decoded, err = jpeg.Decode(reader)
	}
	if err != nil {
		return nil, i18n.NewError("msg.err_decode").WithCause(err)
	}
	return decoded, nil
}

// toGray flattens transparency onto white, then takes luma, so the result is a plain gray plane.
func toGray(source image.Image) *image.Gray {
	bounds := source.Bounds()
	gray := image.NewGray(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			red, green, blue, alpha := source.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			// premultiplied: composite over white by adding the missing alpha.
			missing := 0xFFFF - alpha
			luma := (299*(red+missing) + 587*(green+missing) + 114*(blue+missing)) / 1000
			gray.Pix[y*gray.Stride+x] = uint8(luma >> 8)
		}
	}
	return gray
}

// fitToScreen scales gray into a size-sized image: fill covers and crops, fit letterboxes on white.
func fitToScreen(gray *image.Gray, size Size, mode Mode) *image.Gray {
	sourceWidth, sourceHeight := float64(gray.Rect.Dx()), float64(gray.Rect.Dy())
	scaleX, scaleY := float64(size.Width)/sourceWidth, float64(size.Height)/sourceHeight
	scale := math.Max(scaleX, scaleY)
	if mode == ModeFit {
		scale = math.Min(scaleX, scaleY)
	}
	drawnWidth := clampPositive(int(math.Round(sourceWidth*scale)), size.Width, mode)
	drawnHeight := clampPositive(int(math.Round(sourceHeight*scale)), size.Height, mode)
	canvas := image.NewGray(image.Rect(0, 0, size.Width, size.Height))
	for index := range canvas.Pix {
		canvas.Pix[index] = 255
	}
	offsetX, offsetY := (size.Width-drawnWidth)/2, (size.Height-drawnHeight)/2
	for y := 0; y < size.Height; y++ {
		for x := 0; x < size.Width; x++ {
			drawnX, drawnY := x-offsetX, y-offsetY
			if drawnX < 0 || drawnY < 0 || drawnX >= drawnWidth || drawnY >= drawnHeight {
				continue
			}
			canvas.Pix[y*canvas.Stride+x] = sample(gray, drawnX, drawnY, drawnWidth, drawnHeight)
		}
	}
	return canvas
}

// clampPositive keeps the drawn size at least 1 and, for fit, no larger than the screen.
func clampPositive(value, limit int, mode Mode) int {
	if value < 1 {
		return 1
	}
	if mode == ModeFit && value > limit {
		return limit
	}
	return value
}

// sample returns pixel (x, y) of the picture scaled to drawnWidth x drawnHeight: area average when
// shrinking, bilinear when enlarging. Fill mode crops the overflow because the canvas is smaller.
func sample(gray *image.Gray, x, y, drawnWidth, drawnHeight int) uint8 {
	stepX := float64(gray.Rect.Dx()) / float64(drawnWidth)
	stepY := float64(gray.Rect.Dy()) / float64(drawnHeight)
	if stepX > 1 || stepY > 1 {
		return areaAverage(gray, float64(x)*stepX, float64(y)*stepY, stepX, stepY)
	}
	return bilinear(gray, (float64(x)+0.5)*stepX-0.5, (float64(y)+0.5)*stepY-0.5)
}

func areaAverage(gray *image.Gray, left, top, width, height float64) uint8 {
	x0, y0 := int(left), int(top)
	x1, y1 := int(math.Ceil(left+width)), int(math.Ceil(top+height))
	x1, y1 = min(max(x1, x0+1), gray.Rect.Dx()), min(max(y1, y0+1), gray.Rect.Dy())
	sum, count := 0, 0
	for row := y0; row < y1; row++ {
		offset := row * gray.Stride
		for column := x0; column < x1; column++ {
			sum += int(gray.Pix[offset+column])
			count++
		}
	}
	return uint8(sum / count)
}

func bilinear(gray *image.Gray, x, y float64) uint8 {
	x = math.Min(math.Max(x, 0), float64(gray.Rect.Dx()-1))
	y = math.Min(math.Max(y, 0), float64(gray.Rect.Dy()-1))
	x0, y0 := int(x), int(y)
	x1, y1 := min(x0+1, gray.Rect.Dx()-1), min(y0+1, gray.Rect.Dy()-1)
	fractionX, fractionY := x-float64(x0), y-float64(y0)
	at := func(column, row int) float64 { return float64(gray.Pix[row*gray.Stride+column]) }
	top := at(x0, y0)*(1-fractionX) + at(x1, y0)*fractionX
	bottom := at(x0, y1)*(1-fractionX) + at(x1, y1)*fractionX
	return uint8(math.Round(top*(1-fractionY) + bottom*fractionY))
}

// ditherFloydSteinberg quantizes to 16 levels in place, pushing the error to neighbours so
// gradients survive on the e-ink panel. Rows alternate direction to avoid diagonal streaks.
func ditherFloydSteinberg(gray *image.Gray) {
	width, height := gray.Rect.Dx(), gray.Rect.Dy()
	current := make([]int, width+2)
	next := make([]int, width+2)
	for y := 0; y < height; y++ {
		leftToRight := y%2 == 0
		for step := 0; step < width; step++ {
			x := step
			direction := 1
			if !leftToRight {
				x, direction = width-1-step, -1
			}
			index := y*gray.Stride + x
			wanted := int(gray.Pix[index]) + current[x+1]
			chosen := nearestLevel(wanted)
			gray.Pix[index] = uint8(chosen)
			spread := wanted - chosen
			current[x+1+direction] += spread * 7 / 16
			next[x+1-direction] += spread * 3 / 16
			next[x+1] += spread * 5 / 16
			next[x+1+direction] += spread / 16
		}
		current, next = next, current
		for index := range next {
			next[index] = 0
		}
	}
}

func nearestLevel(value int) int {
	level := (value + levelStep/2) / levelStep
	return min(max(level, 0), grayLevels-1) * levelStep
}

func encodePNG(gray *image.Gray) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.BestCompression}
	if err := encoder.Encode(&buffer, gray); err != nil {
		return nil, i18n.NewError("msg.err_encode").WithCause(err)
	}
	return buffer.Bytes(), nil
}
