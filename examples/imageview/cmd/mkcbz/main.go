// Command mkcbz packs the png/jpg files of one folder into a comic-book zip (.cbz) for Kobo's
// reader. Images are already compressed, so entries are stored without compression: the Kobo's
// CPU stays idle and the album opens fast. BusyBox on the Kobo has no `zip` applet, hence this tool.
package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

func main() {
	albumPath := flag.String("out", "", "album file to write (required), e.g. Albums/Holiday.cbz")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: mkcbz -out album.cbz imagesFolder")
		flag.PrintDefaults()
	}
	flag.Parse()
	if *albumPath == "" || flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}
	if err := packFolder(flag.Arg(0), *albumPath); err != nil {
		fmt.Fprintln(os.Stderr, "mkcbz:", err)
		os.Exit(1)
	}
}

// packFolder writes the album to a temp file next to albumPath and renames it, so the reader
// never sees a half-written album and no file stays open afterwards.
func packFolder(folder, albumPath string) error {
	imageNames, err := listImageNames(folder)
	if err != nil {
		return err
	}
	if len(imageNames) == 0 {
		return fmt.Errorf("no png or jpg files in %s", folder)
	}
	tempPath := albumPath + ".tmp"
	if err := writeAlbum(folder, imageNames, tempPath); err != nil {
		os.Remove(tempPath)
		return err
	}
	if err := os.Rename(tempPath, albumPath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("cannot move the album into place: %w", err)
	}
	return nil
}

func listImageNames(folder string) ([]string, error) {
	entries, err := os.ReadDir(folder)
	if err != nil {
		return nil, fmt.Errorf("cannot read folder %s: %w", folder, err)
	}
	var names []string
	for _, entry := range entries {
		if entry.Type().IsRegular() && isImageName(entry.Name()) {
			names = append(names, entry.Name())
		}
	}
	sort.Slice(names, func(first, second int) bool { return naturalLess(names[first], names[second]) })
	return names, nil
}

func isImageName(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".png", ".jpg", ".jpeg":
		return !strings.HasPrefix(name, ".")
	}
	return false
}

func writeAlbum(folder string, imageNames []string, albumPath string) (err error) {
	file, err := os.Create(albumPath)
	if err != nil {
		return fmt.Errorf("cannot create %s: %w", albumPath, err)
	}
	defer func() {
		if closeErr := file.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
	}()
	archive := zip.NewWriter(file)
	for _, name := range imageNames {
		if err := addStoredEntry(archive, filepath.Join(folder, name), name); err != nil {
			return err
		}
	}
	return archive.Close()
}

func addStoredEntry(archive *zip.Writer, sourcePath, entryName string) error {
	info, err := os.Stat(sourcePath)
	if err != nil {
		return err
	}
	header := &zip.FileHeader{Name: entryName, Method: zip.Store, Modified: info.ModTime()}
	entry, err := archive.CreateHeader(header)
	if err != nil {
		return err
	}
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()
	if _, err := io.Copy(entry, source); err != nil {
		return fmt.Errorf("cannot add %s: %w", sourcePath, err)
	}
	return nil
}

// naturalLess orders "page2.jpg" before "page10.jpg" (digit runs compare as numbers), the order
// a person expects for pages.
func naturalLess(first, second string) bool {
	first, second = strings.ToLower(first), strings.ToLower(second)
	for first != "" && second != "" {
		firstRun, firstRest := splitRun(first)
		secondRun, secondRest := splitRun(second)
		if firstRun != secondRun {
			return runLess(firstRun, secondRun)
		}
		first, second = firstRest, secondRest
	}
	return len(first) < len(second)
}

func splitRun(text string) (run, rest string) {
	isDigitRun := unicode.IsDigit(rune(text[0]))
	end := 1
	for end < len(text) && unicode.IsDigit(rune(text[end])) == isDigitRun {
		end++
	}
	return text[:end], text[end:]
}

func runLess(first, second string) bool {
	if unicode.IsDigit(rune(first[0])) && unicode.IsDigit(rune(second[0])) {
		firstDigits, secondDigits := strings.TrimLeft(first, "0"), strings.TrimLeft(second, "0")
		if len(firstDigits) != len(secondDigits) {
			return len(firstDigits) < len(secondDigits)
		}
		if firstDigits != secondDigits {
			return firstDigits < secondDigits
		}
	}
	return first < second
}
