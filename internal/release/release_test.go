package release

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for relative, content := range files {
		full := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		mode := os.FileMode(0o644)
		if strings.Contains(relative, "/bin/") {
			mode = 0o755
		}
		if err := os.WriteFile(full, []byte(content), mode); err != nil {
			t.Fatal(err)
		}
	}
}

func sampleDist(t *testing.T) string {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		".adds/koboapps/bin/appsctl":       "binary",
		".adds/koboapps/lib/common.sh":     "lib",
		".adds/koboapps/locales/en.json":   "{}",
		".adds/koboapps/data/errors.txt":   "never shipped",
		".adds/apps/demo/app.json":         `{"id":"demo","version":"1.2.3"}`,
		".adds/apps/demo/bin/run.sh":       "run",
		".adds/apps/demo/data/token":       "secret",
		".adds/apps/demo/bin/._run.sh":     "appledouble",
		".adds/nm/koboapps":                "generated menu",
		".adds/apps/demo/locales/vi.json":  "{}",
		".adds/apps/demo/icon.svg":         "outside allowed folders",
		".adds/apps/demo/icon.png":         "png",
		".adds/koboapps/icons/refresh.png": "png",
		".adds/apps/demo/data/log/old.txt": "x",
	})
	return root
}

func TestBuildManifestListsOnlyUpdatableFiles(t *testing.T) {
	manifest, err := BuildManifest(sampleDist(t), "1.0.0", "0.1.0", time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, file := range manifest.Files {
		paths = append(paths, file.Path)
	}
	want := ".adds/koboapps/bin/appsctl,.adds/koboapps/icons/refresh.png,.adds/koboapps/lib/common.sh,.adds/koboapps/locales/en.json"
	if strings.Join(paths, ",") != want {
		t.Fatalf("paths = %v (apps must never be part of a drawer release)", paths)
	}
	if len(manifest.AppPackages) != 0 {
		t.Fatalf("app packages = %+v", manifest.AppPackages)
	}
	if manifest.Files[0].Mode != "0755" {
		t.Fatalf("mode = %s", manifest.Files[1].Mode)
	}
}

func TestCheckPathRefusesUserDataAndEscapes(t *testing.T) {
	good := []string{
		".adds/koboapps/bin/appsctl", ".adds/koboapps/lib/common.sh",
		".adds/koboapps/icons/updates.png", ModPath,
		".adds/koboapps/screens/settings/ui.json", ".adds/koboapps/screens/settings/locales/vi.json",
		".adds/koboapps/config/project.json",
	}
	bad := []string{
		".adds/apps/ghsync/app.json", ".adds/apps/ghsync/bin/ghsync", ".adds/apps/ghsync/locales/vi.json",
		".adds/apps/ghsync/icon.png", ".adds/apps/ghsync/config.example.json",
		"", "/etc/passwd", "../x", ".adds/apps/ghsync/data/token", ".adds/koboapps/data/update.json",
		".adds/apps/ghsync/bin/../data/token", ".adds/apps/ghsync/app.json/x", ".adds/apps/Bad/app.json",
		".adds/nm/koboapps", ".kobo/KoboReader.sqlite", ".adds/apps/ghsync/bin", ".adds/koboapps/bin",
		".adds/apps/ghsync/icon.svg", ".adds/apps/ghsync/icon.png/x", ".adds/koboapps/icons",
		`.adds\koboapps\bin\x`, ".adds/koboapps//bin/x",
		".adds/koboapps/mod/other.tgz", ".adds/koboapps/mod/KoboRoot-nickeldrawer.tgz/x", ".kobo/KoboRoot.tgz",
	}
	for _, p := range good {
		if err := CheckPath(p); err != nil {
			t.Errorf("CheckPath(%q) = %v", p, err)
		}
	}
	for _, p := range bad {
		if CheckPath(p) == nil {
			t.Errorf("CheckPath(%q) accepted", p)
		}
	}
}

func TestParseRejectsBadManifests(t *testing.T) {
	manifest, err := BuildManifest(sampleDist(t), "1.0.0", "0.1.0", time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	data, _ := manifest.Marshal()
	if _, err := Parse(data); err != nil {
		t.Fatal(err)
	}
	broken := map[string]func(*Manifest){
		"schema":    func(m *Manifest) { m.Schema = 2 },
		"version":   func(m *Manifest) { m.Version = "1.0" },
		"created":   func(m *Manifest) { m.Created = "yesterday" },
		"empty":     func(m *Manifest) { m.Files = nil },
		"data path": func(m *Manifest) { m.Files[0].Path = ".adds/apps/demo/data/token" },
		"sha":       func(m *Manifest) { m.Files[0].SHA256 = "abc" },
		"mode":      func(m *Manifest) { m.Files[0].Mode = "755" },
		"duplicate": func(m *Manifest) { m.Files = append(m.Files, m.Files[0]) },
		"notes":     func(m *Manifest) { m.Notes = strings.Repeat("x", maxNotesRunes+1) },
	}
	for name, mutate := range broken {
		copyOf := *manifest
		copyOf.Files = append([]File(nil), manifest.Files...)
		mutate(&copyOf)
		if err := copyOf.Validate(); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if _, err := Parse([]byte("{")); err == nil {
		t.Error("invalid JSON accepted")
	}
}

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{{"1.0.0", "1.0.0", 0}, {"1.0.1", "1.0.0", 1}, {"0.9.9", "1.0.0", -1}, {"1.10.0", "1.9.0", 1},
		{"1.0.0-beta.1", "1.0.0", -1}, {"1.0.0", "1.0.0-rc.1", 1}, {"1.0.0-beta.2", "1.0.0-beta.1", 1},
		{"1.0.0-beta.11", "1.0.0-beta.2", 1}, {"1.0.0-beta.1", "1.0.0-beta.1", 0}, {"1.0.0-alpha", "1.0.0-beta", -1},
		{"1.0.0-1", "1.0.0-alpha", -1}, {"1.0.0-beta", "1.0.0-beta.1", -1}, {"0.9.9", "1.0.0-beta.1", -1}}
	for _, c := range cases {
		if got, err := CompareVersions(c.a, c.b); err != nil || got != c.want {
			t.Errorf("CompareVersions(%s,%s) = %d, %v", c.a, c.b, got, err)
		}
	}
	for _, invalid := range []string{"1.0", "1.0.0-", "1.0.0-beta..1", "1.0.0-01", "v1.0.0", "1.0.0+build"} {
		if _, err := CompareVersions(invalid, "1.0.0"); err == nil {
			t.Errorf("invalid version %q accepted", invalid)
		}
	}
}

func TestSignAndVerify(t *testing.T) {
	publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	data := []byte("exact bytes")
	signature := Sign(privateKey, data)
	if err := Verify(publicKey, data, signature); err != nil {
		t.Fatal(err)
	}
	if Verify(publicKey, []byte("exact bytes "), signature) == nil {
		t.Error("changed bytes verified")
	}
	otherKey, _, _ := ed25519.GenerateKey(rand.Reader)
	if Verify(otherKey, data, signature) == nil {
		t.Error("other key verified")
	}
	if Verify(publicKey, data, "not base64!") == nil {
		t.Error("garbage signature verified")
	}
}

func TestKeyTextRoundTrip(t *testing.T) {
	publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	parsedPublic, err := ParsePublicKey(FormatPublicKey(publicKey))
	if err != nil || !bytes.Equal(parsedPublic, publicKey) {
		t.Fatalf("public key round trip: %v", err)
	}
	parsedPrivate, err := ParsePrivateKey(FormatPrivateKey(privateKey))
	if err != nil || !bytes.Equal(parsedPrivate, privateKey) {
		t.Fatalf("private key round trip: %v", err)
	}
}

func TestPlaceholderKeyIsRefused(t *testing.T) {
	for _, text := range []string{"", "  \n", "placeholder\n", "PLACEHOLDER: run keygen"} {
		if _, err := ParsePublicKey(text); !errors.Is(err, ErrNoPublicKey) {
			t.Errorf("ParsePublicKey(%q) = %v", text, err)
		}
	}
	if _, err := ParsePublicKey("AAAA"); err == nil || errors.Is(err, ErrNoPublicKey) {
		t.Errorf("short key: %v", err)
	}
}

func TestEmbeddedKeyEqualsSpecFile(t *testing.T) {
	spec, err := os.ReadFile(filepath.Join("..", "..", "spec", "release-ed25519.pub"))
	if err != nil {
		t.Fatal(err)
	}
	if string(spec) != embeddedPublicKey {
		t.Fatal("internal/release/release-ed25519.pub differs from spec/release-ed25519.pub; run `make sync-embedded`")
	}
}

func TestZipRoundTripAndTamperDetection(t *testing.T) {
	dist := sampleDist(t)
	manifest, err := BuildManifest(dist, "1.0.0", "0.1.0", time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	zipPath := filepath.Join(t.TempDir(), "release.zip")
	var buffer bytes.Buffer
	if err := WriteZip(dist, manifest, &buffer); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(zipPath, buffer.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := VerifyZip(manifest, zipPath); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "out", "appsctl")
	if err := ExtractFile(manifest, zipPath, ".adds/koboapps/bin/appsctl", destination); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(destination); string(got) != "binary" {
		t.Fatalf("extracted %q", got)
	}
	tampered := *manifest
	tampered.Files = append([]File(nil), manifest.Files...)
	tampered.Files[0].SHA256 = strings.Repeat("0", 64)
	if VerifyZip(&tampered, zipPath) == nil {
		t.Error("sha mismatch accepted")
	}
	if err := ExtractFile(&tampered, zipPath, tampered.Files[0].Path, destination+"2"); err == nil {
		t.Error("extract accepted sha mismatch")
	} else if _, statErr := os.Stat(destination + "2"); statErr == nil {
		t.Error("mismatching file left behind")
	}
	tampered.Files = tampered.Files[1:]
	if VerifyZip(&tampered, zipPath) == nil {
		t.Error("unlisted zip entry accepted")
	}
}

func writeDemoAppTree(t *testing.T, treeRoot string, extra map[string]string) {
	t.Helper()
	files := map[string]string{
		".adds/apps/demo/app.json":        `{"id":"demo","version":"1.2.3"}`,
		".adds/apps/demo/bin/run.sh":      "run",
		".adds/apps/demo/locales/en.json": "{}",
		".adds/apps/demo/data/token":      "secret",
		".adds/apps/demo/bin/._run.sh":    "appledouble",
	}
	for name, content := range extra {
		files[name] = content
	}
	writeTree(t, treeRoot, files)
}

func TestPackAppIsDeterministicAndSkipsNothingElse(t *testing.T) {
	tree := t.TempDir()
	writeDemoAppTree(t, tree, nil)
	// data/ in the staged tree is a mistake the packer must report, not hide.
	if _, err := PackApp(tree, "demo", t.TempDir()); err == nil {
		t.Fatal("a data/ folder in the staged app was packed")
	}
	if err := os.RemoveAll(filepath.Join(tree, ".adds/apps/demo/data")); err != nil {
		t.Fatal(err)
	}
	first, err := PackApp(tree, "demo", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	second, err := PackApp(tree, "demo", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(first) != "app-demo-1.2.3.zip" {
		t.Fatalf("name = %s", first)
	}
	firstHash, _ := HashFile(first)
	secondHash, _ := HashFile(second)
	if firstHash != secondHash {
		t.Fatal("the same sources gave different zips")
	}
}

func TestAppPackagesAreIndexedAndVerified(t *testing.T) {
	tree := t.TempDir()
	writeDemoAppTree(t, tree, nil)
	os.RemoveAll(filepath.Join(tree, ".adds/apps/demo/data"))
	appsDir := t.TempDir()
	if _, err := PackApp(tree, "demo", appsDir); err != nil {
		t.Fatal(err)
	}
	manifest, err := BuildManifest(sampleDist(t), "1.0.0", "0.1.0", time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	if err := manifest.AddAppPackages(appsDir); err != nil {
		t.Fatal(err)
	}
	if len(manifest.AppPackages) != 1 || manifest.AppPackages[0].ID != "demo" || manifest.AppPackages[0].Version != "1.2.3" ||
		manifest.AppPackages[0].Zip != "app-demo-1.2.3.zip" || manifest.AppPackages[0].Size <= 0 {
		t.Fatalf("app packages = %+v", manifest.AppPackages)
	}
	if err := VerifyAppPackages(manifest, appsDir); err != nil {
		t.Fatal(err)
	}
	zipPath := filepath.Join(appsDir, "app-demo-1.2.3.zip")
	data, _ := os.ReadFile(zipPath)
	data[len(data)/2] ^= 0xff
	os.WriteFile(zipPath, data, 0o644)
	if VerifyAppPackages(manifest, appsDir) == nil {
		t.Error("a tampered app zip verified")
	}
	os.Remove(zipPath)
	if VerifyAppPackages(manifest, appsDir) == nil {
		t.Error("a missing app zip verified")
	}
}

func TestAppZipEntriesStayInsideTheirApp(t *testing.T) {
	good := []string{".adds/apps/demo/app.json", ".adds/apps/demo/config.example.json", ".adds/apps/demo/screens/a.json"}
	bad := []string{".adds/apps/other/app.json", ".adds/apps/demo/data/token", ".adds/koboapps/bin/appsctl",
		".adds/apps/demo/../../koboapps/x", "/.adds/apps/demo/app.json", ".kobo/KoboRoot.tgz", ".adds/apps/demo"}
	for _, entry := range good {
		if err := CheckAppEntry("demo", entry); err != nil {
			t.Errorf("CheckAppEntry(%q) = %v", entry, err)
		}
	}
	for _, entry := range bad {
		if CheckAppEntry("demo", entry) == nil {
			t.Errorf("CheckAppEntry(%q) accepted", entry)
		}
	}
}

func TestAppPackageIndexRejectsBadEntries(t *testing.T) {
	manifest, err := BuildManifest(sampleDist(t), "1.0.0", "0.1.0", time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	manifest.AppPackages = []AppPackage{{ID: "demo", Version: "1.2.3", Zip: "app-demo-1.0.0.zip", SHA256: strings.Repeat("a", 64), Size: 10}}
	if manifest.Validate() == nil {
		t.Error("a zip name that does not match the version was accepted")
	}
	manifest.AppPackages = []AppPackage{
		{ID: "demo", Version: "1.2.3", Zip: "app-demo-1.2.3.zip", SHA256: strings.Repeat("a", 64), Size: 10},
		{ID: "demo", Version: "1.2.3", Zip: "app-demo-1.2.3.zip", SHA256: strings.Repeat("a", 64), Size: 10},
	}
	if manifest.Validate() == nil {
		t.Error("a duplicate app package was accepted")
	}
}
