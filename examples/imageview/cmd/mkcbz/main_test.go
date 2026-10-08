package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestPackFolderStoresImagesInNaturalOrder(t *testing.T) {
	folder := t.TempDir()
	for _, name := range []string{"page10.jpg", "page2.PNG", "page1.jpeg", "notes.txt", ".hidden.jpg"} {
		if err := os.WriteFile(filepath.Join(folder, name), []byte("data-"+name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	albumPath := filepath.Join(t.TempDir(), "album.cbz")

	if err := packFolder(folder, albumPath); err != nil {
		t.Fatalf("packFolder: %v", err)
	}

	reader, err := zip.OpenReader(albumPath)
	if err != nil {
		t.Fatalf("open album: %v", err)
	}
	defer reader.Close()
	var names []string
	for _, entry := range reader.File {
		names = append(names, entry.Name)
		if entry.Method != zip.Store {
			t.Errorf("%s is compressed; want stored", entry.Name)
		}
	}
	want := []string{"page1.jpeg", "page2.PNG", "page10.jpg"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("entries = %v, want %v", names, want)
	}
	if _, err := os.Stat(albumPath + ".tmp"); !os.IsNotExist(err) {
		t.Errorf("temp file left behind")
	}
}

func TestPackFolderWithoutImagesFails(t *testing.T) {
	folder := t.TempDir()
	if err := packFolder(folder, filepath.Join(t.TempDir(), "a.cbz")); err == nil {
		t.Fatal("want an error for a folder without images")
	}
}
