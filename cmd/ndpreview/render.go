package main

import (
	"fmt"
	"image"
	"io"
	"os"
)

type renderOptions struct {
	AppDir, OutPath, Screen, Locale, StatePath string
	Page, Grey                                 int
	FontSans, FontSerif, FontMono              string
}

func renderToFile(options renderOptions, stdout, stderr io.Writer) int {
	loaded, err := loadApp(options.AppDir, options.Locale)
	if err != nil {
		fmt.Fprintln(stderr, "ndpreview:", err)
		return exitInvalid
	}
	overrides, err := readStateFile(options.StatePath)
	if err != nil {
		fmt.Fprintln(stderr, "ndpreview:", err)
		return exitFailure
	}
	vars := map[string]any{}
	for name, value := range loaded.doc.Vars {
		vars[name] = value
	}
	for name, value := range overrides {
		vars[name] = value
	}
	screenID := firstNonEmpty(options.Screen, loaded.doc.Start)
	rc := renderContext{doc: loaded.doc, locale: options.Locale, vars: vars}
	page, err := layoutScreen(rc, screenID, options.Page-1, Overlay{})
	if err != nil {
		fmt.Fprintln(stderr, "ndpreview:", err)
		return exitInvalid
	}
	fonts, err := loadFonts(options.FontSans, options.FontSerif, options.FontMono)
	if err != nil {
		fmt.Fprintln(stderr, "ndpreview:", err)
		return exitFailure
	}
	if !fonts.hasText() {
		fmt.Fprintln(stderr, "ndpreview: warning: no font found, text is skipped; pass --font /path/to/font.ttf")
	}
	canvas, err := renderPNG(page, fonts, options.Grey)
	if err != nil {
		fmt.Fprintln(stderr, "ndpreview:", err)
		return exitFailure
	}
	if err := saveCanvas(options.OutPath, canvas); err != nil {
		fmt.Fprintln(stderr, "ndpreview:", err)
		return exitFailure
	}
	fmt.Fprintf(stdout, "wrote %s (%dx%d, screen %s, page %d/%d)\n", options.OutPath, screenWidth, screenHeight, screenID, page.Index+1, page.Count)
	return exitOK
}

func saveCanvas(path string, canvas *image.Gray) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("cannot create %s: %w", path, err)
	}
	if err := writePNG(file, canvas); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("cannot finish writing %s: %w", path, err)
	}
	return nil
}
