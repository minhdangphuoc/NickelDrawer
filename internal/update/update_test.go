package update

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"koboapps/internal/i18n"
	"koboapps/internal/release"
)

const demoManifestTemplate = `{"schema":1,"id":"demo","name":"Demo","version":"%s",` +
	`"actions":[{"id":"go","label":"Run","run":"bin/run.sh","output":"show"}]}`

func writeFile(t *testing.T, root, relative, content string, mode os.FileMode) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
}

// writeVersionTree writes the files of one release (host plus the demo app) under root.
// printedVersion is what the fake appsctl prints for --version.
func writeVersionTree(t *testing.T, root, version, printedVersion string) {
	t.Helper()
	writeFile(t, root, ".adds/koboapps/bin/appsctl", "#!/bin/sh\necho appsctl "+printedVersion+"\n", 0o755)
	writeFile(t, root, ".adds/koboapps/lib/common.sh", "# lib "+version+"\n", 0o644)
	writeFile(t, root, ".adds/koboapps/locales/en.json", `{"v":"`+version+`"}`, 0o644)
	writeFile(t, root, ".adds/apps/demo/app.json", fmt.Sprintf(demoManifestTemplate, version), 0o644)
	writeFile(t, root, ".adds/apps/demo/bin/run.sh", "#!/bin/sh\necho "+version+"\n", 0o755)
	writeFile(t, root, ".adds/apps/demo/locales/en.json", `{"name":"Demo"}`, 0o644)
}

type publishedRelease struct {
	manifestBytes []byte
	signature     string
	zipBytes      []byte
}

// buildRelease signs a release made from dist; zipSource, when set, supplies the zip
// content instead (to simulate a tampered download).
func buildRelease(t *testing.T, dist, version string, signer ed25519.PrivateKey, zipSource string) publishedRelease {
	t.Helper()
	parsed, err := release.BuildManifest(dist, version, "0.1.0", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	manifestBytes, err := parsed.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if zipSource == "" {
		zipSource = dist
	}
	var zipBuffer bytes.Buffer
	if err := release.WriteZip(zipSource, parsed, &zipBuffer); err != nil {
		t.Fatal(err)
	}
	return publishedRelease{manifestBytes, release.Sign(signer, manifestBytes), zipBuffer.Bytes()}
}

type fakeGitHub struct {
	server   *httptest.Server
	releases map[string]publishedRelease // by tag
	stable   string
	beta     string
	requests int
}

func newFakeGitHub(t *testing.T) *fakeGitHub {
	fake := &fakeGitHub{releases: map[string]publishedRelease{}}
	fake.server = httptest.NewServer(fake)
	t.Cleanup(fake.server.Close)
	return fake
}

func (fake *fakeGitHub) publishStable(version string, published publishedRelease) {
	fake.releases["v"+version] = published
	fake.stable = "v" + version
}

func (fake *fakeGitHub) releaseJSON(tag string, prerelease bool) map[string]any {
	base := fake.server.URL + "/dl/" + tag + "/"
	return map[string]any{
		"tag_name": tag, "prerelease": prerelease,
		"assets": []map[string]string{
			{"name": release.ManifestName, "browser_download_url": base + release.ManifestName},
			{"name": release.SignatureName, "browser_download_url": base + release.SignatureName},
			{"name": "nickeldrawer-" + strings.TrimPrefix(tag, "v") + ".zip", "browser_download_url": base + "release.zip"},
		},
	}
}

func (fake *fakeGitHub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fake.requests++
	switch {
	case r.URL.Path == "/repos/o/r/releases/latest" && fake.stable != "":
		json.NewEncoder(w).Encode(fake.releaseJSON(fake.stable, false))
	case r.URL.Path == "/repos/o/r/releases":
		var list []map[string]any
		if fake.beta != "" {
			list = append(list, fake.releaseJSON(fake.beta, true))
		}
		if fake.stable != "" {
			list = append(list, fake.releaseJSON(fake.stable, false))
		}
		json.NewEncoder(w).Encode(list)
	case strings.HasPrefix(r.URL.Path, "/dl/"):
		fake.serveAsset(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (fake *fakeGitHub) serveAsset(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/dl/"), "/")
	published, found := fake.releases[parts[0]]
	if !found || len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	switch parts[1] {
	case release.ManifestName:
		w.Write(published.manifestBytes)
	case release.SignatureName:
		w.Write([]byte(published.signature))
	default:
		w.Write(published.zipBytes)
	}
}

type fixture struct {
	t          *testing.T
	root       string
	github     *fakeGitHub
	publicKey  ed25519.PublicKey
	privateKey ed25519.PrivateKey
	menuCalls  int
	menuError  error
}

func newFixture(t *testing.T, installedVersion string) *fixture {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	f := &fixture{t: t, root: t.TempDir(), github: newFakeGitHub(t), publicKey: publicKey, privateKey: privateKey}
	writeVersionTree(t, f.root, installedVersion, installedVersion)
	writeFile(t, f.root, ".adds/koboapps/data/update.json", `{"repo":"o/r","channel":"stable"}`, 0o644)
	writeFile(t, f.root, ".adds/apps/demo/data/token", "secret-token", 0o600)
	writeFile(t, f.root, ".adds/apps/demo/data/nested/state.json", `{"n":1}`, 0o644)
	return f
}

// publishVersion builds a release tree for version and serves it as the stable release.
func (f *fixture) publishVersion(version, printedVersion string) {
	dist := f.t.TempDir()
	writeVersionTree(f.t, dist, version, printedVersion)
	f.github.publishStable(version, buildRelease(f.t, dist, version, f.privateKey, ""))
}

func (f *fixture) updater(currentVersion string) *Updater {
	return New(Options{
		OnboardRoot: f.root, APIBase: f.github.server.URL, TempDir: f.t.TempDir(),
		CurrentVersion: currentVersion,
		PublicKey:      func() (ed25519.PublicKey, error) { return f.publicKey, nil },
		RegenerateMenu: func() error { f.menuCalls++; return f.menuError },
	})
}

// snapshot maps every file under root to its content and mode, skipping update bookkeeping.
func snapshot(t *testing.T, root string, skip ...string) map[string]string {
	t.Helper()
	result := map[string]string{}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		relative, _ := filepath.Rel(root, path)
		relative = filepath.ToSlash(relative)
		for _, prefix := range skip {
			if strings.HasPrefix(relative, prefix) {
				return nil
			}
		}
		content, readErr := os.ReadFile(path)
		result[relative] = fmt.Sprintf("%o:%s", info.Mode().Perm(), content)
		return readErr
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func appDataSnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	all := snapshot(t, root)
	data := map[string]string{}
	for path, content := range all {
		if strings.HasPrefix(path, ".adds/apps/") && strings.Contains(path, "/data/") {
			data[path] = content
		}
	}
	if len(data) == 0 {
		t.Fatal("fixture has no app data")
	}
	return data
}

func sameSnapshot(t *testing.T, label string, got, want map[string]string) {
	t.Helper()
	var differences []string
	for path, content := range want {
		if got[path] != content {
			differences = append(differences, "changed or missing: "+path)
		}
	}
	for path := range got {
		if _, found := want[path]; !found {
			differences = append(differences, "unexpected: "+path)
		}
	}
	sort.Strings(differences)
	if len(differences) > 0 {
		t.Fatalf("%s differs:\n%s", label, strings.Join(differences, "\n"))
	}
}

const bookkeeping = ".adds/koboapps/data/update/"

func messageKey(err error) string {
	var messageErr *i18n.MessageError
	if errors.As(err, &messageErr) {
		return messageErr.Key
	}
	return fmt.Sprint(err)
}

func TestValidUpdateIsApplied(t *testing.T) {
	f := newFixture(t, "1.0.0")
	dataBefore := appDataSnapshot(t, f.root)
	f.publishVersion("1.1.0", "1.1.0")

	result, err := f.updater("1.0.0").Apply(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Key != "update.applied" || result.Params["version"] != "1.1.0" || result.Params["previous"] != "1.0.0" {
		t.Fatalf("report = %+v", result)
	}
	want := t.TempDir()
	writeVersionTree(t, want, "1.1.0", "1.1.0")
	got := snapshot(t, f.root, ".adds/koboapps/data/", ".adds/apps/demo/data/")
	sameSnapshot(t, "installed files", got, snapshot(t, want))
	sameSnapshot(t, "app data", appDataSnapshot(t, f.root), dataBefore)
	if f.menuCalls != 1 {
		t.Fatalf("menu regenerated %d times, want 1", f.menuCalls)
	}
	if _, err := os.Stat(filepath.Join(f.root, ".adds/koboapps/data/rollback/1.0.0/journal.json")); err != nil {
		t.Fatalf("no backup journal: %v", err)
	}
	if _, err := os.Stat(filepath.Join(f.root, filepath.FromSlash(bookkeeping), "staging")); !os.IsNotExist(err) {
		t.Fatal("staging folder left behind")
	}
	status, found, err := ReadStatus(Layout{OnboardRoot: f.root})
	if err != nil || !found || status.State != StateDone || status.Report.Key != "update.applied" {
		t.Fatalf("status = %+v found=%v err=%v", status, found, err)
	}
	info, _ := os.Stat(filepath.Join(f.root, ".adds/koboapps/bin/appsctl"))
	if info.Mode().Perm()&0o111 == 0 {
		t.Fatal("appsctl lost its executable bit")
	}
}

func TestBadSignatureChangesNothing(t *testing.T) {
	f := newFixture(t, "1.0.0")
	_, otherKey, _ := ed25519.GenerateKey(rand.Reader)
	dist := t.TempDir()
	writeVersionTree(t, dist, "1.1.0", "1.1.0")
	f.github.publishStable("1.1.0", buildRelease(t, dist, "1.1.0", otherKey, ""))
	before := snapshot(t, f.root, bookkeeping)

	_, err := f.updater("1.0.0").Apply(context.Background())
	if messageKey(err) != "update.bad_signature" {
		t.Fatalf("err = %v", err)
	}
	sameSnapshot(t, "tree", snapshot(t, f.root, bookkeeping), before)
	if f.menuCalls != 0 {
		t.Fatal("menu regenerated for a rejected update")
	}
}

func TestShaMismatchChangesNothing(t *testing.T) {
	f := newFixture(t, "1.0.0")
	good := t.TempDir()
	writeVersionTree(t, good, "1.1.0", "1.1.0")
	evil := t.TempDir()
	writeVersionTree(t, evil, "1.1.0", "1.1.0")
	writeFile(t, evil, ".adds/apps/demo/bin/run.sh", "#!/bin/sh\necho evil-same-size!\n", 0o755)
	f.github.publishStable("1.1.0", buildRelease(t, good, "1.1.0", f.privateKey, evil))
	before := snapshot(t, f.root, bookkeeping)

	_, err := f.updater("1.0.0").Apply(context.Background())
	if messageKey(err) != "update.file_mismatch" {
		t.Fatalf("err = %v", err)
	}
	sameSnapshot(t, "tree", snapshot(t, f.root, bookkeeping), before)
}

func TestSignedManifestCannotReachUserData(t *testing.T) {
	f := newFixture(t, "1.0.0")
	manifestJSON := `{"schema":1,"version":"1.1.0","created":"2026-10-08T10:00:00Z","min_appsctl":"0.1.0",` +
		`"files":[{"path":".adds/apps/demo/data/token","sha256":"` + strings.Repeat("a", 64) + `","size":1,"mode":"0644"}],"apps":[]}`
	f.github.releases["v1.1.0"] = publishedRelease{[]byte(manifestJSON), release.Sign(f.privateKey, []byte(manifestJSON)), nil}
	f.github.stable = "v1.1.0"
	dataBefore := appDataSnapshot(t, f.root)

	_, err := f.updater("1.0.0").Apply(context.Background())
	if messageKey(err) != "update.bad_manifest" {
		t.Fatalf("err = %v", err)
	}
	sameSnapshot(t, "app data", appDataSnapshot(t, f.root), dataBefore)
}

func TestDowngradeNeedsExplicitPermission(t *testing.T) {
	f := newFixture(t, "2.0.0")
	f.publishVersion("1.1.0", "1.1.0")
	before := snapshot(t, f.root, bookkeeping)

	_, err := f.updater("2.0.0").Apply(context.Background())
	if messageKey(err) != "update.downgrade" {
		t.Fatalf("err = %v", err)
	}
	sameSnapshot(t, "tree", snapshot(t, f.root, bookkeeping), before)

	allowing := f.updater("2.0.0")
	allowing.options.AllowDowngrade = true
	result, err := allowing.Apply(context.Background())
	if err != nil || result.Key != "update.applied" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestFailedSelfCheckRestoresEverything(t *testing.T) {
	f := newFixture(t, "1.0.0")
	f.publishVersion("1.1.0", "9.9.9") // the new appsctl claims a different version
	before := snapshot(t, f.root, bookkeeping)
	dataBefore := appDataSnapshot(t, f.root)

	_, err := f.updater("1.0.0").Apply(context.Background())
	if messageKey(err) != "update.failed_restored" {
		t.Fatalf("err = %v", err)
	}
	sameSnapshot(t, "tree", snapshot(t, f.root, bookkeeping), before)
	sameSnapshot(t, "app data", appDataSnapshot(t, f.root), dataBefore)
	if f.menuCalls != 2 {
		t.Fatalf("menu regenerated %d times, want 2 (after swap and after restore)", f.menuCalls)
	}
	if _, statErr := os.Stat(filepath.Join(f.root, ".adds/koboapps/data/rollback/1.0.0")); !os.IsNotExist(statErr) {
		t.Fatal("restored backup should be dropped")
	}
	status, _, _ := ReadStatus(Layout{OnboardRoot: f.root})
	if status.State != StateFailed || status.Report.Key != "update.failed_restored" {
		t.Fatalf("status = %+v", status)
	}
}

func TestFailedMenuGenerationRestoresEverything(t *testing.T) {
	f := newFixture(t, "1.0.0")
	f.publishVersion("1.1.0", "1.1.0")
	f.menuError = errors.New("disk full")
	before := snapshot(t, f.root, bookkeeping)

	_, err := f.updater("1.0.0").Apply(context.Background())
	if messageKey(err) != "update.restore_failed" { // the restore's own menu rebuild also fails
		t.Fatalf("err = %v", err)
	}
	sameSnapshot(t, "tree", snapshot(t, f.root, bookkeeping, ".adds/koboapps/data/rollback/"), before)
}

func withoutPrefix(files map[string]string, prefix string) map[string]string {
	kept := map[string]string{}
	for path, content := range files {
		if !strings.HasPrefix(path, prefix) {
			kept[path] = content
		}
	}
	return kept
}

func TestSecondApplyOfTheSameVersionIsANoOp(t *testing.T) {
	f := newFixture(t, "1.0.0")
	f.publishVersion("1.1.0", "1.1.0")
	if _, err := f.updater("1.0.0").Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, f.root, bookkeeping)
	menuCalls := f.menuCalls

	result, err := f.updater("1.1.0").Apply(context.Background())
	if err != nil || result.Key != "update.up_to_date" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	sameSnapshot(t, "tree", snapshot(t, f.root, bookkeeping), before)
	if f.menuCalls != menuCalls {
		t.Fatal("no-op regenerated the menu")
	}
}

func TestRollbackRestoresFilesAndKeepsData(t *testing.T) {
	f := newFixture(t, "1.0.0")
	original := snapshot(t, f.root, bookkeeping)
	dataBefore := appDataSnapshot(t, f.root)
	f.publishVersion("1.1.0", "1.1.0")
	if _, err := f.updater("1.0.0").Apply(context.Background()); err != nil {
		t.Fatal(err)
	}

	result, err := f.updater("1.1.0").Rollback(context.Background())
	if err != nil || result.Key != "update.rolled_back" || result.Params["version"] != "1.0.0" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	sameSnapshot(t, "tree", snapshot(t, f.root, ".adds/koboapps/data/update/", ".adds/koboapps/data/rollback/"), withoutPrefix(original, ".adds/koboapps/data/rollback/"))
	sameSnapshot(t, "app data", appDataSnapshot(t, f.root), dataBefore)
	if _, err := f.updater("1.0.0").Rollback(context.Background()); messageKey(err) != "update.no_rollback" {
		t.Fatalf("second rollback err = %v", err)
	}
}

func TestRollbackDeletesFilesTheUpdateAdded(t *testing.T) {
	f := newFixture(t, "1.0.0")
	dist := t.TempDir()
	writeVersionTree(t, dist, "1.1.0", "1.1.0")
	writeFile(t, dist, ".adds/koboapps/bin/update-apply.sh", "#!/bin/sh\n", 0o755)
	f.github.publishStable("1.1.0", buildRelease(t, dist, "1.1.0", f.privateKey, ""))
	if _, err := f.updater("1.0.0").Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	added := filepath.Join(f.root, ".adds/koboapps/bin/update-apply.sh")
	if _, err := os.Stat(added); err != nil {
		t.Fatal(err)
	}
	if _, err := f.updater("1.1.0").Rollback(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(added); !os.IsNotExist(err) {
		t.Fatal("added file survived the rollback")
	}
}

func TestOnlyTheLastTwoBackupsAreKept(t *testing.T) {
	f := newFixture(t, "1.0.0")
	previous := "1.0.0"
	for _, version := range []string{"1.1.0", "1.2.0", "1.3.0"} {
		f.publishVersion(version, version)
		if _, err := f.updater(previous).Apply(context.Background()); err != nil {
			t.Fatal(err)
		}
		previous = version
	}
	versions, err := Layout{OnboardRoot: f.root}.backupVersions()
	if err != nil || strings.Join(versions, ",") != "1.2.0,1.1.0" {
		t.Fatalf("backups = %v, err=%v", versions, err)
	}
}

func TestMissingConfigAndPlaceholderKeyGiveClearMessages(t *testing.T) {
	f := newFixture(t, "1.0.0")
	f.publishVersion("1.1.0", "1.1.0")
	os.Remove(filepath.Join(f.root, ".adds/koboapps/data/update.json"))
	_, err := f.updater("1.0.0").Apply(context.Background())
	assertLocalized(t, err, "update.no_config", "update.json")
	if f.github.requests != 0 {
		t.Fatal("network used without configuration")
	}

	writeFile(t, f.root, ".adds/koboapps/data/update.json", `{"repo":"o/r"}`, 0o644)
	placeholder := f.updater("1.0.0")
	placeholder.options.PublicKey = func() (ed25519.PublicKey, error) { return nil, release.ErrNoPublicKey }
	_, err = placeholder.Apply(context.Background())
	assertLocalized(t, err, "update.no_public_key", "")
	if f.github.requests != 0 {
		t.Fatal("network used without a trusted key")
	}
}

// assertLocalized checks the error has the key and renders to real text in en and vi.
func assertLocalized(t *testing.T, err error, key, mustContain string) {
	t.Helper()
	if messageKey(err) != key {
		t.Fatalf("err = %v, want key %s", err, key)
	}
	for _, locale := range []string{"en", "vi"} {
		bundle, loadErr := i18n.LoadBundle(filepath.Join("..", "..", "host", "locales"), locale)
		if loadErr != nil {
			t.Fatal(loadErr)
		}
		text := bundle.Describe(err)
		if strings.Contains(text, "update.") && !strings.Contains(text, "update.json") || text == key {
			t.Errorf("%s: %q is not localized", locale, text)
		}
		if locale == "en" && !strings.Contains(text, mustContain) {
			t.Errorf("en text %q lacks %q", text, mustContain)
		}
	}
}

func TestEveryUpdateMessageKeyExistsInBothLocales(t *testing.T) {
	keys := []string{
		"update.no_config", "update.bad_config", "update.bad_repo", "update.bad_channel", "update.no_public_key",
		"update.bad_public_key", "update.network", "update.no_release", "update.missing_asset", "update.bad_signature",
		"update.bad_manifest", "update.file_mismatch", "update.downgrade", "update.needs_newer_appsctl",
		"update.up_to_date", "update.available", "update.applied", "update.failed_restored", "update.restore_failed",
		"update.menu_failed", "update.already_running", "update.no_rollback", "update.rolled_back", "update.running",
		"update.interrupted", "update.failed", "update.status_never", "update.status_version", "update.started", "update.wifi_missing",
	}
	for _, locale := range []string{"en", "vi"} {
		catalog, err := i18n.LoadCatalog(filepath.Join("..", "..", "host", "locales", locale+".json"))
		if err != nil {
			t.Fatal(err)
		}
		for _, key := range keys {
			if catalog[key] == "" {
				t.Errorf("%s.json lacks %s", locale, key)
			}
		}
	}
}

func TestCheckReportsAvailableAndUpToDate(t *testing.T) {
	f := newFixture(t, "1.0.0")
	f.publishVersion("1.1.0", "1.1.0")
	before := snapshot(t, f.root)
	result, err := f.updater("1.0.0").Check(context.Background())
	if err != nil || result.Key != "update.available" || result.Params["version"] != "1.1.0" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if result, err = f.updater("1.1.0").Check(context.Background()); err != nil || result.Key != "update.up_to_date" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	sameSnapshot(t, "tree", snapshot(t, f.root), before)
}

func TestMinimumAppsctlIsEnforced(t *testing.T) {
	f := newFixture(t, "1.0.0")
	f.publishVersion("1.1.0", "1.1.0")
	_, err := f.updater("0.0.5").Apply(context.Background())
	if messageKey(err) != "update.needs_newer_appsctl" {
		t.Fatalf("err = %v", err)
	}
}

func TestBetaChannelAcceptsPrereleasesStableDoesNot(t *testing.T) {
	f := newFixture(t, "1.0.0")
	f.publishVersion("1.1.0", "1.1.0")
	betaDist := t.TempDir()
	writeVersionTree(t, betaDist, "1.2.0", "1.2.0")
	f.github.releases["v1.2.0-beta"] = buildRelease(t, betaDist, "1.2.0", f.privateKey, "")
	f.github.beta = "v1.2.0-beta"

	result, err := f.updater("1.0.0").Check(context.Background())
	if err != nil || result.Params["version"] != "1.1.0" {
		t.Fatalf("stable result=%+v err=%v", result, err)
	}
	writeFile(t, f.root, ".adds/koboapps/data/update.json", `{"repo":"o/r","channel":"beta"}`, 0o644)
	result, err = f.updater("1.0.0").Check(context.Background())
	if err != nil || result.Params["version"] != "1.2.0" {
		t.Fatalf("beta result=%+v err=%v", result, err)
	}
}

func TestBadConfigValues(t *testing.T) {
	f := newFixture(t, "1.0.0")
	cases := map[string]string{
		`{"repo":"no-slash"}`:                     "update.bad_repo",
		`{"repo":"o/r","channel":"nightly"}`:      "update.bad_channel",
		`not json`:                                "update.bad_config",
		`{"repo":"../../etc/passwd/../x/y"}`:      "update.bad_repo",
		`{"repo":"o/r","channel":"stable","x":1}`: "",
	}
	for content, wantKey := range cases {
		writeFile(t, f.root, ".adds/koboapps/data/update.json", content, 0o644)
		_, err := LoadConfig(Layout{OnboardRoot: f.root})
		if wantKey == "" && err != nil || wantKey != "" && messageKey(err) != wantKey {
			t.Errorf("%s: err = %v, want %q", content, err, wantKey)
		}
	}
}

func TestSecondRunIsRefusedWhileOneIsRunning(t *testing.T) {
	f := newFixture(t, "1.0.0")
	f.publishVersion("1.1.0", "1.1.0")
	layout := Layout{OnboardRoot: f.root}
	// PID 1 is always alive and is never this test process.
	if err := writeStatus(layout, Status{State: StateRunning, PID: 1, Report: report("update.running")}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.updater("1.0.0").Apply(context.Background()); messageKey(err) != "update.already_running" {
		t.Fatalf("err = %v", err)
	}
	if err := writeStatus(layout, Status{State: StateRunning, PID: 0x7ffffff0, Report: report("update.running")}); err != nil {
		t.Fatal(err)
	}
	status, _, _ := ReadStatus(layout)
	if status.State != StateFailed || status.Report.Key != "update.interrupted" {
		t.Fatalf("dead runner status = %+v", status)
	}
}
