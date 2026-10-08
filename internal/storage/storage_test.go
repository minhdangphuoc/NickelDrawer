package storage

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func pngBytes(t *testing.T) []byte {
	t.Helper()
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, image.NewGray(image.Rect(0, 0, 4, 4))); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func jpegBytes(t *testing.T) []byte {
	t.Helper()
	var buffer bytes.Buffer
	if err := jpeg.Encode(&buffer, image.NewGray(image.Rect(0, 0, 8, 8)), nil); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func write(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
}

func names(files []File) string {
	var list []string
	for _, file := range files {
		list = append(list, file.Name)
	}
	return strings.Join(list, ",")
}

func TestLayoutPathsFollowRoot(t *testing.T) {
	defer SetRoot("/x")()
	want := map[Folder]string{
		Books: "/x/Books", Pictures: "/x/Pictures", Documents: "/x/Documents",
		Comics: "/x/Comics", Wallpapers: "/x/Pictures/Wallpapers",
	}
	for folder, path := range want {
		if got := Path(folder); got != path {
			t.Errorf("Path(%s) = %q, want %q", folder, got, path)
		}
	}
	if Path("etc") != "" {
		t.Error("a name outside the layout must give no path")
	}
}

func TestDefaultRootIsOnboard(t *testing.T) {
	if Path(Wallpapers) != "/mnt/onboard/Pictures/Wallpapers" {
		t.Errorf("default wallpapers path = %q", Path(Wallpapers))
	}
}

func TestListImagesDetectsByMagicBytesNotName(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "a.txt"), pngBytes(t))
	write(t, filepath.Join(dir, "b.png"), []byte("not an image at all"))
	write(t, filepath.Join(dir, "c.dat"), jpegBytes(t))
	var logged []string
	files := ListImages(dir, 1<<20, func(format string, args ...any) { logged = append(logged, format) })
	if names(files) != "a.txt,c.dat" {
		t.Fatalf("files = %s", names(files))
	}
	if files[0].Kind != KindPNG || files[1].Kind != KindJPEG {
		t.Errorf("kinds = %s, %s", files[0].Kind, files[1].Kind)
	}
	if len(logged) != 0 {
		t.Errorf("non-images must be ignored silently, logged %v", logged)
	}
}

func TestListImagesReadsDirectChildrenOnly(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "top.png"), pngBytes(t))
	write(t, filepath.Join(dir, "sub", "deep.png"), pngBytes(t))
	write(t, filepath.Join(dir, ".hidden.png"), pngBytes(t))
	write(t, filepath.Join(dir, "_draft.png"), pngBytes(t))
	if got := names(ListImages(dir, 1<<20, nil)); got != "top.png" {
		t.Fatalf("files = %s", got)
	}
}

func TestListImagesRefusesSymlinkEscape(t *testing.T) {
	outside := t.TempDir()
	write(t, filepath.Join(outside, "secret.png"), pngBytes(t))
	dir := t.TempDir()
	write(t, filepath.Join(dir, "ok.png"), pngBytes(t))
	mustLink(t, filepath.Join(outside, "secret.png"), filepath.Join(dir, "escape.png"))
	mustLink(t, outside, filepath.Join(dir, "folder-link"))
	write(t, filepath.Join(dir, "sub", "inner.png"), pngBytes(t))
	mustLink(t, filepath.Join(dir, "sub", "inner.png"), filepath.Join(dir, "to-sub.png"))
	var logged []string
	files := ListImages(dir, 1<<20, func(format string, args ...any) { logged = append(logged, format) })
	if names(files) != "ok.png" {
		t.Fatalf("files = %s, want only ok.png", names(files))
	}
	if len(logged) != 3 {
		t.Errorf("escaping links must be logged, got %v", logged)
	}
}

func TestListImagesAllowsLinkToSiblingInsideFolder(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "real.png"), pngBytes(t))
	mustLink(t, filepath.Join(dir, "real.png"), filepath.Join(dir, "alias.png"))
	if got := names(ListImages(dir, 1<<20, nil)); got != "alias.png,real.png" {
		t.Fatalf("files = %s", got)
	}
}

func TestListImagesFollowsLinkedRootFolder(t *testing.T) {
	real := t.TempDir()
	write(t, filepath.Join(real, "a.png"), pngBytes(t))
	link := filepath.Join(t.TempDir(), "Pictures")
	mustLink(t, real, link)
	if got := names(ListImages(link, 1<<20, nil)); got != "a.png" {
		t.Fatalf("files = %s", got)
	}
}

func TestListImagesSkipsNonRegularFiles(t *testing.T) {
	dir := t.TempDir()
	if err := syscall.Mkfifo(filepath.Join(dir, "pipe.png"), 0o644); err != nil {
		t.Skip("no fifo support:", err)
	}
	if files := ListImages(dir, 1<<20, nil); len(files) != 0 {
		t.Fatalf("files = %s", names(files))
	}
}

func TestListImagesSizeCapIsLogged(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "big.png"), append(pngBytes(t), make([]byte, 100)...))
	var logged []string
	files := ListImages(dir, 50, func(format string, args ...any) { logged = append(logged, format) })
	if len(files) != 0 || len(logged) != 1 {
		t.Fatalf("files=%s logged=%v", names(files), logged)
	}
}

func TestListImagesMissingFolderIsEmpty(t *testing.T) {
	if files := ListImages(filepath.Join(t.TempDir(), "nope"), 1<<20, nil); files != nil {
		t.Fatal("want nothing")
	}
}

func TestReadLimitedEnforcesCap(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f")
	write(t, path, make([]byte, 100))
	if _, err := ReadLimited(path, 100); err != nil {
		t.Errorf("exact size must pass: %v", err)
	}
	if _, err := ReadLimited(path, 99); err == nil {
		t.Error("one byte over must fail")
	}
}

func mustLink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Skip("symlinks unavailable:", err)
	}
}
