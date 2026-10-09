// Command iconsgen draws the shipped icons: black outline strokes on a transparent
// background, 96x96 for the drawer and 50x50 for the More-tab row (Nickel's icon size). The output is deterministic, so a test can compare it with the
// committed PNGs. Usage: go run ./cmd/iconsgen -repo .
package main

import (
	"bytes"
	"flag"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
)

func main() {
	repo := flag.String("repo", ".", "repository root the icon paths are relative to")
	flag.Parse()
	if err := writeAll(*repo); err != nil {
		fmt.Fprintln(os.Stderr, "iconsgen:", err)
		os.Exit(1)
	}
}

func writeAll(repo string) error {
	for _, entry := range icons {
		data, err := encode(entry)
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

func encode(entry icon) ([]byte, error) {
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, render(entry).image()); err != nil {
		return nil, fmt.Errorf("cannot encode %s: %w", entry.path, err)
	}
	return buffer.Bytes(), nil
}
