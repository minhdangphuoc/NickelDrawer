package library

import (
	"encoding/json"
	"errors"
	"os"
	"path"
	"path/filepath"
	"regexp"

	"golang.org/x/text/unicode/norm"

	"koboapps/internal/i18n"
)

const phaseNormalize = "normalize"

// dcTextElement matches the text of dc:title and dc:creator in the package document.
var dcTextElement = regexp.MustCompile(
	`(?s)(<(?:[A-Za-z0-9_.-]+:)?(?:title|creator)(?:\s[^>]*)?>)([^<]*)(</(?:[A-Za-z0-9_.-]+:)?(?:title|creator)\s*>)`)

// nfcRewrite puts the title and the author of a book into Unicode NFC. Body text is left as
// it is: decomposed marks there display fine and rewriting every chapter would be a much
// bigger change than the library tool promises.
var nfcRewrite = epubRewrite{
	packageDocument: func(content []byte) ([]byte, error) {
		return dcTextElement.ReplaceAllFunc(content, func(element []byte) []byte {
			parts := dcTextElement.FindSubmatch(element)
			return append(append(append([]byte{}, parts[1]...), norm.NFC.Bytes(parts[2])...), parts[3]...)
		}), nil
	},
	verify: func(pkg packageXML) error {
		if !isNFC(append(append([]string{}, pkg.Titles...), pkg.Creators...)...) {
			return errors.New("rewritten package document still has a title or author that is not NFC")
		}
		return nil
	},
}

// NormalizeResult counts what a normalisation run did.
type NormalizeResult struct {
	Date       string
	Candidates int
	Fixed      int
	Failed     int
	Cancelled  bool
	Paths      []string // the new relative paths (with dry-run: the old ones)
}

// renameEntry records one rename of a normalisation run.
type renameEntry struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// NormalizeNFC renames files whose name is not NFC and rewrites the title and author of
// EPUBs that are not NFC. A book whose content changes is copied to data/backups/<date>/
// first, like a language fix; every rename is journaled, so Undo puts names back too.
// The caller holds the lock. With dryRun nothing is written.
func (service *Service) NormalizeNFC(dryRun bool) (NormalizeResult, error) {
	result := NormalizeResult{Date: today()}
	progress := &progressWriter{path: service.Paths.Progress(), phase: phaseNormalize}
	defer progress.clear()
	candidates := service.nfcCandidates(Discover(service.Logger).Files, progress)
	result.Candidates = len(candidates)
	result.Cancelled = service.Cancelled()
	changes := map[string]string{}
	for index, book := range candidates {
		if service.Cancelled() {
			result.Cancelled = true
			break
		}
		progress.update(index, len(candidates), book.Path)
		current, err := service.normalizeOne(book, dryRun, result.Date)
		if err != nil {
			result.Failed++
			service.Logger.Printf("normalize %s: %v", book.Path, err)
			continue
		}
		result.Fixed++
		result.Paths = append(result.Paths, current)
		changes[book.Path] = current
	}
	progress.update(len(candidates), len(candidates), "")
	service.Logger.Printf("normalize-nfc dryRun=%v: %d candidates, %d fixed, %d failed",
		dryRun, result.Candidates, result.Fixed, result.Failed)
	if !dryRun && result.Fixed > 0 {
		service.refreshReport(changes)
		service.queueRescan()
	}
	return result, nil
}

func (service *Service) nfcCandidates(files []Found, progress *progressWriter) []Book {
	var candidates []Book
	for index, file := range files {
		if service.Cancelled() {
			return candidates
		}
		progress.update(index, len(files), file.Relative)
		if book := Analyze(file); book.NeedsNFCFix() {
			candidates = append(candidates, book)
		}
	}
	return candidates
}

// normalizeOne fixes one book and returns its path afterwards. The target name is checked
// before anything is written, so a name clash leaves the book exactly as it was.
func (service *Service) normalizeOne(book Book, dryRun bool, date string) (current string, err error) {
	absolute, err := resolveInside(book.Path)
	if err != nil {
		return "", err
	}
	size, err := regularFile(absolute)
	if err != nil {
		return "", err
	}
	current = path.Join(path.Dir(book.Path), norm.NFC.String(path.Base(book.Path)))
	if err := ensureNameFree(absolute, current); err != nil {
		return "", err
	}
	if book.IsEpub() && book.MetaNotNFC && size > maxFixBytes {
		return "", errTooLarge{size: size}
	}
	if dryRun {
		return current, nil
	}
	if book.IsEpub() && book.MetaNotNFC {
		if err := service.rewriteMetadata(book, absolute, size, date); err != nil {
			return "", err
		}
	}
	if current == book.Path {
		return current, nil
	}
	if err := service.renameAndJournal(book.Path, current, absolute, date); err != nil {
		return "", err
	}
	return current, nil
}

// ensureNameFree refuses a rename onto a different existing file. A file system that treats
// the decomposed and the composed spelling as one name (APFS on a Mac) reports the file itself
// under the new name; that is not a clash.
func ensureNameFree(absolute, newRelative string) error {
	target := filepath.Join(filepath.Dir(absolute), path.Base(newRelative))
	if _, err := os.Lstat(target); err != nil {
		return nil
	}
	if isSameFile(absolute, target) {
		return nil
	}
	return i18n.NewError("msg.err_exists", "path", newRelative)
}

func (service *Service) rewriteMetadata(book Book, absolute string, size int64, date string) error {
	if err := ensureFreeSpace(filepath.Dir(absolute), 2*size+spaceMargin); err != nil {
		return err
	}
	if err := service.backup(book.Path, absolute, date); err != nil {
		return err
	}
	return rewriteEpubInPlace(absolute, nfcRewrite)
}

func (service *Service) renameAndJournal(from, to, absolute, date string) error {
	if err := os.MkdirAll(filepath.Join(service.Paths.Backups(), date), 0o755); err != nil {
		return i18n.NewError("msg.err_mkdir", "path", service.Paths.Backups()).WithCause(err)
	}
	target := filepath.Join(filepath.Dir(absolute), path.Base(to))
	if err := os.Rename(absolute, target); err != nil {
		return i18n.NewError("msg.err_write", "path", to).WithCause(err)
	}
	if err := service.appendRename(date, renameEntry{From: from, To: to}); err != nil {
		// The file is renamed but cannot be undone; put the old name back rather than lose track of it.
		if undoErr := os.Rename(target, absolute); undoErr != nil {
			return errors.Join(err, undoErr)
		}
		return err
	}
	return nil
}

func (service *Service) appendRename(date string, entry renameEntry) error {
	entries, err := service.readRenames(date)
	if err != nil {
		return err
	}
	content, err := json.Marshal(append(entries, entry))
	if err != nil {
		return err
	}
	return writeFileAtomic(service.Paths.RenameJournal(date), content)
}

func (service *Service) readRenames(date string) ([]renameEntry, error) {
	content, err := os.ReadFile(service.Paths.RenameJournal(date))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, i18n.NewError("msg.err_read", "path", service.Paths.RenameJournal(date)).WithCause(err)
	}
	var entries []renameEntry
	if err := json.Unmarshal(content, &entries); err != nil {
		return nil, i18n.NewError("msg.err_report_broken", "path", service.Paths.RenameJournal(date)).WithCause(err)
	}
	return entries, nil
}

// undoRenames reverses the renames of one day. copied holds the paths whose original content
// the backup copy brings back: for those the renamed file is deleted (the original name is
// back); the others are renamed back. Returns how many files got their old name back.
func (service *Service) undoRenames(entries []renameEntry, copied map[string]bool) (renamed int, firstFailure error) {
	for _, entry := range entries {
		count, err := service.undoRename(entry, copied[entry.From])
		renamed += count
		if err != nil {
			service.Logger.Printf("undo rename %s: %v", entry.To, err)
			if firstFailure == nil {
				firstFailure = err
			}
		}
	}
	return renamed, firstFailure
}

func (service *Service) undoRename(entry renameEntry, isRestoredFromBackup bool) (int, error) {
	toAbsolute, err := resolveInside(entry.To)
	if err != nil {
		return 0, err
	}
	if _, err := regularFile(toAbsolute); err != nil {
		return 0, nil // the renamed file is gone already; nothing to undo
	}
	fromAbsolute, err := resolveInside(entry.From)
	if err != nil {
		return 0, err
	}
	if isRestoredFromBackup {
		if isSameFile(toAbsolute, fromAbsolute) {
			return 0, nil // one name for both spellings: the restored original is the file
		}
		return 0, removeRegular(toAbsolute)
	}
	if err := ensureNameFree(toAbsolute, entry.From); err != nil {
		return 0, err
	}
	if err := os.Rename(toAbsolute, fromAbsolute); err != nil {
		return 0, i18n.NewError("msg.err_write", "path", entry.From).WithCause(err)
	}
	return 1, nil
}

func isSameFile(first, second string) bool {
	firstInfo, firstErr := os.Lstat(first)
	secondInfo, secondErr := os.Lstat(second)
	return firstErr == nil && secondErr == nil && os.SameFile(firstInfo, secondInfo)
}

func removeRegular(absolute string) error {
	if err := os.Remove(absolute); err != nil {
		return i18n.NewError("msg.err_remove", "path", absolute).WithCause(err)
	}
	return nil
}

func (service *Service) hasBackupCopy(date, relative string) bool {
	_, err := regularFile(filepath.Join(service.Paths.Backups(), date, filepath.FromSlash(relative)))
	return err == nil
}
