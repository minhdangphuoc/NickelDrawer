package update

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"koboapps/internal/release"
)

const testDrawerVersion = "1.0.0-beta.5"

// appSpec describes one app version the fake release host serves.
type appSpec struct {
	id, version, name string
	vi                string
	hidden            bool
	minHost           string
	extra             map[string]string // more files below the app folder
}

func (spec appSpec) manifestText() string {
	extra := ""
	if spec.hidden {
		extra += `"hidden":true,`
	}
	if spec.minHost != "" {
		extra += `"min_host":"` + spec.minHost + `",`
	}
	return fmt.Sprintf(`{"schema":1,"id":%q,"name":%q,"version":%q,%s"actions":[{"id":"go","label":"Run","run":"bin/run.sh","output":"show"}]}`,
		spec.id, spec.name, spec.version, extra)
}

// files are the zip entries below the app folder.
func (spec appSpec) files() map[string]string {
	files := map[string]string{"app.json": spec.manifestText(), "bin/run.sh": "#!/bin/sh\necho " + spec.version + "\n"}
	if spec.vi != "" {
		files["locales/vi.json"] = `{"name":"` + spec.vi + `"}`
	}
	for name, content := range spec.extra {
		files[name] = content
	}
	return files
}

// appsHost is a fake GitHub download host serving a signed catalog and app zips.
type appsHost struct {
	t          *testing.T
	server     *httptest.Server
	publicKey  ed25519.PublicKey
	privateKey ed25519.PrivateKey

	mu       sync.Mutex
	files    map[string][]byte // URL path -> body
	requests []string
}

func newAppsHost(t *testing.T) *appsHost {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	host := &appsHost{t: t, publicKey: publicKey, privateKey: privateKey, files: map[string][]byte{}}
	host.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host.mu.Lock()
		defer host.mu.Unlock()
		host.requests = append(host.requests, r.URL.Path)
		body, found := host.files[r.URL.Path]
		if !found {
			http.NotFound(w, r)
			return
		}
		w.Write(body)
	}))
	t.Cleanup(host.server.Close)
	return host
}

func zipOf(t *testing.T, id string, files map[string]string, links map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	add := func(name, content string, mode fs.FileMode) {
		header := &zip.FileHeader{Name: ".adds/apps/" + id + "/" + name, Method: zip.Deflate}
		header.SetMode(mode)
		entry, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		entry.Write([]byte(content))
	}
	for name, content := range files {
		mode := fs.FileMode(0o644)
		if strings.HasPrefix(name, "bin/") {
			mode = 0o755
		}
		add(name, content, mode)
	}
	for name, target := range links {
		add(name, target, fs.ModeSymlink|0o777)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func entryFor(spec appSpec, zipBytes []byte) release.CatalogApp {
	sum := sha256.Sum256(zipBytes)
	app := release.CatalogApp{ID: spec.id, Version: spec.version, Tag: release.AppTag(spec.id, spec.version),
		Zip: release.AppZipName(spec.id, spec.version), SHA256: hex.EncodeToString(sum[:]), Size: int64(len(zipBytes)),
		Name: spec.name, Hidden: spec.hidden}
	if spec.vi != "" {
		app.Names = map[string]string{"vi": spec.vi}
	}
	if spec.minHost != "" {
		app.Requires = &release.CatalogRequires{Drawer: ">=" + spec.minHost}
	}
	return app
}

// publish serves the apps' zips on their own tags and a signed catalog on the `apps` tag.
func (host *appsHost) publish(sequence int64, specs ...appSpec) {
	host.t.Helper()
	catalog := &release.Catalog{V: 1, Sequence: sequence, Issued: time.Now().UTC().Format(time.RFC3339)}
	host.mu.Lock()
	for _, spec := range specs {
		zipBytes := zipOf(host.t, spec.id, spec.files(), nil)
		app := entryFor(spec, zipBytes)
		host.files["/o/r/"+app.DownloadPath()] = zipBytes
		catalog.Apps = append(catalog.Apps, app)
	}
	host.mu.Unlock()
	host.serveCatalog(catalog, nil)
}

// serveCatalog signs the catalog (a nil signer means the project key) and serves it.
func (host *appsHost) serveCatalog(catalog *release.Catalog, sign func([]byte) string) {
	host.t.Helper()
	data, err := release.MarshalCatalog(catalog)
	if err != nil {
		host.t.Fatal(err)
	}
	if sign == nil {
		sign = func(data []byte) string { return release.SignCatalog(host.privateKey, data) }
	}
	host.mu.Lock()
	defer host.mu.Unlock()
	base := "/o/r/releases/download/" + release.CatalogTag + "/"
	host.files[base+release.CatalogName] = data
	host.files[base+release.CatalogSignatureName] = []byte(sign(data))
}

// replaceZip serves different bytes for an app's zip than the catalog describes.
func (host *appsHost) replaceZip(app release.CatalogApp, body []byte) {
	host.mu.Lock()
	defer host.mu.Unlock()
	host.files["/o/r/"+app.DownloadPath()] = body
}

// appsDevice is a fake Kobo onboard tree with the update config.
type appsDevice struct {
	t    *testing.T
	root string
	host *appsHost
}

func newAppsDevice(t *testing.T, host *appsHost) *appsDevice {
	root := t.TempDir()
	writeFile(t, root, ".adds/koboapps/data/update.json", `{"repo":"o/r","channel":"stable"}`, 0o644)
	if err := os.MkdirAll(filepath.Join(root, ".adds/apps"), 0o755); err != nil {
		t.Fatal(err)
	}
	return &appsDevice{t: t, root: root, host: host}
}

func (device *appsDevice) layout() Layout { return Layout{OnboardRoot: device.root} }

func (device *appsDevice) updater() *Updater {
	return New(Options{
		OnboardRoot: device.root, DownloadBase: device.host.server.URL, TempDir: device.t.TempDir(),
		CurrentVersion: testDrawerVersion, Language: "vi",
		PublicKey: func() (ed25519.PublicKey, error) { return device.host.publicKey, nil },
	})
}

// install lays an app on the device as the user would have it, with data.
func (device *appsDevice) install(spec appSpec, data map[string]string) {
	device.t.Helper()
	for name, content := range spec.files() {
		mode := fs.FileMode(0o644)
		if strings.HasPrefix(name, "bin/") {
			mode = 0o755
		}
		writeFile(device.t, device.root, ".adds/apps/"+spec.id+"/"+name, content, mode)
	}
	for name, content := range data {
		writeFile(device.t, device.root, ".adds/apps/"+spec.id+"/data/"+name, content, 0o600)
	}
}

func (device *appsDevice) installedVersion(id string) string {
	return InstalledAppVersions(device.layout())[id]
}

// dataOf reads the data folder of an app as name -> content, or nil when it has none.
func (device *appsDevice) dataOf(id string) map[string]string {
	return readTree(device.t, filepath.Join(device.root, ".adds/apps", id, "data"))
}

func readTree(t *testing.T, root string) map[string]string {
	t.Helper()
	tree := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		content, readErr := os.ReadFile(path)
		relative, _ := filepath.Rel(root, path)
		tree[filepath.ToSlash(relative)] = string(content)
		return readErr
	})
	if err != nil {
		return nil
	}
	return tree
}

// leftovers lists the staged and old folders in the apps folder.
func (device *appsDevice) leftovers() []string {
	entries, _ := os.ReadDir(filepath.Join(device.root, ".adds/apps"))
	var names []string
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			names = append(names, entry.Name())
		}
	}
	return names
}

func (device *appsDevice) checkApps() (Report, error) {
	return device.updater().CheckApps(context.Background())
}
