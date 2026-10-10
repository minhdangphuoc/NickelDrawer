package release

import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// The app catalog (apps.json) lists the project's own apps, each released on its own GitHub
// release. It is signed with the project release key like release.json, but over a prefixed
// message, so a drawer signature can never pass as a catalog or the other way round.
const (
	CatalogName          = "apps.json"
	CatalogSignatureName = "apps.json.sig"
	CatalogSchema        = 1
	// CatalogTag is the fixed, rolling release that holds the current catalog.
	CatalogTag = "apps"
	// CatalogSignaturePrefix is prepended to the exact catalog bytes before signing.
	CatalogSignaturePrefix = "nickeldrawer-apps-v1\n"
	// MaxCatalogBytes caps what a device reads before it has verified anything.
	MaxCatalogBytes = 64 << 10

	maxCatalogApps  = 200
	maxAppNameRunes = 60
	// MaxAppZipBytes is the largest app zip a device downloads (32 MiB).
	MaxAppZipBytes = 32 << 20
)

var catalogLanguagePattern = regexp.MustCompile(`^[a-z]{2,3}([_-][A-Za-z0-9]{2,8})?$`)

// CatalogRequires states what an app needs from the drawer.
type CatalogRequires struct {
	// Drawer is a constraint ">=X.Y.Z"; empty means any drawer.
	Drawer string `json:"drawer,omitempty"`
}

// CatalogApp is the signed entry of one app version.
type CatalogApp struct {
	ID      string            `json:"id"`
	Version string            `json:"version"`
	Tag     string            `json:"tag"`
	Zip     string            `json:"zip"`
	SHA256  string            `json:"sha256"`
	Size    int64             `json:"size"`
	Name    string            `json:"name"`
	Names   map[string]string `json:"names,omitempty"`
	Hidden  bool              `json:"hidden,omitempty"`
	// Requires is a pointer so an app without requirements writes no empty object.
	Requires *CatalogRequires `json:"requires,omitempty"`
}

// Catalog is the content of apps.json.
type Catalog struct {
	V        int          `json:"v"`
	Sequence int64        `json:"sequence"`
	Issued   string       `json:"issued"`
	Apps     []CatalogApp `json:"apps"`
}

// AppTag is the git tag of one app release, e.g. todo-v1.0.2.
func AppTag(id, version string) string { return id + "-v" + version }

// DownloadPath is the path of the app's zip below the repo URL:
// releases/download/<tag>/<zip>.
func (app CatalogApp) DownloadPath() string {
	return "releases/download/" + app.Tag + "/" + app.Zip
}

// DisplayName is the name in the given language (exact, then the language part), else English.
func (app CatalogApp) DisplayName(language string) string {
	for _, candidate := range []string{language, strings.SplitN(strings.ReplaceAll(language, "-", "_"), "_", 2)[0]} {
		if name := app.Names[candidate]; name != "" {
			return name
		}
	}
	return app.Name
}

// Find returns the entry of an app id.
func (catalog *Catalog) Find(id string) (CatalogApp, bool) {
	for _, app := range catalog.Apps {
		if app.ID == id {
			return app, true
		}
	}
	return CatalogApp{}, false
}

// SignCatalog signs the exact catalog bytes with the domain-separation prefix.
func SignCatalog(key ed25519.PrivateKey, catalogBytes []byte) string {
	return Sign(key, catalogSignedMessage(catalogBytes))
}

// VerifyCatalog checks a detached signature of the catalog bytes. Callers parse only after
// this succeeds. A drawer release.json signature fails here because it lacks the prefix.
func VerifyCatalog(key ed25519.PublicKey, catalogBytes []byte, signatureText string) error {
	if len(catalogBytes) > MaxCatalogBytes {
		return fmt.Errorf("%s is larger than %d bytes", CatalogName, MaxCatalogBytes)
	}
	if err := Verify(key, catalogSignedMessage(catalogBytes), signatureText); err != nil {
		return errors.New("signature does not match the app catalog")
	}
	return nil
}

func catalogSignedMessage(catalogBytes []byte) []byte {
	return append([]byte(CatalogSignaturePrefix), catalogBytes...)
}

// ParseCatalog decodes and validates apps.json bytes. It does not check the signature.
func ParseCatalog(data []byte) (*Catalog, error) {
	if len(data) > MaxCatalogBytes {
		return nil, fmt.Errorf("%s is larger than %d bytes", CatalogName, MaxCatalogBytes)
	}
	var catalog Catalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("%s is not valid JSON for this format: %w", CatalogName, err)
	}
	if err := catalog.Validate(); err != nil {
		return nil, err
	}
	return &catalog, nil
}

// Validate checks every rule of the format.
func (catalog *Catalog) Validate() error {
	if catalog.V != CatalogSchema {
		return fmt.Errorf("v must be %d (got %d)", CatalogSchema, catalog.V)
	}
	if catalog.Sequence < 1 {
		return fmt.Errorf("sequence must be 1 or more (got %d)", catalog.Sequence)
	}
	if _, err := time.Parse(time.RFC3339, catalog.Issued); err != nil {
		return fmt.Errorf("issued %q must be an RFC3339 timestamp: %w", catalog.Issued, err)
	}
	if len(catalog.Apps) > maxCatalogApps {
		return fmt.Errorf("the catalog lists %d apps; at most %d", len(catalog.Apps), maxCatalogApps)
	}
	seen := map[string]bool{}
	for _, app := range catalog.Apps {
		if err := app.validate(); err != nil {
			return fmt.Errorf("app %q: %w", app.ID, err)
		}
		if seen[app.ID] {
			return fmt.Errorf("app %q is listed twice", app.ID)
		}
		seen[app.ID] = true
	}
	return nil
}

func (app CatalogApp) validate() error {
	if !appIDPattern.MatchString(app.ID) || !ValidVersion(app.Version) {
		return fmt.Errorf("id and version %q must be a valid app id and MAJOR.MINOR.PATCH", app.Version)
	}
	if app.Tag != AppTag(app.ID, app.Version) {
		return fmt.Errorf("tag must be %s (got %q)", AppTag(app.ID, app.Version), app.Tag)
	}
	if app.Zip != AppZipName(app.ID, app.Version) {
		return fmt.Errorf("zip must be named %s (got %q)", AppZipName(app.ID, app.Version), app.Zip)
	}
	if !sha256Pattern.MatchString(app.SHA256) {
		return fmt.Errorf("sha256 must be 64 lowercase hex digits")
	}
	if app.Size <= 0 || app.Size > MaxAppZipBytes {
		return fmt.Errorf("size %d must be between 1 and %d bytes", app.Size, MaxAppZipBytes)
	}
	if err := validateDisplayName(app.Name); err != nil {
		return fmt.Errorf("name: %w", err)
	}
	if err := validateNames(app.Names); err != nil {
		return err
	}
	if app.Requires != nil && app.Requires.Drawer != "" {
		if _, err := ParseDrawerConstraint(app.Requires.Drawer); err != nil {
			return err
		}
	}
	return nil
}

func validateNames(names map[string]string) error {
	for language, name := range names {
		if !catalogLanguagePattern.MatchString(language) {
			return fmt.Errorf("names: language %q is not a locale code like vi or pt_BR", language)
		}
		if err := validateDisplayName(name); err != nil {
			return fmt.Errorf("names[%s]: %w", language, err)
		}
	}
	return nil
}

func validateDisplayName(name string) error {
	if strings.TrimSpace(name) == "" || utf8.RuneCountInString(name) > maxAppNameRunes {
		return fmt.Errorf("must be 1 to %d characters", maxAppNameRunes)
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return fmt.Errorf("must not contain control characters")
		}
	}
	return nil
}

// ParseDrawerConstraint reads ">=X.Y.Z" and returns the minimum version.
func ParseDrawerConstraint(constraint string) (string, error) {
	minimum, hasPrefix := strings.CutPrefix(constraint, ">=")
	if !hasPrefix || !ValidVersion(minimum) {
		return "", fmt.Errorf("requires.drawer %q must look like >=1.0.0", constraint)
	}
	return minimum, nil
}

// MeetsDrawerRequirement reports whether the running drawer version satisfies the app's
// requires.drawer. An app without a requirement fits every drawer.
func (app CatalogApp) MeetsDrawerRequirement(drawerVersion string) (bool, error) {
	if app.Requires == nil || app.Requires.Drawer == "" {
		return true, nil
	}
	minimum, err := ParseDrawerConstraint(app.Requires.Drawer)
	if err != nil {
		return false, err
	}
	order, err := CompareVersions(drawerVersion, minimum)
	return order >= 0, err
}

// MarshalCatalog writes the catalog as indented JSON with a trailing newline, apps sorted by id.
func MarshalCatalog(catalog *Catalog) ([]byte, error) {
	sort.Slice(catalog.Apps, func(i, j int) bool { return catalog.Apps[i].ID < catalog.Apps[j].ID })
	if err := catalog.Validate(); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		return nil, err
	}
	if len(data)+1 > MaxCatalogBytes {
		return nil, fmt.Errorf("the catalog is %d bytes; devices read at most %d", len(data)+1, MaxCatalogBytes)
	}
	return append(data, '\n'), nil
}
