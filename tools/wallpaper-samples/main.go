// Command wallpaper-samples makes 7 greyscale sample pictures and runs the real Wallpaper import on
// them, so previews (ndpreview) have thumbnails and pictures to show. Everything is written below
// --out and a temporary folder; nothing else is touched.
//
//	go run ./tools/wallpaper-samples --out build/preview/wallpaper-samples
package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"

	"koboapps/internal/wallpaper"
)

// libra2 is the screen the previews are made for.
var libra2 = wallpaper.Size{Width: 1264, Height: 1680}

// sample is one picture: file name (the extension picks PNG or JPEG), size and drawing.
type sample struct {
	name          string
	width, height int
	shade         func(x, y, width, height float64) float64 // 0 black .. 1 white
}

// samples are in import order, which fixes the ids 0001..0007.
var samples = []sample{
	{"nui-trang.png", 1600, 1200, mountains},
	{"bien-xanh.jpg", 1200, 1600, horizontalWaves},
	{"hoa-sen.png", 2000, 2000, rings},
	{"rung-thong.jpg", 1200, 1600, verticalStripes},
	{"pho-co.png", 1600, 1200, checkerboard},
	{"dong-lua.jpg", 2000, 2000, sunGradient},
	{"hoang-hon.png", 1600, 1200, setting},
}

func main() {
	out := flag.String("out", "", "folder that receives data/images, data/thumbs and index.json")
	flag.Parse()
	if *out == "" {
		fmt.Fprintln(os.Stderr, "wallpaper-samples: --out DIR is required, e.g. --out build/preview/wallpaper-samples")
		os.Exit(2)
	}
	if err := run(*out); err != nil {
		fmt.Fprintln(os.Stderr, "wallpaper-samples:", err)
		os.Exit(1)
	}
}

func run(out string) error {
	work, err := os.MkdirTemp("", "wallpaper-samples-")
	if err != nil {
		return fmt.Errorf("cannot create a temporary folder: %w", err)
	}
	defer os.RemoveAll(work)
	// Start clean so the ids are the same on every run.
	if err := os.RemoveAll(filepath.Join(out, "data")); err != nil {
		return fmt.Errorf("cannot clear %s: %w", filepath.Join(out, "data"), err)
	}
	pictures := filepath.Join(work, "Pictures")
	if err := os.MkdirAll(pictures, 0o755); err != nil {
		return fmt.Errorf("cannot create %s: %w", pictures, err)
	}
	service := wallpaper.NewService(wallpaper.Paths{AppDir: out}, libra2, nil)
	service.Sources = []string{pictures}
	service.ScreensaverDir = filepath.Join(work, "screensaver")
	for _, one := range samples {
		if err := writeSample(filepath.Join(pictures, one.name), one); err != nil {
			return err
		}
		if _, _, err := service.ImportAndApply(); err != nil {
			return fmt.Errorf("import after %s failed: %w", one.name, err)
		}
	}
	return nil
}

func writeSample(path string, one sample) error {
	picture := image.NewGray(image.Rect(0, 0, one.width, one.height))
	for y := 0; y < one.height; y++ {
		for x := 0; x < one.width; x++ {
			level := math.Min(math.Max(one.shade(float64(x), float64(y), float64(one.width), float64(one.height)), 0), 1)
			picture.SetGray(x, y, color.Gray{Y: uint8(level * 255)})
		}
	}
	var encoded bytes.Buffer
	var err error
	if filepath.Ext(path) == ".png" {
		err = png.Encode(&encoded, picture)
	} else {
		err = jpeg.Encode(&encoded, picture, &jpeg.Options{Quality: 85})
	}
	if err != nil {
		return fmt.Errorf("cannot encode %s: %w", path, err)
	}
	if err := os.WriteFile(path, encoded.Bytes(), 0o644); err != nil {
		return fmt.Errorf("cannot write %s: %w", path, err)
	}
	return nil
}

// mountains: a dark ridge line under a light sky.
func mountains(x, y, width, height float64) float64 {
	ridge := height*0.55 - 0.25*height*math.Abs(math.Sin(x/width*math.Pi*2.2)) - 0.1*height*math.Sin(x/width*math.Pi*7)
	if y > ridge {
		return 0.15 + 0.25*(y-ridge)/height
	}
	return 0.95 - 0.3*y/height
}

func horizontalWaves(x, y, width, height float64) float64 {
	return 0.5 + 0.4*math.Sin(y/height*math.Pi*14+math.Sin(x/width*math.Pi*3))
}

// rings: concentric circles around the centre.
func rings(x, y, width, height float64) float64 {
	distance := math.Hypot(x-width/2, y-height/2) / (width / 2)
	return 0.5 + 0.5*math.Cos(distance*math.Pi*9)
}

func verticalStripes(x, y, width, height float64) float64 {
	if int(x/width*24)%2 == 0 {
		return 0.15 + 0.3*y/height
	}
	return 0.9
}

func checkerboard(x, y, width, height float64) float64 {
	if (int(x/width*16)+int(y/height*12))%2 == 0 {
		return 0.1
	}
	return 0.9
}

// sunGradient: a bright disc on a vertical gradient.
func sunGradient(x, y, width, height float64) float64 {
	if math.Hypot(x-width*0.5, y-height*0.6) < width*0.18 {
		return 1
	}
	return 0.1 + 0.6*y/height
}

// setting: horizontal gradient with a dark ground strip.
func setting(x, y, width, height float64) float64 {
	if y > height*0.75 {
		return 0.08
	}
	return 0.25 + 0.7*x/width
}
