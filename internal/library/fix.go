package library

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	phaseFix = "fix"
	// maxRewriteBytes keeps a huge chapter from filling memory; larger entries are copied untouched.
	maxRewriteBytes = 8 << 20
	targetLanguage  = "vi"
)

var (
	// A language attribute on an element, "en" or "en-GB": the whitespace before it keeps
	// hreflang= and data-lang= out of reach.
	langAttribute = regexp.MustCompile(`(\s(?:xml:)?lang\s*=\s*["'])(?i:en)(?:[-_][A-Za-z0-9]+)*(["'])`)
	// The text of every dc:language element in the package document.
	languageElement = regexp.MustCompile(`(?s)(<(?:[A-Za-z0-9_.-]+:)?language(?:\s[^>]*)?>)[^<]*(</(?:[A-Za-z0-9_.-]+:)?language\s*>)`)
	metadataEnd     = regexp.MustCompile(`</(?:[A-Za-z0-9_.-]+:)?metadata\s*>`)
	titlePrefix     = regexp.MustCompile(`<([A-Za-z0-9_.-]+):title[\s>]`)

	rewritableExtensions = map[string]bool{".xhtml": true, ".html": true, ".htm": true, ".ncx": true}
)

// FixResult counts what a fix run did.
type FixResult struct {
	Date       string
	Candidates int
	Fixed      int
	Skipped    int // too large to rewrite
	Failed     int
	Paths      []string // relative paths that were (or with dry-run would be) fixed
}

// FixLanguage sets the language of Vietnamese EPUBs tagged otherwise to "vi". Each book is
// backed up first (data/backups/<date>/<same relative path>), rewritten into a temp file,
// re-read to prove it is a valid EPUB, and only then renamed over the original.
// The caller holds the lock. With dryRun nothing is written.
func (service *Service) FixLanguage(dryRun bool) (FixResult, error) {
	result := FixResult{Date: today()}
	discovery := Discover(service.Logger)
	progress := &progressWriter{path: service.Paths.Progress(), phase: phaseFix}
	defer progress.clear()
	candidates := service.languageCandidates(discovery.Files, progress)
	result.Candidates = len(candidates)
	for index, book := range candidates {
		progress.update(index, len(candidates), book.Path)
		if err := service.fixOne(book, dryRun, result.Date); err != nil {
			service.recordFailure(&result, book.Path, err)
			continue
		}
		result.Fixed++
		result.Paths = append(result.Paths, book.Path)
	}
	progress.update(len(candidates), len(candidates), "")
	service.Logger.Printf("fix-language dryRun=%v: %d candidates, %d fixed, %d skipped, %d failed",
		dryRun, result.Candidates, result.Fixed, result.Skipped, result.Failed)
	if !dryRun && result.Fixed > 0 {
		service.refreshReport(result.Paths)
	}
	return result, nil
}

func (service *Service) recordFailure(result *FixResult, relative string, err error) {
	var tooLarge errTooLarge
	if errors.As(err, &tooLarge) {
		result.Skipped++
	} else {
		result.Failed++
	}
	service.Logger.Printf("fix %s: %v", relative, err)
}

type errTooLarge struct{ size int64 }

func (err errTooLarge) Error() string {
	return fmt.Sprintf("larger than %d MB (%s)", maxFixBytes>>20, megabytes(err.size))
}

func (service *Service) languageCandidates(files []Found, progress *progressWriter) []Book {
	var candidates []Book
	for index, file := range files {
		progress.update(index, len(files), file.Relative)
		if book := Analyze(file); book.NeedsLanguageFix() {
			candidates = append(candidates, book)
		}
	}
	return candidates
}

func (service *Service) fixOne(book Book, dryRun bool, date string) error {
	absolute, err := resolveInside(book.Path)
	if err != nil {
		return err
	}
	size, err := regularFile(absolute)
	if err != nil {
		return err
	}
	if size > maxFixBytes {
		return errTooLarge{size: size}
	}
	if dryRun {
		return nil
	}
	if err := ensureFreeSpace(filepath.Dir(absolute), 2*size+spaceMargin); err != nil {
		return err
	}
	if err := service.backup(book.Path, absolute, date); err != nil {
		return err
	}
	return rewriteLanguage(absolute)
}

// backup copies the original to data/backups/<date>/<relative>. An existing backup of the same
// day is the true original, so it is kept.
func (service *Service) backup(relative, absolute, date string) error {
	target := filepath.Join(service.Paths.Backups(), date, filepath.FromSlash(relative))
	if _, err := os.Lstat(target); err == nil {
		return nil
	}
	return copyFileAtomic(absolute, target)
}

// rewriteLanguage replaces the file with a copy whose language is "vi", after checking the copy.
func rewriteLanguage(absolute string) error {
	return writeAtomic(absolute, func(temp *os.File) error {
		if err := writeFixedEpub(absolute, temp); err != nil {
			return err
		}
		if _, err := temp.Seek(0, io.SeekStart); err != nil {
			return err
		}
		return verifyFixedEpub(temp)
	})
}

// writeFixedEpub copies every entry untouched (raw, still compressed) except the package
// document and content documents, which get the new language. mimetype goes first, stored.
func writeFixedEpub(source string, output io.Writer) error {
	archive, err := zip.OpenReader(source)
	if err != nil {
		return err
	}
	defer archive.Close()
	info, err := ReadEpub(source)
	if err != nil {
		return err
	}
	writer := zip.NewWriter(output)
	for _, file := range orderedEntries(archive.File) {
		if err := copyEntry(writer, file, file.Name == info.OPFPath); err != nil {
			return fmt.Errorf("%s: %w", file.Name, err)
		}
	}
	return writer.Close()
}

// orderedEntries puts the stored mimetype entry first, as the EPUB container format requires.
func orderedEntries(files []*zip.File) []*zip.File {
	ordered := make([]*zip.File, 0, len(files))
	for _, file := range files {
		if file.Name == "mimetype" {
			ordered = append(ordered, file)
		}
	}
	for _, file := range files {
		if file.Name != "mimetype" {
			ordered = append(ordered, file)
		}
	}
	return ordered
}

func copyEntry(writer *zip.Writer, file *zip.File, isPackage bool) error {
	extension := strings.ToLower(path.Ext(file.Name))
	rewrite := isPackage || rewritableExtensions[extension]
	if !rewrite || file.UncompressedSize64 > maxRewriteBytes {
		return writer.Copy(file)
	}
	content, err := readFile(file)
	if err != nil {
		return err
	}
	var changed []byte
	if isPackage {
		changed, err = setPackageLanguage(content)
		if err != nil {
			return err
		}
	} else {
		changed = langAttribute.ReplaceAll(content, []byte("${1}"+targetLanguage+"${2}"))
	}
	if bytes.Equal(changed, content) {
		return writer.Copy(file)
	}
	header := zip.FileHeader{Name: file.Name, Method: file.Method, Modified: file.Modified}
	if file.Name == "mimetype" {
		header.Method = zip.Store
	}
	entry, err := writer.CreateHeader(&header)
	if err != nil {
		return err
	}
	_, err = entry.Write(changed)
	return err
}

func readFile(file *zip.File) ([]byte, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(io.LimitReader(reader, maxRewriteBytes+1))
}

// setPackageLanguage rewrites dc:language and any language attribute in the package document,
// or adds a dc:language element when the book has none.
func setPackageLanguage(content []byte) ([]byte, error) {
	changed := langAttribute.ReplaceAll(content, []byte("${1}"+targetLanguage+"${2}"))
	if languageElement.Match(changed) {
		return languageElement.ReplaceAll(changed, []byte("${1}"+targetLanguage+"${2}")), nil
	}
	prefix := titlePrefix.FindSubmatch(changed)
	end := metadataEnd.FindIndex(changed)
	if prefix == nil || end == nil {
		return nil, errors.New("package document has no dc:language and no place to add one")
	}
	element := fmt.Sprintf("<%s:language>%s</%s:language>", prefix[1], targetLanguage, prefix[1])
	return append(append(append([]byte{}, changed[:end[0]]...), element...), changed[end[0]:]...), nil
}

// verifyFixedEpub re-reads the finished copy: every entry must decompress, mimetype must be
// first and stored, and the language must now read "vi".
func verifyFixedEpub(file *os.File) error {
	info, err := file.Stat()
	if err != nil {
		return err
	}
	archive, err := zip.NewReader(file, info.Size())
	if err != nil {
		return fmt.Errorf("rewritten file is not a zip: %w", err)
	}
	if len(archive.File) == 0 || archive.File[0].Name != "mimetype" || archive.File[0].Method != zip.Store {
		return errors.New("rewritten file lacks a stored mimetype entry first")
	}
	for _, entry := range archive.File {
		if err := drain(entry); err != nil {
			return fmt.Errorf("rewritten entry %s: %w", entry.Name, err)
		}
	}
	return verifyLanguage(archive)
}

func drain(entry *zip.File) error {
	reader, err := entry.Open()
	if err != nil {
		return err
	}
	defer reader.Close()
	_, err = io.Copy(io.Discard, reader)
	return err
}

func verifyLanguage(archive *zip.Reader) error {
	entries := indexEntries(archive.File)
	opfPath, err := findOPFPath(entries)
	if err != nil {
		return err
	}
	var pkg packageXML
	if err := decodeEntry(entries, opfPath, maxOPFSize, &pkg); err != nil {
		return err
	}
	if languageBase(firstText(pkg.Languages)) != targetLanguage {
		return errors.New("rewritten package document does not say vi")
	}
	return nil
}

// refreshReport re-analyzes the fixed books and updates their rows, so the screen counts
// drop without a new scan.
func (service *Service) refreshReport(relatives []string) {
	report, ok, err := LoadReport(service.Paths)
	if err != nil || !ok {
		return
	}
	fixed := make(map[string]bool, len(relatives))
	for _, relative := range relatives {
		fixed[relative] = true
	}
	for index, book := range report.Books {
		if !fixed[book.Path] {
			continue
		}
		reread := Analyze(Found{Path: absoluteOf(book.Path), Relative: book.Path, Size: book.Size})
		reread.SHA256 = ""
		if info, err := os.Lstat(absoluteOf(book.Path)); err == nil {
			reread.Size = info.Size()
		}
		report.Books[index] = reread
	}
	report.Duplicates = FindDuplicates(report.Books)
	report.refreshCounts()
	if err := SaveReport(service.Paths, report); err != nil {
		service.Logger.Printf("cannot update report after fix: %v", err)
	}
}
