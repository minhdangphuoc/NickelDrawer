package release

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// appZipTime is the fixed timestamp of every packed entry, so the same sources give the same bytes.
var appZipTime = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

// AppZipName is the file name of an app's zip, e.g. app-todo-1.0.1.zip.
func AppZipName(id, version string) string {
	return "app-" + id + "-" + version + ".zip"
}

// CheckAppEntry accepts only zip entries that merge onto the Kobo drive root as
// .adds/apps/<id>/..., and never a data/ folder: an app zip cannot reach anything but its own app.
func CheckAppEntry(id, entry string) error {
	if entry == "" || strings.Contains(entry, `\`) || path.Clean(entry) != entry || path.IsAbs(entry) {
		return fmt.Errorf("path must be clean, relative and use forward slashes")
	}
	prefix := appsPrefix + "/" + id + "/"
	if !strings.HasPrefix(entry, prefix) {
		return fmt.Errorf("path is outside %s", prefix)
	}
	for _, part := range strings.Split(strings.TrimPrefix(entry, prefix), "/") {
		if part == "data" {
			return fmt.Errorf("an app zip never carries a data/ folder")
		}
	}
	return nil
}

// PackApp writes <outDir>/app-<id>-<version>.zip from <treeRoot>/.adds/apps/<id>, the version
// read from its app.json. Entries are sorted and carry a fixed time and normalized modes.
func PackApp(treeRoot, id, outDir string) (zipPath string, err error) {
	appDir := filepath.Join(treeRoot, filepath.FromSlash(appsPrefix), id)
	version, err := readAppVersion(filepath.Join(appDir, "app.json"), id)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", fmt.Errorf("cannot create %s: %w", outDir, err)
	}
	zipPath = filepath.Join(outDir, AppZipName(id, version))
	output, err := os.Create(zipPath)
	if err != nil {
		return "", fmt.Errorf("cannot create %s: %w", zipPath, err)
	}
	err = writeAppZip(treeRoot, id, output)
	if closeErr := output.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		os.Remove(zipPath)
		return "", err
	}
	return zipPath, nil
}

func writeAppZip(treeRoot, id string, output io.Writer) error {
	entries, err := listAppEntries(treeRoot, id)
	if err != nil {
		return err
	}
	archive := zip.NewWriter(output)
	for _, entry := range entries {
		if err := addAppEntry(archive, treeRoot, entry); err != nil {
			return err
		}
	}
	return archive.Close()
}

func listAppEntries(treeRoot, id string) ([]string, error) {
	var entries []string
	appDir := filepath.Join(treeRoot, filepath.FromSlash(appsPrefix), id)
	err := filepath.WalkDir(appDir, func(fullPath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() || isJunk(entry.Name()) {
			return walkErr
		}
		relative, err := filepath.Rel(treeRoot, fullPath)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if err := CheckAppEntry(id, relative); err != nil {
			return fmt.Errorf("%s: %w", relative, err)
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("%s is not a regular file", relative)
		}
		entries = append(entries, relative)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("cannot list app %s: %w", id, err)
	}
	sort.Strings(entries)
	return entries, nil
}

func addAppEntry(archive *zip.Writer, treeRoot, entry string) error {
	fullPath := filepath.Join(treeRoot, filepath.FromSlash(entry))
	info, err := os.Stat(fullPath)
	if err != nil {
		return err
	}
	mode := fs.FileMode(0o644)
	if info.Mode().Perm()&0o111 != 0 {
		mode = 0o755
	}
	header := &zip.FileHeader{Name: entry, Method: zip.Deflate, Modified: appZipTime}
	header.SetMode(mode)
	writer, err := archive.CreateHeader(header)
	if err != nil {
		return err
	}
	source, err := os.Open(fullPath)
	if err != nil {
		return err
	}
	defer source.Close()
	if _, err := io.Copy(writer, source); err != nil {
		return fmt.Errorf("cannot add %s to the app zip: %w", entry, err)
	}
	return nil
}

func readAppVersion(appJSONPath, id string) (string, error) {
	data, err := os.ReadFile(appJSONPath)
	if err != nil {
		return "", fmt.Errorf("cannot read %s: %w", appJSONPath, err)
	}
	return parseAppHeader(data, id, appJSONPath)
}

func parseAppHeader(data []byte, id, source string) (string, error) {
	var header struct {
		ID      string `json:"id"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &header); err != nil || header.ID != id {
		return "", fmt.Errorf("%s must be JSON whose id equals the folder name %q", source, id)
	}
	if !ValidVersion(header.Version) {
		return "", fmt.Errorf("%s: version %q must look like MAJOR.MINOR.PATCH", source, header.Version)
	}
	return header.Version, nil
}

// AddAppPackages indexes every app-<id>-<version>.zip in dir: it checks that the zip holds only
// its own app and an app.json matching the file name, and records the zip's sha256 and size.
// A missing dir means a release without apps.
func (manifest *Manifest) AddAppPackages(dir string) error {
	names, err := filepath.Glob(filepath.Join(dir, "app-*.zip"))
	if err != nil {
		return err
	}
	for _, name := range names {
		pkg, err := describeAppZip(name)
		if err != nil {
			return err
		}
		manifest.AppPackages = append(manifest.AppPackages, pkg)
	}
	sort.Slice(manifest.AppPackages, func(i, j int) bool { return manifest.AppPackages[i].ID < manifest.AppPackages[j].ID })
	return manifest.Validate()
}

func describeAppZip(zipPath string) (AppPackage, error) {
	id, version, err := identifyAppZip(zipPath)
	if err != nil {
		return AppPackage{}, err
	}
	info, err := os.Stat(zipPath)
	if err != nil {
		return AppPackage{}, err
	}
	hash, err := HashFile(zipPath)
	if err != nil {
		return AppPackage{}, err
	}
	pkg := AppPackage{ID: id, Version: version, Zip: filepath.Base(zipPath), SHA256: hash, Size: info.Size()}
	if pkg.Zip != AppZipName(id, version) {
		return AppPackage{}, fmt.Errorf("%s holds app %s %s, so it must be named %s", pkg.Zip, id, version, AppZipName(id, version))
	}
	return pkg, nil
}

// identifyAppZip reads the id from the entries' common .adds/apps/<id>/ folder and the version
// from that app's app.json, and refuses any entry outside the folder.
func identifyAppZip(zipPath string) (id, version string, err error) {
	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", "", fmt.Errorf("cannot open app zip %s: %w", zipPath, err)
	}
	defer archive.Close()
	if id, err = appIDOfEntries(zipPath, archive.File); err != nil {
		return "", "", err
	}
	for _, entry := range archive.File {
		if entry.FileInfo().IsDir() {
			continue
		}
		if err := CheckAppEntry(id, entry.Name); err != nil {
			return "", "", fmt.Errorf("%s: %s: %w", zipPath, entry.Name, err)
		}
		if entry.Name == appsPrefix+"/"+id+"/app.json" {
			if version, err = readZippedVersion(entry, id); err != nil {
				return "", "", fmt.Errorf("%s: %w", zipPath, err)
			}
		}
	}
	if version == "" {
		return "", "", fmt.Errorf("%s has no %s/%s/app.json", zipPath, appsPrefix, id)
	}
	return id, version, nil
}

func appIDOfEntries(zipPath string, entries []*zip.File) (string, error) {
	for _, entry := range entries {
		parts := strings.Split(entry.Name, "/")
		if len(parts) >= 4 && parts[0]+"/"+parts[1] == appsPrefix && appIDPattern.MatchString(parts[2]) {
			return parts[2], nil
		}
	}
	return "", fmt.Errorf("%s holds no %s/<id>/ entries", zipPath, appsPrefix)
}

func readZippedVersion(entry *zip.File, id string) (string, error) {
	reader, err := entry.Open()
	if err != nil {
		return "", err
	}
	defer reader.Close()
	data, err := io.ReadAll(io.LimitReader(reader, 1<<20))
	if err != nil {
		return "", err
	}
	return parseAppHeader(data, id, entry.Name)
}

// VerifyAppPackages checks every indexed app zip in dir against the signed sha256 and size and
// re-checks its contents. Nothing is extracted or installed.
func VerifyAppPackages(manifest *Manifest, dir string) error {
	for _, app := range manifest.AppPackages {
		zipPath := filepath.Join(dir, app.Zip)
		info, err := os.Stat(zipPath)
		if err != nil {
			return fmt.Errorf("%s is listed in release.json but missing: %w", app.Zip, err)
		}
		if info.Size() != app.Size {
			return fmt.Errorf("%s has the wrong size (release.json says %d bytes)", app.Zip, app.Size)
		}
		hash, err := HashFile(zipPath)
		if err != nil {
			return err
		}
		if hash != app.SHA256 {
			return fmt.Errorf("%s does not match the sha256 in release.json", app.Zip)
		}
		id, version, err := identifyAppZip(zipPath)
		if err != nil {
			return err
		}
		if id != app.ID || version != app.Version {
			return fmt.Errorf("%s holds app %s %s but release.json says %s %s", app.Zip, id, version, app.ID, app.Version)
		}
	}
	return nil
}
