package main

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

const repoRoot = "../.."

// A stale committed icon fails here; fix with `go run ./cmd/iconsgen -repo .`.
func TestCommittedIconsMatchGenerator(t *testing.T) {
	for _, entry := range icons {
		t.Run(entry.path, func(t *testing.T) {
			committed, err := os.ReadFile(filepath.Join(repoRoot, entry.path))
			if err != nil {
				t.Fatalf("%v; run: go run ./cmd/iconsgen -repo .", err)
			}
			want, err := encode(repoRoot, entry)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(committed, want) {
				t.Fatalf("%s is stale; run: go run ./cmd/iconsgen -repo .", entry.path)
			}
		})
	}
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
