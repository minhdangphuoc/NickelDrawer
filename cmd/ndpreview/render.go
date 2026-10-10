package main

import (
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type renderOptions struct {
	AppDir, OutPath, Screen, Locale, StatePath, IconsDir string
	Page, Grey                                           int
	Value                                                *string // --value: the screen value (viewer picture id)
	Chrome                                               string  // --chrome shown|hidden (viewer)
	FontSans, FontSerif, FontMono                        string
}

func renderToFile(options renderOptions, stdout, stderr io.Writer) int {
	loaded, err := loadApp(options.AppDir, options.Locale)
	if err != nil {
		fmt.Fprintln(stderr, "ndpreview:", err)
		return exitInvalid
	}
	options.StatePath = localizedStatePath(options.StatePath, options.Locale)
	setIconDir(options.IconsDir)
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
	setSelected(vars, options.Value)
	screenID := firstNonEmpty(options.Screen, loaded.doc.Start)
	rc := renderContext{doc: loaded.doc, locale: options.Locale, vars: vars, value: options.Value,
		baseDir: stateDir(options.StatePath), chrome: options.Chrome}
	fonts, err := loadFonts(options.FontSans, options.FontSerif, options.FontMono)
	if err != nil {
		fmt.Fprintln(stderr, "ndpreview:", err)
		return exitFailure
	}
	useFonts(fonts, stderr)
	page, err := layoutScreen(rc, screenID, options.Page-1, Overlay{})
	if err != nil {
		fmt.Fprintln(stderr, "ndpreview:", err)
		return exitInvalid
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

// stateDir is the folder relative picture paths of the state resolve against.
func stateDir(statePath string) string {
	if statePath == "" {
		return ""
	}
	return filepath.Dir(statePath)
}

// setSelected stores the screen value in the variable "selected", like `appsctl drawer screen
// --value` does, when the app declares it.
func setSelected(vars map[string]any, value *string) {
	if _, declared := vars[selectedVarName]; declared && value != nil {
		vars[selectedVarName] = *value
	}
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

// localizedStatePath picks the English fixture next to a state file: for --locale en,
// preview-state.json becomes preview-state.en.json when that file exists (sample data in the
// language of the preview). Other locales, and files without an English sibling, are unchanged.
func localizedStatePath(path, locale string) string {
	if path == "" || locale != "en" || strings.HasSuffix(path, ".en.json") {
		return path
	}
	english := strings.TrimSuffix(path, ".json") + ".en.json"
	if _, err := os.Stat(english); err == nil {
		return english
	}
	return path
}
