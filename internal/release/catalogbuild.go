package release

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	maxZippedJSONBytes = 1 << 20
	englishLanguage    = "en"
)

// appZipInfo is what the catalog copies out of a zip: the app.json fields it publishes and the
// translated names from locales/<language>.json.
type appZipInfo struct {
	name    string
	names   map[string]string
	hidden  bool
	minHost string
}

// BuildCatalog describes every app-<id>-<version>.zip in appsDir. Each zip is checked against
// the zip rules first, so a catalog never lists a package a device would refuse.
func BuildCatalog(appsDir string, sequence int64, now time.Time) (*Catalog, error) {
	names, err := filepath.Glob(filepath.Join(appsDir, "app-*.zip"))
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("no app-<id>-<version>.zip in %s; run `make dist` first", appsDir)
	}
	catalog := &Catalog{V: CatalogSchema, Sequence: sequence, Issued: now.UTC().Format(time.RFC3339)}
	for _, zipPath := range names {
		app, err := describeCatalogApp(zipPath)
		if err != nil {
			return nil, err
		}
		catalog.Apps = append(catalog.Apps, app)
	}
	return catalog, catalog.Validate()
}

func describeCatalogApp(zipPath string) (CatalogApp, error) {
	pkg, err := describeAppZip(zipPath)
	if err != nil {
		return CatalogApp{}, err
	}
	info, err := readAppZipInfo(zipPath, pkg.ID)
	if err != nil {
		return CatalogApp{}, err
	}
	app := CatalogApp{ID: pkg.ID, Version: pkg.Version, Tag: AppTag(pkg.ID, pkg.Version), Zip: pkg.Zip,
		SHA256: pkg.SHA256, Size: pkg.Size, Name: info.name, Names: info.names, Hidden: info.hidden}
	if info.minHost != "" {
		app.Requires = &CatalogRequires{Drawer: ">=" + info.minHost}
	}
	return app, nil
}

func readAppZipInfo(zipPath, id string) (appZipInfo, error) {
	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		return appZipInfo{}, fmt.Errorf("cannot open app zip %s: %w", zipPath, err)
	}
	defer archive.Close()
	entries, err := CheckAppZip(&archive.Reader, id)
	if err != nil {
		return appZipInfo{}, fmt.Errorf("%s: %w", zipPath, err)
	}
	var info appZipInfo
	for _, entry := range entries {
		switch {
		case entry.Name == "app.json":
			err = info.readManifest(entry)
		case isLocaleFile(entry):
			err = info.readLocale(entry)
		}
		if err != nil {
			return appZipInfo{}, fmt.Errorf("%s: %s: %w", zipPath, entry.Name, err)
		}
	}
	return info, nil
}

func isLocaleFile(entry AppZipEntry) bool {
	return !entry.IsDir && strings.HasPrefix(entry.Name, "locales/") && strings.HasSuffix(entry.Name, ".json") &&
		strings.Count(entry.Name, "/") == 1
}

func (info *appZipInfo) readManifest(entry AppZipEntry) error {
	var header struct {
		Name    string `json:"name"`
		Hidden  bool   `json:"hidden"`
		MinHost string `json:"min_host"`
	}
	if err := decodeZippedJSON(entry, &header); err != nil {
		return err
	}
	info.name, info.hidden, info.minHost = header.Name, header.Hidden, header.MinHost
	return nil
}

// readLocale keeps the app's translated name; "name" is the key every app locale file uses.
func (info *appZipInfo) readLocale(entry AppZipEntry) error {
	var texts map[string]any
	if err := decodeZippedJSON(entry, &texts); err != nil {
		return err
	}
	language := strings.TrimSuffix(filepath.Base(entry.Name), ".json")
	name, isText := texts["name"].(string)
	if !isText || name == "" || language == englishLanguage {
		return nil // English is the catalog's own name
	}
	if info.names == nil {
		info.names = map[string]string{}
	}
	info.names[language] = name
	return nil
}

func decodeZippedJSON(entry AppZipEntry, target any) error {
	reader, err := entry.File.Open()
	if err != nil {
		return err
	}
	defer reader.Close()
	data, err := io.ReadAll(io.LimitReader(reader, maxZippedJSONBytes))
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

// VerifyCatalogZip checks one zip against its signed catalog entry: size, sha256, the id and
// version inside, and the zip safety rules.
func VerifyCatalogZip(app CatalogApp, zipPath string) error {
	info, err := os.Stat(zipPath)
	if err != nil {
		return fmt.Errorf("%s is listed in %s but cannot be read: %w", app.Zip, CatalogName, err)
	}
	if info.Size() != app.Size {
		return fmt.Errorf("%s has the wrong size (%s says %d bytes)", app.Zip, CatalogName, app.Size)
	}
	hash, err := HashFile(zipPath)
	if err != nil {
		return err
	}
	if hash != app.SHA256 {
		return fmt.Errorf("%s does not match the sha256 in %s", app.Zip, CatalogName)
	}
	id, version, err := identifyAppZip(zipPath)
	if err != nil {
		return err
	}
	if id != app.ID || version != app.Version {
		return fmt.Errorf("%s holds app %s %s but %s says %s %s", app.Zip, id, version, CatalogName, app.ID, app.Version)
	}
	_, err = readAppZipInfo(zipPath, id)
	return err
}
