package uikit

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"koboapps/internal/i18n"
)

const (
	// FileName is the screen file of an app.
	FileName = "ui.json"
	// ScreensDirName holds extra screens, one file per screen: screens/<id>.json.
	ScreensDirName = "screens"
	localesDirName = "locales"
)

// HasUI reports whether the app folder has a ui.json.
func HasUI(appDir string) bool {
	info, err := os.Stat(filepath.Join(appDir, FileName))
	return err == nil && !info.IsDir()
}

// LoadDir reads and validates <appDir>/ui.json, <appDir>/screens/*.json and the app's locale
// files. The error is only for I/O problems; rule violations come back in the Report.
func LoadDir(appDir string, actionIDs []string) (*Document, Report, error) {
	data, err := os.ReadFile(filepath.Join(appDir, FileName))
	if err != nil {
		return nil, Report{}, fmt.Errorf("cannot read %s in %s: %w", FileName, appDir, err)
	}
	extra, err := readExtraScreens(filepath.Join(appDir, ScreensDirName))
	if err != nil {
		return nil, Report{}, err
	}
	catalogs, catalogIssues, err := readCatalogs(filepath.Join(appDir, localesDirName))
	if err != nil {
		return nil, Report{}, err
	}
	document, report := ParseAndValidate(data, extra, Env{ActionIDs: actionIDs, Catalogs: catalogs})
	report.Errors = append(catalogIssues, report.Errors...)
	if len(catalogIssues) > 0 {
		document = nil
	}
	return document, report, nil
}

func readExtraScreens(dir string) (map[string][]byte, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", dir, err)
	}
	screens := map[string][]byte{}
	for _, entry := range entries {
		id, isJSON := strings.CutSuffix(entry.Name(), ".json")
		if !isJSON || entry.IsDir() {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(dir, entry.Name()))
		if readErr != nil {
			return nil, fmt.Errorf("cannot read %s: %w", filepath.Join(dir, entry.Name()), readErr)
		}
		screens[id] = data
	}
	return screens, nil
}

// readCatalogs loads every locales/<locale>.json; a broken file becomes an Issue.
func readCatalogs(dir string) (map[string]i18n.Catalog, []Issue, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]i18n.Catalog{}, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("cannot read %s: %w", dir, err)
	}
	catalogs := map[string]i18n.Catalog{}
	var issues []Issue
	for _, entry := range entries {
		locale, isJSON := strings.CutSuffix(entry.Name(), ".json")
		if !isJSON || entry.IsDir() {
			continue
		}
		catalog, loadErr := i18n.LoadCatalog(filepath.Join(dir, entry.Name()))
		if loadErr != nil {
			issues = append(issues, Issue{CodeNotJSON, localesDirName + "/" + entry.Name(), errors.Unwrap(loadErr).Error()})
			continue
		}
		catalogs[locale] = catalog
	}
	return catalogs, issues, nil
}
