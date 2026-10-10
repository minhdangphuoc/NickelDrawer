// Package release defines the signed release manifest (release.json) that NickelDrawer updates
// are made of: its format, the allowed file paths, signing and verification. See docs/UPDATES.md.
package release

import (
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	SchemaVersion = 1
	ManifestName  = "release.json"
	SignatureName = "release.json.sig"

	hostPrefix = ".adds/koboapps"
	appsPrefix = ".adds/apps"
)

var (
	semverPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-((?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*)(?:\.(?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*))*))?$`)
	appIDPattern  = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,30}$`)
	sha256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
	modePattern   = regexp.MustCompile(`^0[0-7]{3}$`)

	hostFolders = []string{"bin", "lib", "locales", "icons"}
	appFolders  = []string{"bin", "locales"}
)

// File is one file of the release, addressed relative to the onboard root.
type File struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
	Mode   string `json:"mode"` // octal permission bits, e.g. "0755"
}

// AppVersion records which version of an app the release ships.
type AppVersion struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

// Manifest is the content of release.json.
type Manifest struct {
	Schema     int          `json:"schema"`
	Version    string       `json:"version"`
	Created    string       `json:"created"`
	Files      []File       `json:"files"`
	Apps       []AppVersion `json:"apps"`
	MinAppsctl string       `json:"min_appsctl"`
}

// Parse decodes and validates release.json bytes.
func Parse(data []byte) (*Manifest, error) {
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("release.json is not valid JSON for this format: %w", err)
	}
	if err := manifest.Validate(); err != nil {
		return nil, err
	}
	return &manifest, nil
}

// Validate checks every rule of the format; it never touches the disk.
func (manifest *Manifest) Validate() error {
	if manifest.Schema != SchemaVersion {
		return fmt.Errorf("schema must be %d (got %d)", SchemaVersion, manifest.Schema)
	}
	if !ValidVersion(manifest.Version) {
		return fmt.Errorf("version %q must look like MAJOR.MINOR.PATCH", manifest.Version)
	}
	if !ValidVersion(manifest.MinAppsctl) {
		return fmt.Errorf("min_appsctl %q must look like MAJOR.MINOR.PATCH", manifest.MinAppsctl)
	}
	if _, err := time.Parse(time.RFC3339, manifest.Created); err != nil {
		return fmt.Errorf("created %q must be an RFC3339 timestamp: %w", manifest.Created, err)
	}
	if len(manifest.Files) == 0 {
		return fmt.Errorf("files must not be empty")
	}
	if err := manifest.validateFiles(); err != nil {
		return err
	}
	return manifest.validateApps()
}

func (manifest *Manifest) validateFiles() error {
	seen := map[string]bool{}
	for _, file := range manifest.Files {
		if err := file.validate(); err != nil {
			return fmt.Errorf("file %q: %w", file.Path, err)
		}
		if seen[file.Path] {
			return fmt.Errorf("file %q is listed twice", file.Path)
		}
		seen[file.Path] = true
	}
	return nil
}

func (manifest *Manifest) validateApps() error {
	seen := map[string]bool{}
	for _, app := range manifest.Apps {
		if !appIDPattern.MatchString(app.ID) || !ValidVersion(app.Version) {
			return fmt.Errorf("app %q version %q is not a valid id and MAJOR.MINOR.PATCH", app.ID, app.Version)
		}
		if seen[app.ID] {
			return fmt.Errorf("app %q is listed twice", app.ID)
		}
		seen[app.ID] = true
	}
	return nil
}

func (file File) validate() error {
	if err := CheckPath(file.Path); err != nil {
		return err
	}
	if !sha256Pattern.MatchString(file.SHA256) {
		return fmt.Errorf("sha256 must be 64 lowercase hex digits")
	}
	if file.Size < 0 {
		return fmt.Errorf("size must not be negative")
	}
	if !modePattern.MatchString(file.Mode) {
		return fmt.Errorf("mode %q must be octal like 0755", file.Mode)
	}
	return nil
}

// ParseMode converts the octal Mode text to permission bits.
func (file File) ParseMode() (uint32, error) {
	bits, err := strconv.ParseUint(file.Mode, 8, 32)
	if err != nil {
		return 0, fmt.Errorf("mode %q is not octal: %w", file.Mode, err)
	}
	return uint32(bits), nil
}

// CheckPath accepts only paths an update may write: host bin/lib/locales/icons and an app's
// app.json, icon.png, bin/ and locales/. Everything else, above all any data/ folder, is refused,
// so a signed manifest still cannot make an update touch user data.
func CheckPath(filePath string) error {
	if filePath == "" || strings.Contains(filePath, `\`) || path.Clean(filePath) != filePath || path.IsAbs(filePath) {
		return fmt.Errorf("path must be clean, relative and use forward slashes")
	}
	parts := strings.Split(filePath, "/")
	if isHostPath(parts) || isAppPath(parts) {
		return nil
	}
	return fmt.Errorf("path is outside the folders an update may write (%s/{bin,lib,locales,icons}, %s/<id>/{app.json,icon.png,bin,locales})", hostPrefix, appsPrefix)
}

func isHostPath(parts []string) bool {
	return len(parts) >= 4 && parts[0]+"/"+parts[1] == hostPrefix && contains(hostFolders, parts[2])
}

func isAppPath(parts []string) bool {
	if len(parts) < 4 || parts[0]+"/"+parts[1] != appsPrefix || !appIDPattern.MatchString(parts[2]) {
		return false
	}
	if len(parts) == 4 && (parts[3] == "app.json" || parts[3] == "icon.png") {
		return true
	}
	return len(parts) >= 5 && contains(appFolders, parts[3])
}

func contains(list []string, item string) bool {
	for _, candidate := range list {
		if candidate == item {
			return true
		}
	}
	return false
}

// ValidVersion reports whether text is MAJOR.MINOR.PATCH with an optional -prerelease
// (for example 1.0.0-beta.1).
func ValidVersion(text string) bool {
	_, err := parseVersion(text)
	return err == nil
}

// CompareVersions returns -1, 0 or 1 as a is older than, equal to or newer than b, using
// Semantic Versioning precedence: a pre-release is older than its release (1.0.0-beta.1 < 1.0.0).
func CompareVersions(a, b string) (int, error) {
	left, err := parseVersion(a)
	if err != nil {
		return 0, err
	}
	right, err := parseVersion(b)
	if err != nil {
		return 0, err
	}
	for i := range left.numbers {
		if left.numbers[i] != right.numbers[i] {
			return compareNumbers(left.numbers[i], right.numbers[i]), nil
		}
	}
	return comparePrerelease(left.prerelease, right.prerelease), nil
}

type version struct {
	numbers    [3]uint64
	prerelease []string
}

func compareNumbers(a, b uint64) int {
	if a < b {
		return -1
	}
	return 1
}

// comparePrerelease follows semver section 11: no pre-release is newest; numeric identifiers
// compare as numbers and sort below alphanumeric ones; a longer list wins when all else ties.
func comparePrerelease(a, b []string) int {
	switch {
	case len(a) == 0 && len(b) == 0:
		return 0
	case len(a) == 0:
		return 1
	case len(b) == 0:
		return -1
	}
	for i := 0; i < len(a) && i < len(b); i++ {
		if order := compareIdentifier(a[i], b[i]); order != 0 {
			return order
		}
	}
	if len(a) == len(b) {
		return 0
	}
	return compareNumbers(uint64(len(a)), uint64(len(b)))
}

func compareIdentifier(a, b string) int {
	if a == b {
		return 0
	}
	left, errLeft := strconv.ParseUint(a, 10, 64)
	right, errRight := strconv.ParseUint(b, 10, 64)
	switch {
	case errLeft == nil && errRight == nil:
		return compareNumbers(left, right)
	case errLeft == nil:
		return -1
	case errRight == nil:
		return 1
	case a < b:
		return -1
	}
	return 1
}

func parseVersion(text string) (version, error) {
	var parsed version
	match := semverPattern.FindStringSubmatch(text)
	if match == nil {
		return parsed, fmt.Errorf("version %q must look like MAJOR.MINOR.PATCH, optionally -prerelease (1.0.0-beta.1)", text)
	}
	for i := range parsed.numbers {
		number, err := strconv.ParseUint(match[i+1], 10, 32)
		if err != nil {
			return parsed, fmt.Errorf("version %q has a number that is too large: %w", text, err)
		}
		parsed.numbers[i] = number
	}
	if match[4] != "" {
		parsed.prerelease = strings.Split(match[4], ".")
	}
	return parsed, nil
}
