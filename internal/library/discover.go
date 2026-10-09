package library

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"koboapps/internal/storage"
)

// bookExtensions are the files the scan lists; only EPUB and KEPUB are opened.
var bookExtensions = map[string]bool{
	".epub": true, ".mobi": true, ".pdf": true, ".txt": true,
	".html": true, ".htm": true, ".rtf": true, ".cbz": true, ".cbr": true,
}

// Found is a regular file that passed the safe-read rules.
type Found struct {
	Path     string // absolute
	Relative string // below the storage root, slash-separated
	Size     int64
}

// Discovery is the result of walking the library folders.
type Discovery struct {
	Files   []Found
	Skipped int // files over the size cap or beyond the file limit
}

// Discover walks the scanned folders. Safe-read rules, all enforced here:
//   - only the allowlisted folders (books, documents, comics) are entered;
//   - hidden names and names starting with '_' are skipped;
//   - symlinks are never followed, neither files nor folders, so nothing outside is read;
//   - only regular files with a known book extension, not above maxScanBytes;
//   - at most maxDepth levels and maxFiles files.
//
// A missing folder is normal. Nothing is modified.
func Discover(logger *Logger) Discovery {
	var found Discovery
	for _, folder := range scannedFolders {
		root := storage.Path(folder)
		if _, err := os.Stat(root); err != nil {
			continue
		}
		walkFolder(root, 0, logger, &found)
	}
	sort.Slice(found.Files, func(i, j int) bool { return found.Files[i].Relative < found.Files[j].Relative })
	return found
}

func walkFolder(dir string, depth int, logger *Logger, found *Discovery) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		logger.Printf("cannot list %s: %v", dir, err)
		return
	}
	for _, entry := range entries {
		name := entry.Name()
		if storage.IsSkippedName(name) {
			continue
		}
		full := filepath.Join(dir, name)
		switch mode := entry.Type(); {
		case mode&os.ModeSymlink != 0:
			logger.Printf("skip %s: symbolic link", full)
		case mode.IsDir():
			if depth < maxDepth {
				walkFolder(full, depth+1, logger, found)
			}
		case mode.IsRegular() && bookExtensions[strings.ToLower(filepath.Ext(name))]:
			addFile(full, logger, found)
		}
	}
}

func addFile(full string, logger *Logger, found *Discovery) {
	info, err := os.Lstat(full)
	if err != nil || !info.Mode().IsRegular() {
		logger.Printf("skip %s: not a regular file", full)
		return
	}
	relative, err := relativePath(full)
	switch {
	case err != nil:
		logger.Printf("skip %s: %v", full, err)
	case info.Size() > maxScanBytes:
		found.Skipped++
		logger.Printf("skip %s: larger than %d MB", relative, maxScanBytes>>20)
	case len(found.Files) >= maxFiles:
		found.Skipped++
	default:
		found.Files = append(found.Files, Found{Path: full, Relative: relative, Size: info.Size()})
	}
}
