package main

import (
	"bytes"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

const repoRoot = "../.."

func decodeNRGBA(t *testing.T, data []byte) *image.NRGBA {
	t.Helper()
	decoded, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	result := image.NewNRGBA(decoded.Bounds())
	draw.Draw(result, result.Bounds(), decoded, decoded.Bounds().Min, draw.Src)
	return result
}

// A stale committed icon fails here; fix with `go run ./cmd/iconsgen -repo .`.
func TestCommittedIconsMatchGenerator(t *testing.T) {
	for _, entry := range icons {
		t.Run(entry.path, func(t *testing.T) {
			committed, err := os.ReadFile(filepath.Join(repoRoot, entry.path))
			if err != nil {
				t.Fatalf("%v; run: go run ./cmd/iconsgen -repo .", err)
			}
			want := render(entry).image()
			got := decodeNRGBA(t, committed)
			if got.Bounds() != want.Bounds() || !bytes.Equal(got.Pix, want.Pix) {
				t.Fatalf("%s is stale; run: go run ./cmd/iconsgen -repo .", entry.path)
			}
		})
	}
}

func TestIconsAreBlackOutlinesOnTransparent(t *testing.T) {
	for _, entry := range icons {
		t.Run(entry.path, func(t *testing.T) {
			rendered := render(entry).image()
			size := entry.style.size
			if rendered.Bounds().Dx() != size || rendered.Bounds().Dy() != size {
				t.Fatalf("size = %v, want %dx%d", rendered.Bounds(), size, size)
			}
			inked := 0
			for offset := 0; offset < len(rendered.Pix); offset += 4 {
				if rendered.Pix[offset] != 0 || rendered.Pix[offset+1] != 0 || rendered.Pix[offset+2] != 0 {
					t.Fatal("found a non-black pixel")
				}
				if rendered.Pix[offset+3] > 0 {
					inked++
				}
			}
			if inked < size*size/30 || inked > size*size/2 {
				t.Fatalf("%d inked pixels: icon looks empty or filled", inked)
			}
		})
	}
}

func TestIconsStayInsideTheMargin(t *testing.T) {
	for _, entry := range icons {
		t.Run(entry.path, func(t *testing.T) {
			rendered := render(entry).image()
			size, minimumMargin := entry.style.size, entry.style.margin
			for y := 0; y < size; y++ {
				for x := 0; x < size; x++ {
					isInMargin := x < minimumMargin || y < minimumMargin || x >= size-minimumMargin || y >= size-minimumMargin
					if isInMargin && rendered.NRGBAAt(x, y).A > 0 {
						t.Fatalf("ink at (%d,%d) inside the %d px margin", x, y, minimumMargin)
					}
				}
			}
		})
	}
}

func TestRenderIsDeterministic(t *testing.T) {
	for _, entry := range icons {
		first, err := encode(entry)
		if err != nil {
			t.Fatal(err)
		}
		second, _ := encode(entry)
		if !bytes.Equal(first, second) {
			t.Errorf("%s differs between runs", entry.path)
		}
	}
}

// The More-tab row icon sits next to Nickel's own (50x50, ~2 px strokes, 2 px padding).
func TestAppsRowIconMatchesNickelMoreIcons(t *testing.T) {
	for _, entry := range icons {
		if entry.path != "host/icons/apps.png" {
			continue
		}
		rendered := render(entry).image()
		if rendered.Bounds().Dx() != 50 || rendered.Bounds().Dy() != 50 {
			t.Fatalf("apps.png is %v, want 50x50 like Nickel's More icons", rendered.Bounds())
		}
		left, right := 50, -1
		for x := 0; x < 50; x++ {
			if rendered.NRGBAAt(x, 13).A > 128 { // a row through the top two squares
				left, right = min(left, x), max(right, x)
			}
		}
		if left > 5 || right < 44 {
			t.Fatalf("ink spans x %d..%d across the top squares, want about 3..46 like Nickel's icons", left, right)
		}
		return
	}
	t.Fatal("host/icons/apps.png is not generated")
}
