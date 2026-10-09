package library

import (
	"math"
	"path/filepath"
	"strings"

	"golang.org/x/text/unicode/norm"
)

// Book is one report row.
type Book struct {
	Path            string  `json:"path"`
	Size            int64   `json:"size"`
	Format          string  `json:"format"` // "epub" for EPUB and KEPUB, otherwise the extension
	Title           string  `json:"title,omitempty"`
	Author          string  `json:"author,omitempty"`
	Language        string  `json:"language,omitempty"`
	Vietnamese      bool    `json:"vietnamese,omitempty"`
	VietnameseShare float64 `json:"vietnamese_share,omitempty"`
	NotNFC          bool    `json:"not_nfc,omitempty"`
	SHA256          string  `json:"sha256,omitempty"` // only for files whose size matches another file
	Error           string  `json:"error,omitempty"`
}

// IsEpub is true for EPUB and KEPUB files.
func (book Book) IsEpub() bool { return book.Format == "epub" }

// NeedsLanguageFix is true for Vietnamese text tagged with another language.
func (book Book) NeedsLanguageFix() bool {
	return book.IsEpub() && book.Error == "" && book.Vietnamese && languageBase(book.Language) != "vi"
}

// Analyze reads one found file. A damaged EPUB gets Error set instead of failing the scan.
func Analyze(file Found) Book {
	extension := strings.ToLower(filepath.Ext(file.Path))
	book := Book{Path: file.Relative, Size: file.Size, Format: strings.TrimPrefix(extension, ".")}
	if extension != ".epub" {
		return book
	}
	info, err := ReadEpub(file.Path)
	if err != nil {
		book.Error = err.Error()
		return book
	}
	book.Title, book.Author, book.Language = info.Title, info.Author, info.Language
	share, isVietnamese := LooksVietnamese(info.Sample)
	book.Vietnamese = isVietnamese
	book.VietnameseShare = math.Round(share*1000) / 1000
	book.NotNFC = !isNFC(info.Sample, info.Title, info.Author)
	return book
}

func isNFC(texts ...string) bool {
	for _, text := range texts {
		if !norm.NFC.IsNormalString(text) {
			return false
		}
	}
	return true
}
