package inbox

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"koboapps/internal/storage"
)

// Reason is why a file was refused. The codes are stable: they are written to the journal and
// the host locale files have a message for each ("inbox.reason.<code>").
type Reason string

const (
	ReasonSymlink         Reason = "symlink"
	ReasonNotRegular      Reason = "not_regular"
	ReasonTypeNotAllowed  Reason = "type_not_allowed"
	ReasonTooLarge        Reason = "too_large"
	ReasonEmpty           Reason = "empty"
	ReasonContentMismatch Reason = "content_mismatch"
	// ReasonIncomplete marks the half-written ".tmp" file localgo leaves when a transfer is cut.
	// It is deleted instead of kept, because it can be as large as a whole book.
	ReasonIncomplete Reason = "incomplete"
)

// AllReasons lists every code, for tests and locale checks.
var AllReasons = []Reason{ReasonSymlink, ReasonNotRegular, ReasonTypeNotAllowed, ReasonTooLarge,
	ReasonEmpty, ReasonContentMismatch, ReasonIncomplete}

const (
	incompleteSuffix   = ".tmp"
	maxCollisionNumber = 1000
	hiddenPrefix       = "."
)

var (
	// ErrOutsideSource means the path is not directly inside the inbox or the old LocalSend folder.
	ErrOutsideSource = errors.New("file is not directly inside the inbox or the LocalSend folder")
	// ErrGone means another router process already took the file; nothing is left to do.
	ErrGone = errors.New("file is already gone (routed by another process)")
	// ErrIsFolder is returned for a path that is a folder.
	ErrIsFolder = errors.New("path is a folder, not a file")
)

// Outcome describes what happened to one file.
type Outcome struct {
	Name        string // original name as received
	Type        string // rule type, "" when unknown
	Destination storage.Folder
	StoredAs    string // final name in the destination
	Path        string // final path; empty when the file was deleted
	Reason      Reason // set when Destination is storage.Rejected
}

// IsRejected is true when the file went to rejected/ (or was deleted as incomplete).
func (outcome Outcome) IsRejected() bool { return outcome.Destination == storage.Rejected }

// Router applies the rules. Create it with NewRouter; paths come from internal/storage.
type Router struct {
	rules       Rules
	journalPath string
	now         func() time.Time
}

// NewRouter routes with the given rules and journals to journalPath.
func NewRouter(rules Rules, journalPath string) *Router {
	return &Router{rules: rules, journalPath: journalPath, now: time.Now}
}

// Route moves one file out of an allowed source folder. Any outcome, including a rejection,
// is journaled. An error with a non-zero Outcome means the file moved but the journal failed.
func (router *Router) Route(path, source string) (Outcome, error) {
	staged, err := router.locate(path)
	if err != nil {
		return Outcome{}, err
	}
	outcome, err := router.sortOut(staged)
	if err != nil {
		return Outcome{}, err
	}
	if err := AppendEntry(router.journalPath, router.entryFor(source, outcome)); err != nil {
		return outcome, err
	}
	return outcome, nil
}

// stagedFile is a file that was found directly inside an allowed source folder.
type stagedFile struct {
	path string // real path of the folder joined with the name
	name string
}

// allowedSources are the real paths of the folders files may be taken from.
func allowedSources() []string {
	var sources []string
	for _, folder := range []string{storage.Path(storage.Inbox), storage.LegacyLocalSendDir()} {
		if real, err := filepath.EvalSymlinks(folder); err == nil {
			sources = append(sources, real)
		}
	}
	return sources
}

func isAllowedSource(realFolder string) bool {
	for _, source := range allowedSources() {
		if source == realFolder {
			return true
		}
	}
	return false
}

// locate resolves the folder (never the file itself) and requires it to be an allowed source,
// so ".." tricks and links to other folders cannot reach files elsewhere.
func (router *Router) locate(path string) (stagedFile, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return stagedFile{}, fmt.Errorf("cannot resolve %s: %w", path, err)
	}
	name := filepath.Base(absolute)
	if name == "." || name == ".." || name == string(filepath.Separator) {
		return stagedFile{}, ErrOutsideSource
	}
	realFolder, err := filepath.EvalSymlinks(filepath.Dir(absolute))
	if err != nil || !isAllowedSource(realFolder) {
		return stagedFile{}, ErrOutsideSource
	}
	return stagedFile{path: filepath.Join(realFolder, name), name: name}, nil
}

func (router *Router) sortOut(staged stagedFile) (Outcome, error) {
	info, err := os.Lstat(staged.path)
	if errors.Is(err, fs.ErrNotExist) {
		return Outcome{}, ErrGone
	}
	if err != nil {
		return Outcome{}, fmt.Errorf("cannot inspect %s: %w", staged.name, err)
	}
	if info.IsDir() {
		return Outcome{}, ErrIsFolder
	}
	rule, extension, reason, err := router.judge(staged, info)
	if err != nil {
		return Outcome{}, err
	}
	if reason != "" {
		return router.reject(staged, rule.Type, reason)
	}
	return router.deliver(staged, rule, extension)
}

// judge decides whether the file may be delivered. A non-empty reason means rejection; the
// error is for I/O problems that must not be mistaken for a bad file.
func (router *Router) judge(staged stagedFile, info os.FileInfo) (Rule, string, Reason, error) {
	if info.Mode()&os.ModeSymlink != 0 {
		return Rule{}, "", ReasonSymlink, nil
	}
	if !info.Mode().IsRegular() {
		return Rule{}, "", ReasonNotRegular, nil
	}
	cleaned := cleanCharacters(staged.name)
	rule, extension, found := router.rules.Match(cleaned)
	if !found {
		if strings.HasSuffix(strings.ToLower(cleaned), incompleteSuffix) {
			return Rule{}, "", ReasonIncomplete, nil
		}
		return Rule{}, "", ReasonTypeNotAllowed, nil
	}
	if info.Size() == 0 {
		return rule, extension, ReasonEmpty, nil
	}
	if info.Size() > rule.MaxBytes {
		return rule, extension, ReasonTooLarge, nil
	}
	matches, err := fileMatches(staged.path, info, rule.Check)
	if errors.Is(err, fs.ErrNotExist) {
		return Rule{}, "", "", ErrGone // another router moved the file after our Lstat
	}
	if err != nil {
		return Rule{}, "", "", fmt.Errorf("cannot read %s: %w", staged.name, err)
	}
	if !matches {
		return rule, extension, ReasonContentMismatch, nil
	}
	return rule, extension, "", nil
}

// fileMatches opens the file without following a link planted after the check above, and
// confirms it is still the same regular file.
func fileMatches(path string, expected os.FileInfo, check string) (bool, error) {
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return false, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return false, err
	}
	if !opened.Mode().IsRegular() || !os.SameFile(expected, opened) {
		return false, nil
	}
	return contentMatches(check, file)
}

func (router *Router) deliver(staged stagedFile, rule Rule, extension string) (Outcome, error) {
	final := SanitizeName(staged.name, extension)
	destination, err := moveWithoutOverwrite(staged.path, storage.Path(rule.Destination), final, extension)
	if err != nil {
		return Outcome{}, err
	}
	return Outcome{Name: staged.name, Type: rule.Type, Destination: rule.Destination,
		StoredAs: filepath.Base(destination), Path: destination}, nil
}

func (router *Router) reject(staged stagedFile, ruleType string, reason Reason) (Outcome, error) {
	outcome := Outcome{Name: staged.name, Type: ruleType, Destination: storage.Rejected, Reason: reason}
	if reason == ReasonIncomplete {
		if err := os.Remove(staged.path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return Outcome{}, fmt.Errorf("cannot delete the incomplete upload %s: %w", staged.name, err)
		}
		return outcome, nil
	}
	final := SanitizeName(staged.name, "")
	destination, err := moveWithoutOverwrite(staged.path, storage.Path(storage.Rejected), final, filepath.Ext(final))
	if err != nil {
		return Outcome{}, err
	}
	if err := router.stampRejection(destination); err != nil {
		return Outcome{}, err
	}
	outcome.StoredAs, outcome.Path = filepath.Base(destination), destination
	return outcome, nil
}

// stampRejection sets the mtime to now because rename keeps the old one and purge counts the
// 7 days from the rejection. A link is left alone: Chtimes would change its target.
func (router *Router) stampRejection(path string) error {
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 {
		return err
	}
	if err := os.Chtimes(path, router.now(), router.now()); err != nil {
		return fmt.Errorf("cannot stamp %s: %w", path, err)
	}
	return nil
}

func (router *Router) entryFor(source string, outcome Outcome) Entry {
	return Entry{Time: router.now().UTC(), Source: source, Name: outcome.Name, Type: outcome.Type,
		Destination: string(outcome.Destination), StoredAs: outcome.StoredAs, Reason: outcome.Reason}
}

// moveWithoutOverwrite renames source into folder as name, or "stem (2).ext" when taken.
//
// The name is claimed first by creating an empty file with O_EXCL (two processes can never claim
// the same name), then the source is renamed over our own empty claim: one atomic step, and no
// hard links, which vfat does not support. If the rename fails the claim is removed again.
func moveWithoutOverwrite(source, folder, name, extension string) (string, error) {
	if err := ensureRealFolder(folder); err != nil {
		return "", err
	}
	claimed, err := claimName(folder, name, extension)
	if err != nil {
		return "", err
	}
	if err := os.Rename(source, claimed); err != nil {
		os.Remove(claimed)
		if errors.Is(err, fs.ErrNotExist) {
			return "", ErrGone
		}
		return "", fmt.Errorf("cannot move %s to %s: %w", filepath.Base(source), folder, err)
	}
	return claimed, nil
}

// ensureRealFolder creates the destination and refuses a link in its place.
func ensureRealFolder(folder string) error {
	if err := os.MkdirAll(folder, 0o755); err != nil {
		return fmt.Errorf("cannot create %s: %w", folder, err)
	}
	info, err := os.Lstat(folder)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("%s must be a real folder, not a link or a file; remove it and try again", folder)
	}
	return nil
}

func claimName(folder, name, extension string) (string, error) {
	stem := strings.TrimSuffix(name, extension)
	for number := 1; number <= maxCollisionNumber; number++ {
		candidate := name
		if number > 1 {
			candidate = fmt.Sprintf("%s (%d)%s", stem, number, extension)
		}
		path := filepath.Join(folder, candidate)
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("cannot create %s: %w", path, err)
		}
		if err := file.Close(); err != nil {
			os.Remove(path)
			return "", fmt.Errorf("cannot close %s: %w", path, err)
		}
		return path, nil
	}
	return "", fmt.Errorf("%d files named %q already exist in %s; clean the folder up", maxCollisionNumber, name, folder)
}

// SortReport is the result of routing a whole folder.
type SortReport struct {
	Routed   int
	Rejected int
	Failures []error // files that could not be handled; the others were still routed
}

// SortFolder routes every file directly inside folder, which must be the inbox or the old
// LocalSend folder. Hidden files and sub-folders are left alone.
func (router *Router) SortFolder(folder, source string) (SortReport, error) {
	realFolder, err := filepath.EvalSymlinks(folder)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return SortReport{}, nil
		}
		return SortReport{}, fmt.Errorf("cannot resolve %s: %w", folder, err)
	}
	if !isAllowedSource(realFolder) {
		return SortReport{}, ErrOutsideSource
	}
	entries, err := os.ReadDir(realFolder)
	if err != nil {
		return SortReport{}, fmt.Errorf("cannot list %s: %w", folder, err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	var report SortReport
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), hiddenPrefix) {
			continue
		}
		report.add(router.Route(filepath.Join(realFolder, entry.Name()), source))
	}
	return report, nil
}

func (report *SortReport) add(outcome Outcome, err error) {
	switch {
	case errors.Is(err, ErrGone):
		return
	case outcome.Destination == "":
		report.Failures = append(report.Failures, err)
		return
	case err != nil:
		report.Failures = append(report.Failures, err)
	}
	if outcome.IsRejected() {
		report.Rejected++
	} else {
		report.Routed++
	}
}

// PurgeRejected deletes files in rejected/ that were rejected more than maxAge ago. It only
// looks at direct children of the real rejected folder and never follows links.
func (router *Router) PurgeRejected(maxAge time.Duration) (int, error) {
	folder := storage.Path(storage.Rejected)
	info, err := os.Lstat(folder)
	if errors.Is(err, fs.ErrNotExist) {
		return 0, nil
	}
	if err != nil || !info.IsDir() {
		return 0, fmt.Errorf("%s must be a real folder, not a link or a file", folder)
	}
	entries, err := os.ReadDir(folder)
	if err != nil {
		return 0, fmt.Errorf("cannot list %s: %w", folder, err)
	}
	deleted := 0
	var failures []error
	for _, entry := range entries {
		stale, err := router.isOlderThan(folder, entry, maxAge)
		if err != nil {
			failures = append(failures, err)
			continue
		}
		if !stale {
			continue
		}
		if err := os.Remove(filepath.Join(folder, entry.Name())); err != nil {
			failures = append(failures, fmt.Errorf("cannot delete %s: %w", entry.Name(), err))
			continue
		}
		deleted++
	}
	return deleted, errors.Join(failures...)
}

// isOlderThan checks the entry itself (Lstat), so a link is judged by its own age and sub-folders are skipped.
func (router *Router) isOlderThan(folder string, entry fs.DirEntry, maxAge time.Duration) (bool, error) {
	if entry.IsDir() {
		return false, nil
	}
	info, err := os.Lstat(filepath.Join(folder, entry.Name()))
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("cannot inspect %s: %w", entry.Name(), err)
	}
	return router.now().Sub(info.ModTime()) > maxAge, nil
}
