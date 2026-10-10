package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"koboapps/internal/i18n"
	"koboapps/internal/release"
)

const (
	keptBackups     = 2
	journalFileName = "journal.json"
	savedFilesDir   = "files"
)

type journalEntry struct {
	Path    string `json:"path"`
	Existed bool   `json:"existed"`
	Mode    string `json:"mode,omitempty"`
}

// journal records what an update changed so it can be undone: files that existed are
// copied to savedFilesDir, files that did not exist are listed to be deleted again.
type journal struct {
	FromVersion string         `json:"from_version"`
	ToVersion   string         `json:"to_version"`
	Created     string         `json:"created"`
	Entries     []journalEntry `json:"entries"`
}

func (layout Layout) backupDir(version string) string {
	return filepath.Join(layout.rollbackDir(), version)
}

// createBackup saves the current copy of every file the update will replace, then writes
// the journal. It must complete before the first file is replaced.
func (layout Layout) createBackup(fromVersion, toVersion string, changed []release.File) (string, error) {
	dir := layout.backupDir(fromVersion)
	if err := os.RemoveAll(dir); err != nil {
		return "", fmt.Errorf("cannot clear old backup %s: %w", dir, err)
	}
	entries := make([]journalEntry, 0, len(changed))
	for _, file := range changed {
		entry, err := layout.saveFile(dir, file.Path)
		if err != nil {
			return "", err
		}
		entries = append(entries, entry)
	}
	data, err := json.MarshalIndent(journal{
		FromVersion: fromVersion, ToVersion: toVersion,
		Created: time.Now().UTC().Format(time.RFC3339), Entries: entries,
	}, "", "  ")
	if err != nil {
		return "", err
	}
	if err := WriteFileAtomically(filepath.Join(dir, journalFileName), data, 0o644); err != nil {
		return "", err
	}
	return dir, nil
}

func (layout Layout) saveFile(backupDir, manifestPath string) (journalEntry, error) {
	source := layout.target(manifestPath)
	info, err := os.Stat(source)
	if os.IsNotExist(err) {
		return journalEntry{Path: manifestPath}, nil
	}
	if err != nil {
		return journalEntry{}, fmt.Errorf("cannot inspect %s: %w", source, err)
	}
	mode := info.Mode().Perm()
	saved := filepath.Join(backupDir, savedFilesDir, filepath.FromSlash(manifestPath))
	if err := copyFileAtomically(source, saved, mode); err != nil {
		return journalEntry{}, err
	}
	return journalEntry{Path: manifestPath, Existed: true, Mode: fmt.Sprintf("%04o", mode)}, nil
}

func readJournal(dir string) (journal, error) {
	data, err := os.ReadFile(filepath.Join(dir, journalFileName))
	if err != nil {
		return journal{}, fmt.Errorf("cannot read backup journal in %s: %w", dir, err)
	}
	var entries journal
	if err := json.Unmarshal(data, &entries); err != nil {
		return journal{}, fmt.Errorf("backup journal in %s is unreadable: %w", dir, err)
	}
	return entries, nil
}

// restore puts every journaled file back. It keeps going after a failure so as much as
// possible is restored, and returns all problems.
func (layout Layout) restore(dir string) error {
	entries, err := readJournal(dir)
	if err != nil {
		return err
	}
	var problems []error
	for _, entry := range entries.Entries {
		if err := layout.restoreEntry(dir, entry); err != nil {
			problems = append(problems, err)
		}
	}
	return errors.Join(problems...)
}

func (layout Layout) restoreEntry(dir string, entry journalEntry) error {
	if err := release.CheckPath(entry.Path); err != nil {
		return fmt.Errorf("backup journal lists a forbidden path %q: %w", entry.Path, err)
	}
	target := layout.target(entry.Path)
	if !entry.Existed {
		if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("cannot remove %s: %w", target, err)
		}
		return nil
	}
	file := release.File{Mode: entry.Mode}
	mode, err := file.ParseMode()
	if err != nil {
		return err
	}
	saved := filepath.Join(dir, savedFilesDir, filepath.FromSlash(entry.Path))
	return copyFileAtomically(saved, target, os.FileMode(mode))
}

// newestBackup returns the backup of the highest version, ok false when none exists.
func (layout Layout) newestBackup() (dir string, ok bool, err error) {
	versions, err := layout.backupVersions()
	if err != nil || len(versions) == 0 {
		return "", false, err
	}
	return layout.backupDir(versions[0]), true, nil
}

// backupVersions lists backup folder names that are versions, highest first.
func (layout Layout) backupVersions() ([]string, error) {
	entries, err := os.ReadDir(layout.rollbackDir())
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", layout.rollbackDir(), err)
	}
	var versions []string
	for _, entry := range entries {
		if entry.IsDir() && release.ValidVersion(entry.Name()) {
			versions = append(versions, entry.Name())
		}
	}
	sort.Slice(versions, func(i, j int) bool {
		order, _ := release.CompareVersions(versions[i], versions[j])
		return order > 0
	})
	return versions, nil
}

// pruneBackups keeps only the newest keptBackups backups.
func (layout Layout) pruneBackups() error {
	versions, err := layout.backupVersions()
	if err != nil || len(versions) <= keptBackups {
		return err
	}
	var problems []error
	for _, version := range versions[keptBackups:] {
		problems = append(problems, os.RemoveAll(layout.backupDir(version)))
	}
	return errors.Join(problems...)
}

// Rollback restores the newest backup and drops that backup. A mod archive that the update
// staged but Nickel has not installed yet is withdrawn.
func (updater *Updater) Rollback(ctx context.Context) (result Report, err error) {
	if err := updater.beginRun(); err != nil {
		return Report{}, err
	}
	defer func() { updater.finishRun(result, err) }()
	dir, found, err := updater.layout.newestBackup()
	if err != nil {
		return Report{}, err
	}
	if !found {
		return Report{}, i18n.NewError("update.no_rollback")
	}
	entries, err := readJournal(dir)
	if err != nil {
		return Report{}, i18n.NewError("update.restore_failed", "dir", dir).WithCause(err)
	}
	if err := updater.layout.dropStagedModIfSameAs(updater.layout.installedModPath()); err != nil {
		return Report{}, i18n.NewError("update.restore_failed", "dir", dir).WithCause(err)
	}
	if err := updater.layout.restore(dir); err != nil {
		return Report{}, i18n.NewError("update.restore_failed", "dir", dir).WithCause(err)
	}
	if err := os.RemoveAll(dir); err != nil {
		return Report{}, fmt.Errorf("restored, but cannot remove backup %s: %w", dir, err)
	}
	return report("update.rolled_back", "version", entries.FromVersion), nil
}
