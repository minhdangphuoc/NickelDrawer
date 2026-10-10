package wallpaper

import (
	"os"
	"path/filepath"
	"testing"
)

func mustBeDue(t *testing.T, r rig, want bool) {
	t.Helper()
	due, err := r.service.FirstImportDue()
	if err != nil {
		t.Fatal(err)
	}
	if due != want {
		t.Fatalf("FirstImportDue = %v, want %v", due, want)
	}
}

func hasMarker(r rig) bool { return hasFile(r.service.Paths.FirstImportMarker()) }

func TestFirstImportIsDueOnceForEmptyLibraryWithPictures(t *testing.T) {
	r := newRig(t)
	put(t, filepath.Join(r.pictures, "a.png"), gradientPNG(t, 50, 70))
	mustBeDue(t, r, true)
	if !hasMarker(r) {
		t.Fatal("marker not written when due")
	}
	mustBeDue(t, r, false)
}

func TestFirstImportIsNotDueWithoutPicturesAndWritesNoMarker(t *testing.T) {
	r := newRig(t)
	put(t, filepath.Join(r.pictures, "notes.txt"), []byte("text"))
	mustBeDue(t, r, false)
	if hasMarker(r) {
		t.Fatal("marker written although Pictures has no picture")
	}
	put(t, filepath.Join(r.pictures, "a.png"), gradientPNG(t, 50, 70))
	mustBeDue(t, r, true)
}

func TestFirstImportIsNotDueForNonEmptyLibrary(t *testing.T) {
	r := rigWithPictures(t)
	mustBeDue(t, r, false)
	if hasMarker(r) {
		t.Fatal("marker written for a non-empty library")
	}
}

func TestFirstImportIsNotDueAfterMarkerEvenWhenGalleryIsEmpty(t *testing.T) {
	r := newRig(t)
	put(t, filepath.Join(r.pictures, "a.png"), gradientPNG(t, 50, 70))
	put(t, r.service.Paths.FirstImportMarker(), []byte("1\n"))
	mustBeDue(t, r, false)
}

func TestFirstImportIgnoresPicturesTheImportWouldSkip(t *testing.T) {
	r := newRig(t)
	put(t, filepath.Join(r.pictures, ".hidden.png"), gradientPNG(t, 50, 70))
	put(t, filepath.Join(r.pictures, "sub", "deep.png"), gradientPNG(t, 50, 70))
	if err := os.Symlink(filepath.Join(r.pictures, "sub", "deep.png"), filepath.Join(r.pictures, "link.png")); err != nil {
		t.Skip("symlinks unavailable:", err)
	}
	mustBeDue(t, r, false)
}
