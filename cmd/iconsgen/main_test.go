package main

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

const repoRoot = "../.."

// Antialiasing differs by a few levels between CPUs (arm64 vs amd64 float rounding), so
// committed icons are compared pixel by pixel with a small tolerance instead of byte for byte.
// A changed shape or stroke still moves far more pixels than this allows.
const (
	maxAlphaDifference     = 24   // per pixel, out of 255
	maxDifferingPixelShare = 0.02 // of all pixels may differ at all
)

// A stale committed icon fails here; fix with `go run ./cmd/iconsgen -repo .`.
func TestCommittedIconsMatchGenerator(t *testing.T) {
	for _, entry := range icons {
		t.Run(entry.path, func(t *testing.T) {
			file, err := os.Open(filepath.Join(repoRoot, entry.path))
			if err != nil {
				t.Fatalf("%v; run: go run ./cmd/iconsgen -repo .", err)
			}
			defer file.Close()
			committed, err := png.Decode(file)
			if err != nil {
				t.Fatal(err)
			}
			want, err := render(repoRoot, entry)
			if err != nil {
				t.Fatal(err)
			}
			if problem := iconDifference(committed, want); problem != "" {
				t.Fatalf("%s is stale (%s); run: go run ./cmd/iconsgen -repo .", entry.path, problem)
			}
		})
	}
}

// iconDifference describes why two renders of the same icon differ beyond rounding, or "".
func iconDifference(committed image.Image, want *image.NRGBA) string {
	if committed.Bounds() != want.Bounds() {
		return fmt.Sprintf("size %v, want %v", committed.Bounds(), want.Bounds())
	}
	differing := 0
	bounds := want.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, committedAlpha := committed.At(x, y).RGBA()
			wantAlpha := int(want.NRGBAAt(x, y).A)
			difference := int(committedAlpha>>8) - wantAlpha
			if difference < 0 {
				difference = -difference
			}
			if difference > maxAlphaDifference {
				return fmt.Sprintf("pixel %d,%d alpha differs by %d", x, y, difference)
			}
			if difference > 0 {
				differing++
			}
		}
	}
	if share := float64(differing) / float64(bounds.Dx()*bounds.Dy()); share > maxDifferingPixelShare {
		return fmt.Sprintf("%.1f%% of pixels differ", share*100)
	}
	return ""
}

func TestIconsAreBlackStrokesOnTransparent(t *testing.T) {
	for _, entry := range icons {
		t.Run(entry.path, func(t *testing.T) {
			rendered, err := render(repoRoot, entry)
			if err != nil {
				t.Fatal(err)
			}
			if rendered.Bounds().Dx() != entry.size || rendered.Bounds().Dy() != entry.size {
				t.Fatalf("size = %v, want %dx%d", rendered.Bounds(), entry.size, entry.size)
			}
			inked := 0
			for offset := 0; offset < len(rendered.Pix); offset += 4 {
				if rendered.Pix[offset+3] == 0 {
					continue
				}
				inked++
				if rendered.Pix[offset] != 0 || rendered.Pix[offset+1] != 0 || rendered.Pix[offset+2] != 0 {
					t.Fatal("found a non-black pixel")
				}
			}
			if inked < entry.size*entry.size/60 || inked > entry.size*entry.size/2 {
				t.Fatalf("%d inked pixels: icon looks empty or filled", inked)
			}
		})
	}
}

func TestRenderIsDeterministic(t *testing.T) {
	for _, entry := range icons {
		first, err := encode(repoRoot, entry)
		if err != nil {
			t.Fatal(err)
		}
		second, _ := encode(repoRoot, entry)
		if !bytes.Equal(first, second) {
			t.Errorf("%s differs between runs", entry.path)
		}
	}
}

// The More-tab row icon sits next to Nickel's own (50x50).
func TestAppsRowIconIsNickelMoreSize(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoRoot, "host/icons/apps.png"))
	if err != nil {
		t.Fatal(err)
	}
	config, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if config.Width != 50 || config.Height != 50 {
		t.Fatalf("apps.png is %dx%d, want 50x50 like Nickel's More icons", config.Width, config.Height)
	}
}
