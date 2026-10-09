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
			want := render(entry.draw).image()
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
			rendered := render(entry.draw).image()
			if rendered.Bounds().Dx() != iconSize || rendered.Bounds().Dy() != iconSize {
				t.Fatalf("size = %v, want %dx%d", rendered.Bounds(), iconSize, iconSize)
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
			if inked < 300 || inked > iconSize*iconSize/2 {
				t.Fatalf("%d inked pixels: icon looks empty or filled", inked)
			}
		})
	}
}

func TestIconsStayInsideTheMargin(t *testing.T) {
	const minimumMargin = 8
	for _, entry := range icons {
		t.Run(entry.path, func(t *testing.T) {
			rendered := render(entry.draw).image()
			for y := 0; y < iconSize; y++ {
				for x := 0; x < iconSize; x++ {
					isInMargin := x < minimumMargin || y < minimumMargin || x >= iconSize-minimumMargin || y >= iconSize-minimumMargin
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
