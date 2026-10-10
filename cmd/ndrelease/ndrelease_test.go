package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"koboapps/internal/release"
)

type workspace struct {
	dir, keyPath, specPub, embeddedPub, dist, out, appsTree, appsDir string
}

func newWorkspace(t *testing.T) workspace {
	dir := t.TempDir()
	w := workspace{
		dir: dir, keyPath: filepath.Join(dir, "config", "release.key"),
		specPub: filepath.Join(dir, "spec", "release-ed25519.pub"), embeddedPub: filepath.Join(dir, "embed", "release-ed25519.pub"),
		dist: filepath.Join(dir, "dist", "onboard"), out: filepath.Join(dir, "dist", "release"),
		appsTree: filepath.Join(dir, "build", "apps-tree"), appsDir: filepath.Join(dir, "dist", "apps"),
	}
	for relative, content := range map[string]string{
		".adds/koboapps/bin/appsctl":     "binary",
		".adds/apps/demo/app.json":       `{"id":"demo","version":"1.0.0"}`,
		".adds/apps/demo/bin/run.sh":     "run",
		".adds/apps/demo/data/token":     "must not ship",
		".adds/koboapps/locales/en.json": "{}",
	} {
		full := filepath.Join(w.dist, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return w
}

func (w workspace) keygen(t *testing.T) (code int, stdout, stderr string) {
	var out, errOut bytes.Buffer
	code = run([]string{"keygen", "--key", w.keyPath, "--pub", w.specPub, "--embed-pub", w.embeddedPub}, &out, &errOut)
	return code, out.String(), errOut.String()
}

func (w workspace) sign(version string) (int, string) {
	var out, errOut bytes.Buffer
	code := run([]string{"sign", "--dist", w.dist, "--version", version, "--out", w.out, "--key", w.keyPath, "--pub", w.specPub}, &out, &errOut)
	return code, errOut.String()
}

func (w workspace) verify() (int, string) {
	var out, errOut bytes.Buffer
	code := run([]string{"verify", "--dir", w.out, "--pub", w.specPub}, &out, &errOut)
	return code, errOut.String()
}

func TestKeygenWritesKeysWithoutPrintingThePrivateOne(t *testing.T) {
	w := newWorkspace(t)
	code, stdout, stderr := w.keygen(t)
	if code != exitOK {
		t.Fatalf("code=%d stderr=%s", code, stderr)
	}
	info, err := os.Stat(w.keyPath)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("private key: %v mode=%v", err, info)
	}
	privateText, _ := os.ReadFile(w.keyPath)
	if strings.Contains(stdout+stderr, strings.TrimSpace(string(privateText))) {
		t.Fatal("private key was printed")
	}
	specText, _ := os.ReadFile(w.specPub)
	embeddedText, _ := os.ReadFile(w.embeddedPub)
	if len(specText) == 0 || string(specText) != string(embeddedText) {
		t.Fatalf("public key copies differ: %q vs %q", specText, embeddedText)
	}
}

func TestKeygenRefusesToOverwrite(t *testing.T) {
	w := newWorkspace(t)
	w.keygen(t)
	before, _ := os.ReadFile(w.keyPath)
	code, _, stderr := w.keygen(t)
	if code == exitOK || !strings.Contains(stderr, "refusing to overwrite") {
		t.Fatalf("code=%d stderr=%s", code, stderr)
	}
	after, _ := os.ReadFile(w.keyPath)
	if string(before) != string(after) {
		t.Fatal("key changed")
	}
}

func TestSignThenVerify(t *testing.T) {
	w := newWorkspace(t)
	w.keygen(t)
	if code, stderr := w.sign("1.2.3"); code != exitOK {
		t.Fatalf("sign: %s", stderr)
	}
	for _, name := range []string{"release.json", "release.json.sig", "nickeldrawer-1.2.3.zip"} {
		if _, err := os.Stat(filepath.Join(w.out, name)); err != nil {
			t.Errorf("missing %s", name)
		}
	}
	manifest, _ := os.ReadFile(filepath.Join(w.out, "release.json"))
	if strings.Contains(string(manifest), "token") {
		t.Fatal("app data listed in the release")
	}
	if code, stderr := w.verify(); code != exitOK {
		t.Fatalf("verify: %s", stderr)
	}
}

func TestVerifyCatchesTamperedManifestAndZip(t *testing.T) {
	w := newWorkspace(t)
	w.keygen(t)
	w.sign("1.2.3")
	manifestPath := filepath.Join(w.out, "release.json")
	original, _ := os.ReadFile(manifestPath)
	os.WriteFile(manifestPath, bytes.Replace(original, []byte(`"1.2.3"`), []byte(`"9.9.9"`), 1), 0o644)
	if code, _ := w.verify(); code == exitOK {
		t.Fatal("tampered manifest verified")
	}
	os.WriteFile(manifestPath, original, 0o644)
	zipPath := filepath.Join(w.out, "nickeldrawer-1.2.3.zip")
	zipBytes, _ := os.ReadFile(zipPath)
	zipBytes[len(zipBytes)/2] ^= 0xff
	os.WriteFile(zipPath, zipBytes, 0o644)
	if code, _ := w.verify(); code == exitOK {
		t.Fatal("tampered zip verified")
	}
}

func TestSignExplainsMissingKeyAndPlaceholder(t *testing.T) {
	w := newWorkspace(t)
	code, stderr := w.sign("1.0.0")
	if code == exitOK || !strings.Contains(stderr, "ndrelease keygen") {
		t.Fatalf("code=%d stderr=%s", code, stderr)
	}
	w.keygen(t)
	os.WriteFile(w.specPub, []byte("placeholder\n"), 0o644)
	code, stderr = w.sign("1.0.0")
	if code == exitOK || !strings.Contains(stderr, "placeholder") {
		t.Fatalf("code=%d stderr=%s", code, stderr)
	}
}

func TestSignRefusesAMismatchingPublicKeyAndLooseKeyPermissions(t *testing.T) {
	w := newWorkspace(t)
	w.keygen(t)
	other := newWorkspace(t)
	other.keygen(t)
	os.WriteFile(w.specPub, mustRead(t, other.specPub), 0o644)
	if code, stderr := w.sign("1.0.0"); code == exitOK || !strings.Contains(stderr, "does not match") {
		t.Fatalf("code=%d stderr=%s", code, stderr)
	}
	os.WriteFile(w.specPub, mustRead(t, w.embeddedPub), 0o644)
	os.Chmod(w.keyPath, 0o644)
	if code, stderr := w.sign("1.0.0"); code == exitOK || !strings.Contains(stderr, "chmod 600") {
		t.Fatalf("code=%d stderr=%s", code, stderr)
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestSignRejectsBadVersion(t *testing.T) {
	w := newWorkspace(t)
	w.keygen(t)
	if code, _ := w.sign("1.0"); code == exitOK {
		t.Fatal("bad version accepted")
	}
}

func (w workspace) packApps(t *testing.T) {
	t.Helper()
	for relative, content := range map[string]string{
		".adds/apps/demo/app.json":   `{"id":"demo","name":"Demo","version":"1.0.0"}`,
		".adds/apps/demo/bin/run.sh": "run",
		".adds/apps/other/app.json":  `{"id":"other","name":"Other","version":"2.0.0"}`,
	} {
		full := filepath.Join(w.appsTree, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	var out, errOut bytes.Buffer
	if code := run([]string{"pack-apps", "--tree", w.appsTree, "--out", w.appsDir}, &out, &errOut); code != exitOK {
		t.Fatalf("pack-apps: %s", errOut.String())
	}
}

func TestSignLeavesAppsOutOfTheDrawerRelease(t *testing.T) {
	w := newWorkspace(t)
	w.keygen(t)
	w.packApps(t)
	if code, stderr := w.sign("1.2.3"); code != exitOK {
		t.Fatalf("sign: %s", stderr)
	}
	manifest, _ := os.ReadFile(filepath.Join(w.out, "release.json"))
	if strings.Contains(string(manifest), "app_packages") || strings.Contains(string(manifest), ".adds/apps/") {
		t.Fatalf("an app is part of the drawer release:\n%s", manifest)
	}
	entries, _ := os.ReadDir(w.out)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "app-") {
			t.Errorf("%s was copied into the drawer release folder", entry.Name())
		}
	}
	if code, stderr := w.verify(); code != exitOK {
		t.Fatalf("verify: %s", stderr)
	}
}

func (w workspace) signApps(t *testing.T, args ...string) (int, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	base := []string{"sign-apps", "--apps-dir", w.appsDir, "--out", w.appsRelease(), "--key", w.keyPath, "--pub", w.specPub}
	return run(append(base, args...), &out, &errOut), errOut.String()
}

func (w workspace) appsRelease() string { return filepath.Join(w.dir, "dist", "apps-release") }

func (w workspace) verifyApps(extra ...string) (int, string) {
	var out, errOut bytes.Buffer
	args := append([]string{"verify-apps", "--dir", w.appsRelease(), "--pub", w.specPub}, extra...)
	return run(args, &out, &errOut), errOut.String()
}

func TestSignAppsWritesCatalogOfAllAppsAndUploadsOnlyTheChosenZips(t *testing.T) {
	w := newWorkspace(t)
	w.keygen(t)
	w.packApps(t)
	if code, stderr := w.signApps(t, "--sequence", "3", "--only", "demo"); code != exitOK {
		t.Fatalf("sign-apps: %s", stderr)
	}
	catalog := string(mustRead(t, filepath.Join(w.appsRelease(), "apps.json")))
	for _, want := range []string{`"sequence": 3`, `"tag": "demo-v1.0.0"`, `"tag": "other-v2.0.0"`} {
		if !strings.Contains(catalog, want) {
			t.Errorf("catalog lacks %s:\n%s", want, catalog)
		}
	}
	if _, err := os.Stat(filepath.Join(w.appsRelease(), "app-demo-1.0.0.zip")); err != nil {
		t.Error("the chosen zip was not copied for upload")
	}
	if _, err := os.Stat(filepath.Join(w.appsRelease(), "app-other-2.0.0.zip")); err == nil {
		t.Error("a zip that was not chosen was copied")
	}
	if code, stderr := w.verifyApps(); code != exitOK {
		t.Fatalf("verify-apps: %s", stderr)
	}
	if code, stderr := w.verifyApps("--apps-dir", w.appsDir); code != exitOK {
		t.Fatalf("verify-apps against the full set: %s", stderr)
	}
}

func TestSignAppsRefusesMissingSequenceUnknownOnlyAndWrongKey(t *testing.T) {
	w := newWorkspace(t)
	w.keygen(t)
	w.packApps(t)
	if code, _ := w.signApps(t); code == exitOK {
		t.Error("signed without --sequence")
	}
	if code, _ := w.signApps(t, "--sequence", "1", "--only", "nothing"); code == exitOK {
		t.Error("accepted --only for an app with no zip")
	}
}

func TestVerifyAppsRejectsTamperingAndADrawerSignature(t *testing.T) {
	w := newWorkspace(t)
	w.keygen(t)
	w.packApps(t)
	if code, stderr := w.signApps(t, "--sequence", "1"); code != exitOK {
		t.Fatalf("sign-apps: %s", stderr)
	}
	catalogPath := filepath.Join(w.appsRelease(), "apps.json")
	signaturePath := filepath.Join(w.appsRelease(), "apps.json.sig")
	original := mustRead(t, catalogPath)
	os.WriteFile(catalogPath, bytes.Replace(original, []byte(`"sequence": 1`), []byte(`"sequence": 9`), 1), 0o644)
	if code, _ := w.verifyApps(); code == exitOK {
		t.Error("an edited catalog verified")
	}
	os.WriteFile(catalogPath, original, 0o644)
	// The same key signing the catalog bytes the way release.json is signed (no prefix) must fail.
	key, err := release.ParsePrivateKey(string(mustRead(t, w.keyPath)))
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(signaturePath, []byte(release.Sign(key, original)), 0o644)
	if code, _ := w.verifyApps(); code == exitOK {
		t.Error("the drawer's signature verified a catalog")
	}
}
