package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"koboapps/internal/release"
	"koboapps/internal/update"
)

// appsRelease plays the project's GitHub repo: it packs apps the way `make dist` does, signs the
// catalog the way `ndrelease sign-apps` does (with a throwaway key) and serves both.
type appsRelease struct {
	t      *testing.T
	key    ed25519.PrivateKey
	public ed25519.PublicKey
	server *httptest.Server
	mu     sync.Mutex
	files  map[string][]byte
}

func newAppsRelease(t *testing.T) *appsRelease {
	public, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	repo := &appsRelease{t: t, key: key, public: public, files: map[string][]byte{}}
	repo.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		repo.mu.Lock()
		defer repo.mu.Unlock()
		body, found := repo.files[r.URL.Path]
		if !found {
			http.NotFound(w, r)
			return
		}
		w.Write(body)
	}))
	t.Cleanup(repo.server.Close)
	updateOptionsHook = func(options *update.Options) {
		options.PublicKey = func() (ed25519.PublicKey, error) { return public, nil }
	}
	t.Cleanup(func() { updateOptionsHook = func(*update.Options) {} })
	return repo
}

// publish packs the apps, then serves their zips on per-app tags and a catalog on `apps`.
func (repo *appsRelease) publish(sequence int64, apps map[string]string) {
	repo.t.Helper()
	tree, zips := repo.t.TempDir(), repo.t.TempDir()
	for id, version := range apps {
		writeApp(repo.t, filepath.Join(tree, ".adds", "apps"), id,
			`{"schema":1,"id":"`+id+`","name":"`+strings.ToUpper(id)+`","version":"`+version+`","actions":[{"id":"go","label":"Run","run":"bin/run.sh","output":"show"}]}`)
		writeLocale(repo.t, filepath.Join(tree, ".adds", "apps"), id, "vi", `{"name":"Ứng dụng `+id+`"}`)
		if _, err := release.PackApp(tree, id, zips); err != nil {
			repo.t.Fatal(err)
		}
	}
	catalog, err := release.BuildCatalog(zips, sequence, time.Now())
	if err != nil {
		repo.t.Fatal(err)
	}
	data, err := release.MarshalCatalog(catalog)
	if err != nil {
		repo.t.Fatal(err)
	}
	repo.mu.Lock()
	defer repo.mu.Unlock()
	for _, app := range catalog.Apps {
		body, _ := os.ReadFile(filepath.Join(zips, app.Zip))
		repo.files["/o/r/"+app.DownloadPath()] = body
	}
	repo.files["/o/r/releases/download/apps/apps.json"] = data
	repo.files["/o/r/releases/download/apps/apps.json.sig"] = []byte(release.SignCatalog(repo.key, data))
}

// kobo is a fake onboard tree with the drawer's host folder and locales.
type kobo struct {
	t    *testing.T
	root string
	repo *appsRelease
}

func newKobo(t *testing.T, repo *appsRelease) *kobo {
	root := t.TempDir()
	copyHostLocales(t, filepath.Join(root, ".adds", "koboapps"))
	writeFileAt(t, filepath.Join(root, ".adds/koboapps/data/update.json"), `{"repo":"o/r"}`)
	return &kobo{t: t, root: root, repo: repo}
}

func writeFileAt(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// appsctl runs `appsctl update <args>` against the fake Kobo and returns exit code and output.
func (device *kobo) appsctl(locale string, args ...string) (int, string, string) {
	device.t.Helper()
	var out, errOut bytes.Buffer
	full := append([]string{"update"}, args...)
	full = append(full, "--onboard-root", device.root, "--download-base", device.repo.server.URL,
		"--tmp-dir", device.t.TempDir(), "--locale", locale)
	return run(full, &out, &errOut), out.String(), errOut.String()
}

func (device *kobo) read(relative string) string {
	data, _ := os.ReadFile(filepath.Join(device.root, filepath.FromSlash(relative)))
	return string(data)
}

func (device *kobo) installedVersion(id string) string {
	return update.InstalledAppVersions(update.Layout{OnboardRoot: device.root})[id]
}

// TestAppUpdatesEndToEnd walks the whole path through the command line: check, update one app,
// install another, recover from a power cut, roll back, update all, and refuse an old catalog.
func TestAppUpdatesEndToEnd(t *testing.T) {
	repo := newAppsRelease(t)
	device := newKobo(t, repo)
	apps := filepath.Join(device.root, ".adds", "apps")
	writeApp(t, apps, "todo", `{"schema":1,"id":"todo","name":"TODO","version":"1.0.0","actions":[{"id":"go","label":"Run","run":"bin/run.sh","output":"show"}]}`)
	writeFileAt(t, filepath.Join(apps, "todo/data/list.json"), `["milk","eggs"]`)
	repo.publish(1, map[string]string{"todo": "1.0.1", "paper": "0.5.0"})

	code, out, errOut := device.appsctl("en", "apps")
	if code != exitOK || !strings.Contains(out, "updates available: 1") || !strings.Contains(out, "TODO: 1.0.0 → 1.0.1") ||
		!strings.Contains(out, "PAPER (not installed)") {
		t.Fatalf("check: code=%d out=%q err=%q", code, out, errOut)
	}

	code, out, _ = device.appsctl("vi", "app", "todo")
	if code != exitOK || !strings.Contains(out, "Đã cập nhật Ứng dụng todo lên bản 1.0.1") {
		t.Fatalf("update: code=%d out=%q", code, out)
	}
	if device.installedVersion("todo") != "1.0.1" || device.read(".adds/apps/todo/data/list.json") != `["milk","eggs"]` {
		t.Fatalf("todo=%s data=%q", device.installedVersion("todo"), device.read(".adds/apps/todo/data/list.json"))
	}

	if code, out, _ = device.appsctl("en", "app", "paper"); code != exitOK || !strings.Contains(out, "Installed PAPER 0.5.0") || device.installedVersion("paper") != "0.5.0" {
		t.Fatalf("install: code=%d out=%q", code, out)
	}

	// A power cut after the installed folder was moved aside and its data into the staged one.
	if err := os.MkdirAll(filepath.Join(apps, ".todo.new"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(apps, "todo/data"), filepath.Join(apps, ".todo.new/data")); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(apps, "todo"), filepath.Join(apps, ".todo.old")); err != nil {
		t.Fatal(err)
	}
	if code, _, errOut = device.appsctl("en", "apps"); code != exitOK {
		t.Fatalf("recovery run: %d %s", code, errOut)
	}
	if device.installedVersion("todo") != "1.0.1" || device.read(".adds/apps/todo/data/list.json") != `["milk","eggs"]` {
		t.Fatalf("after recovery: todo=%s data=%q", device.installedVersion("todo"), device.read(".adds/apps/todo/data/list.json"))
	}

	if code, out, _ = device.appsctl("en", "rollback-app", "todo"); code != exitOK || !strings.Contains(out, "Restored TODO 1.0.0") || device.installedVersion("todo") != "1.0.0" {
		t.Fatalf("rollback: code=%d out=%q todo=%s", code, out, device.installedVersion("todo"))
	}
	if code, out, _ = device.appsctl("en", "rollback-app", "todo"); code != exitFailure || !strings.Contains(out, "no earlier version") {
		t.Fatalf("second rollback: code=%d out=%q", code, out)
	}

	repo.publish(2, map[string]string{"todo": "1.0.2", "paper": "0.6.0"})
	if code, out, _ = device.appsctl("en", "apps", "--all"); code != exitOK || !strings.Contains(out, "Updated 2 apps") ||
		device.installedVersion("todo") != "1.0.2" || device.installedVersion("paper") != "0.6.0" {
		t.Fatalf("update all: code=%d out=%q todo=%s paper=%s", code, out, device.installedVersion("todo"), device.installedVersion("paper"))
	}

	repo.publish(1, map[string]string{"todo": "1.0.1", "paper": "0.5.0"}) // an old catalog replayed
	if code, out, _ = device.appsctl("en", "apps"); code != exitFailure || !strings.Contains(out, "Old app list refused") {
		t.Fatalf("stale catalog: code=%d out=%q", code, out)
	}
	if device.installedVersion("todo") != "1.0.2" {
		t.Error("an old catalog changed an installed app")
	}
}

func TestUpdateAppNeedsExactlyOneId(t *testing.T) {
	device := newKobo(t, newAppsRelease(t))
	for _, command := range []string{"app", "rollback-app"} {
		if code, _, errOut := device.appsctl("en", command); code != exitFailure || !strings.Contains(errOut, "usage") {
			t.Errorf("%s without an id: code=%d err=%q", command, code, errOut)
		}
	}
}
