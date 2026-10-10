package storage

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ImageKind is a picture format found by magic bytes, never by file name.
type ImageKind string

const (
	KindPNG  ImageKind = "png"
	KindJPEG ImageKind = "jpeg"

	// MagicLength is how many leading bytes DetectKind needs.
	MagicLength = 8
)

var (
	pngMagic  = []byte("\x89PNG\r\n\x1a\n")
	jpegMagic = []byte{0xFF, 0xD8, 0xFF}
)

// DetectKind identifies PNG and JPEG from the first bytes of a file.
func DetectKind(header []byte) (ImageKind, bool) {
	switch {
	case bytes.HasPrefix(header, pngMagic):
		return KindPNG, true
	case bytes.HasPrefix(header, jpegMagic):
		return KindJPEG, true
	}
	return "", false
}

// Logf receives one line per skipped file; nil discards.
type Logf func(format string, args ...any)

func (logf Logf) printf(format string, args ...any) {
	if logf != nil {
		logf(format, args...)
	}
}

// File is a regular file that passed every check below.
type File struct {
	Path string // real path, symlinks resolved, a direct child of the real folder
	Name string
	Kind ImageKind
	Size int64
}

// ListImages returns the PNG and JPEG files that sit directly in dir, sorted by name.
//
// Safety rules (all enforced here, so callers cannot forget them):
//   - only direct children are read; sub-folders are never entered;
//   - names starting with '.' or '_' are skipped;
//   - every entry is resolved with EvalSymlinks and must be a direct child of the real dir,
//     so a link to another folder or file elsewhere is refused;
//   - only regular files (no devices, pipes, sockets);
//   - the type comes from magic bytes; other files are ignored silently;
//   - files over maxBytes are skipped with a logged reason.
//
// A missing dir is normal and yields nothing. Nothing is modified.
func ListImages(dir string, maxBytes int64, logf Logf) []File {
	var files []File
	eachImage(dir, maxBytes, logf, func(file File) bool {
		files = append(files, file)
		return true
	})
	return files
}

// FirstImage returns the first picture ListImages would return, reading only as many files as needed.
func FirstImage(dir string, maxBytes int64, logf Logf) (File, bool) {
	var first File
	var found bool
	eachImage(dir, maxBytes, logf, func(file File) bool {
		first, found = file, true
		return false
	})
	return first, found
}

// eachImage applies every ListImages rule and calls visit per picture in name order until it returns false.
func eachImage(dir string, maxBytes int64, logf Logf, visit func(File) bool) {
	realDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return
	}
	entries, err := os.ReadDir(realDir)
	if err != nil {
		logf.printf("cannot list %s: %v", dir, err)
		return
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		if IsSkippedName(entry.Name()) || entry.IsDir() {
			continue
		}
		file, ok := checkImage(dir, realDir, entry.Name(), maxBytes, logf)
		if ok && !visit(file) {
			return
		}
	}
}

// IsSkippedName is true for hidden names and names starting with '_'.
func IsSkippedName(name string) bool {
	return strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")
}

func checkImage(dir, realDir, name string, maxBytes int64, logf Logf) (File, bool) {
	realPath, err := filepath.EvalSymlinks(filepath.Join(realDir, name))
	if err != nil {
		logf.printf("skip %s: cannot resolve: %v", name, err)
		return File{}, false
	}
	if filepath.Dir(realPath) != realDir {
		logf.printf("skip %s: link points outside %s", name, dir)
		return File{}, false
	}
	info, err := os.Lstat(realPath)
	if err != nil || !info.Mode().IsRegular() {
		logf.printf("skip %s: not a regular file", name)
		return File{}, false
	}
	kind, err := sniffKind(realPath)
	if err != nil {
		logf.printf("skip %s: cannot read: %v", name, err)
		return File{}, false
	}
	if kind == "" {
		return File{}, false
	}
	if info.Size() > maxBytes {
		logf.printf("skip %s: larger than %d MB", name, maxBytes>>20)
		return File{}, false
	}
	return File{Path: realPath, Name: name, Kind: kind, Size: info.Size()}, true
}

// sniffKind reads only the first bytes; "" means not a PNG or JPEG.
func sniffKind(path string) (ImageKind, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	header := make([]byte, MagicLength)
	count, err := io.ReadFull(file, header)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return "", err
	}
	kind, _ := DetectKind(header[:count])
	return kind, nil
}

// ReadLimited loads a whole file but never more than maxBytes, and closes it at once so no
// handle stays open on /mnt/onboard.
func ReadLimited(path string, maxBytes int64) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > maxBytes {
		return nil, errors.New("file is larger than the size cap")
	}
	return content, nil
}
