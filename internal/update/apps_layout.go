package update

import (
	"os"
	"path/filepath"
	"strings"
)

// Folder names next to an app while it is replaced. They start with a dot, so the app scanner
// (manifest.LoadAll) never lists them as apps, even after a crash leaves them behind.
const (
	stagedSuffix = ".new"
	oldSuffix    = ".old"
	dataFolder   = "data"
)

func (layout Layout) appDir(id string) string { return filepath.Join(layout.appsDir(), id) }

func (layout Layout) stagedAppDir(id string) string {
	return filepath.Join(layout.appsDir(), "."+id+stagedSuffix)
}

func (layout Layout) oldAppDir(id string) string {
	return filepath.Join(layout.appsDir(), "."+id+oldSuffix)
}

// appRollbackDir holds the previous version of an app, without its data/.
func (layout Layout) appRollbackDir(id string) string {
	return filepath.Join(layout.workDir(), "app-rollback", id)
}

func (layout Layout) appsCatalogFile() string {
	return filepath.Join(layout.workDir(), "apps-catalog.json")
}
func (layout Layout) appsSequenceFile() string {
	return filepath.Join(layout.workDir(), "apps-sequence.json")
}
func (layout Layout) appsJobFile() string { return filepath.Join(layout.workDir(), "apps-job.json") }
func (layout Layout) appsCheckFile() string {
	return filepath.Join(layout.workDir(), "apps-check.json")
}

// appsRescanMarker is left by a finished job; the next screen read hands one rescan request
// to the mod and removes it.
func (layout Layout) appsRescanMarker() string {
	return filepath.Join(layout.workDir(), "apps-rescan")
}

// leftoverIDs lists the app ids that have a staged or old folder in the apps folder.
func (layout Layout) leftoverIDs() []string {
	entries, err := os.ReadDir(layout.appsDir())
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	var ids []string
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, ".") || !entry.IsDir() {
			continue
		}
		for _, suffix := range []string{stagedSuffix, oldSuffix} {
			if id, isLeftover := strings.CutSuffix(strings.TrimPrefix(name, "."), suffix); isLeftover && id != "" && !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
	}
	return ids
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

// holdsData is true when the folder has a top-level data/ entry, the user's files.
func holdsData(folder string) bool { return exists(filepath.Join(folder, dataFolder)) }
