package library

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
	Cancelled  bool     // the user stopped the run before the last book
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
	result.Cancelled = service.Cancelled()
	for index, book := range candidates {
		if service.Cancelled() {
			result.Cancelled = true
			break
		}
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
		service.refreshReport(unchangedPaths(result.Paths))
		service.queueRescan()
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
		if service.Cancelled() {
			return candidates
		}
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
	return rewriteEpubInPlace(absolute, languageRewrite)
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

// languageRewrite sets the language of a book to "vi" in the package document and in the
// language attributes of the content documents.
var languageRewrite = epubRewrite{
	packageDocument: setPackageLanguage,
	contentDocument: func(content []byte) []byte {
		return langAttribute.ReplaceAll(content, []byte("${1}"+targetLanguage+"${2}"))
	},
	verify: func(pkg packageXML) error {
		if languageBase(firstText(pkg.Languages)) != targetLanguage {
			return errors.New("rewritten package document does not say vi")
		}
		return nil
	},
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

// unchangedPaths maps each path to itself: a fix that does not rename its books.
func unchangedPaths(paths []string) map[string]string {
	changes := make(map[string]string, len(paths))
	for _, relative := range paths {
		changes[relative] = relative
	}
	return changes
}

// refreshReport re-analyzes the changed books and updates their rows, so the screen counts
// drop without a new scan. changes maps a book's old path to its path now.
func (service *Service) refreshReport(changes map[string]string) {
	report, ok, err := LoadReport(service.Paths)
	if err != nil || !ok {
		return
	}
	for index, book := range report.Books {
		current, isChanged := changes[book.Path]
		if !isChanged {
			continue
		}
		report.Books[index] = reanalyze(book, current)
	}
	report.Duplicates = FindDuplicates(report.Books)
	report.refreshCounts()
	if err := SaveReport(service.Paths, report); err != nil {
		service.Logger.Printf("cannot update report after a change: %v", err)
	}
}

// reanalyze reads a changed book again; a file that was hashed before is hashed again, so a
// group of identical copies stays identical only while it really is.
func reanalyze(previous Book, current string) Book {
	reread := Analyze(Found{Path: absoluteOf(current), Relative: current, Size: previous.Size})
	if info, err := os.Lstat(absoluteOf(current)); err == nil {
		reread.Size = info.Size()
	}
	if previous.SHA256 != "" {
		if sum, err := hashFile(absoluteOf(current)); err == nil {
			reread.SHA256 = sum
		}
	}
	return reread
}
