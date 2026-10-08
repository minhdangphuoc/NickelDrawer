package manifest

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Failure is an app folder whose manifest could not be loaded.
type Failure struct {
	AppID string // folder name; the manifest id may be unreadable
	Err   error
}

// Reason is a one-line explanation suitable for a small dialog.
func (f Failure) Reason() string {
	var validation *ValidationError
	if errors.As(f.Err, &validation) && len(validation.Problems) > 0 {
		return validation.Problems[0]
	}
	return strings.ReplaceAll(f.Err.Error(), "\n", " ")
}

// Scan is the result of loading every app folder under an apps directory.
type Scan struct {
	Apps     []App
	Failures []Failure
	Warnings []Warning // each prefixed with the folder name
}

// LoadAll loads every app folder in appsDir, skipping entries that start with '.' or '_' and non-directories.
func LoadAll(appsDir string) (Scan, error) {
	entries, err := os.ReadDir(appsDir)
	if err != nil {
		return Scan{}, fmt.Errorf("cannot read apps folder %s: %w", appsDir, err)
	}
	var scan Scan
	for _, entry := range entries {
		name := entry.Name()
		if !isAppFolder(entry, name) {
			continue
		}
		app, warnings, err := Load(filepath.Join(appsDir, name))
		for _, warning := range warnings {
			scan.Warnings = append(scan.Warnings, Warning(name+": "+string(warning)))
		}
		if err != nil {
			scan.Failures = append(scan.Failures, Failure{AppID: name, Err: err})
			continue
		}
		scan.Apps = append(scan.Apps, app)
	}
	sort.Slice(scan.Failures, func(i, j int) bool { return scan.Failures[i].AppID < scan.Failures[j].AppID })
	return scan, nil
}

func isAppFolder(entry os.DirEntry, name string) bool {
	if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
		return false
	}
	return entry.IsDir()
}
