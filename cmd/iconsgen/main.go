// Command iconsgen rasterizes the vendored Tabler outline icons (third_party/tabler-icons/svg,
// MIT) into the PNGs the drawer loads: black strokes on a transparent background. Kobo's Qt 5.2
// may lack QtSvg, so the conversion happens here, not on the device. We never draw icons
// ourselves. Output is deterministic; a test fails when a committed PNG is stale.
// Usage: go run ./cmd/iconsgen -repo . [-sheet build/preview/icons-sheet.png]
package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

const svgFolder = "third_party/tabler-icons/svg"

// tablerGrid is the icon grid Tabler draws on; its stroke weight (1.5) is relative to it.
const tablerGrid = 24.0

func main() {
	repo := flag.String("repo", ".", "repository root the icon paths are relative to")
	sheet := flag.String("sheet", "", "also write a contact sheet PNG of all icons to this path")
	flag.Parse()
	if err := run(*repo, *sheet); err != nil {
		fmt.Fprintln(os.Stderr, "iconsgen:", err)
		os.Exit(1)
	}
}

func run(repo, sheetPath string) error {
	if err := writeAll(repo); err != nil {
		return err
	}
	if sheetPath == "" {
		return nil
	}
	return writeSheet(repo, sheetPath)
}

func writeAll(repo string) error {
	for _, entry := range icons {
		data, err := encode(repo, entry)
		if err != nil {
			return err
		}
		target := filepath.Join(repo, entry.path)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("cannot create folder for %s: %w", target, err)
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return fmt.Errorf("cannot write %s: %w", target, err)
		}
		fmt.Println("wrote", target)
	}
	return nil
}

func encode(repo string, entry icon) ([]byte, error) {
	rendered, err := render(repo, entry)
	if err != nil {
		return nil, err
	}
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, rendered); err != nil {
		return nil, fmt.Errorf("cannot encode %s: %w", entry.path, err)
	}
	return buffer.Bytes(), nil
}

var (
	strokeWidthAttribute = regexp.MustCompile(`stroke-width="[^"]*"`)
	viewBoxAttribute     = regexp.MustCompile(`viewBox="[^"]*"`)
)

// styledSvg pins the colour to black (oksvg has no currentColor), sets the stroke weight and
// crops the grid by entry.inset units per side. oksvg does not scale stroke widths with the
// target size, so the weight is given in output pixels.
func styledSvg(source []byte, entry icon) []byte {
	svg := bytes.ReplaceAll(source, []byte("currentColor"), []byte("#000000"))
	strokeWidth := strconv.FormatFloat(entry.strokePixels, 'f', 4, 64)
	svg = strokeWidthAttribute.ReplaceAll(svg, []byte(`stroke-width="`+strokeWidth+`"`))
	viewBox := fmt.Sprintf(`viewBox="%g %g %g %g"`, entry.inset, entry.inset, tablerGrid-2*entry.inset, tablerGrid-2*entry.inset)
	return viewBoxAttribute.ReplaceAll(svg, []byte(viewBox))
}

func render(repo string, entry icon) (*image.NRGBA, error) {
	source, err := os.ReadFile(filepath.Join(repo, svgFolder, entry.tabler+".svg"))
	if err != nil {
		return nil, fmt.Errorf("cannot read Tabler icon %q (vendor it into %s): %w", entry.tabler, svgFolder, err)
	}
	parsed, err := oksvg.ReadIconStream(bytes.NewReader(styledSvg(source, entry)), oksvg.StrictErrorMode)
	if err != nil {
		return nil, fmt.Errorf("cannot parse %s.svg: %w", entry.tabler, err)
	}
	parsed.SetTarget(0, 0, float64(entry.size), float64(entry.size))
	premultiplied := image.NewRGBA(image.Rect(0, 0, entry.size, entry.size))
	scanner := rasterx.NewScannerGV(entry.size, entry.size, premultiplied, premultiplied.Bounds())
	parsed.Draw(rasterx.NewDasher(entry.size, entry.size, scanner), 1)
	result := image.NewNRGBA(premultiplied.Bounds())
	draw.Draw(result, result.Bounds(), premultiplied, image.Point{}, draw.Src)
	return result, nil
}

// writeSheet lays every icon out on a white grid so a person can eyeball the set.
func writeSheet(repo, sheetPath string) error {
	const cell, columns = 112, 8
	rows := (len(icons) + columns - 1) / columns
	sheet := image.NewNRGBA(image.Rect(0, 0, columns*cell, rows*cell))
	draw.Draw(sheet, sheet.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	for index, entry := range icons {
		rendered, err := render(repo, entry)
		if err != nil {
			return err
		}
		origin := image.Pt(index%columns*cell+(cell-entry.size)/2, index/columns*cell+(cell-entry.size)/2)
		draw.Draw(sheet, rendered.Bounds().Add(origin), rendered, image.Point{}, draw.Over)
	}
	if err := os.MkdirAll(filepath.Dir(sheetPath), 0o755); err != nil {
		return err
	}
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, sheet); err != nil {
		return err
	}
	return os.WriteFile(sheetPath, buffer.Bytes(), 0o644)
}
