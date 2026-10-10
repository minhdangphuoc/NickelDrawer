package release

import (
	"archive/zip"
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// zipFile is one entry of a hand-made test zip.
type zipFile struct {
	name string
	body string
	mode fs.FileMode
}

func makeZip(t *testing.T, files []zipFile) *zip.Reader {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, file := range files {
		header := &zip.FileHeader{Name: file.name, Method: zip.Deflate}
		header.SetMode(file.mode)
		entry, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		entry.Write([]byte(file.body))
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	reader, err := zip.NewReader(bytes.NewReader(buffer.Bytes()), int64(buffer.Len()))
	if err != nil {
		t.Fatal(err)
	}
	return reader
}

func goodAppFiles() []zipFile {
	return []zipFile{
		{".adds/apps/demo/app.json", `{"id":"demo","version":"1.0.0"}`, 0o644},
		{".adds/apps/demo/bin/run.sh", "run", 0o755},
	}
}

func TestCatalogSignatureIsDomainSeparatedFromTheDrawerRelease(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte(`{"v":1}`)
	if err := VerifyCatalog(publicKey, payload, SignCatalog(privateKey, payload)); err != nil {
		t.Fatalf("catalog signature rejected: %v", err)
	}
	if VerifyCatalog(publicKey, payload, Sign(privateKey, payload)) == nil {
		t.Error("a drawer (unprefixed) signature was accepted as a catalog signature")
	}
	if Verify(publicKey, payload, SignCatalog(privateKey, payload)) == nil {
		t.Error("a catalog signature was accepted as a drawer signature")
	}
	if VerifyCatalog(publicKey, append(payload, ' '), SignCatalog(privateKey, payload)) == nil {
		t.Error("a changed catalog verified")
	}
}

func TestCatalogLargerThanTheCapIsRefusedBeforeParsing(t *testing.T) {
	publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	huge := bytes.Repeat([]byte{' '}, MaxCatalogBytes+1)
	if VerifyCatalog(publicKey, huge, SignCatalog(privateKey, huge)) == nil {
		t.Error("an oversized catalog verified")
	}
	if _, err := ParseCatalog(huge); err == nil {
		t.Error("an oversized catalog parsed")
	}
}

func sampleCatalogApp() CatalogApp {
	return CatalogApp{ID: "todo", Version: "1.0.2", Tag: "todo-v1.0.2", Zip: "app-todo-1.0.2.zip",
		SHA256: strings.Repeat("a", 64), Size: 1000, Name: "To-do", Names: map[string]string{"vi": "Việc cần làm"}}
}

func TestCatalogRulesRejectBadEntries(t *testing.T) {
	cases := map[string]func(*CatalogApp){
		"wrong tag":      func(app *CatalogApp) { app.Tag = "todo-1.0.2" },
		"tag of another": func(app *CatalogApp) { app.Tag = "other-v1.0.2" },
		"wrong zip":      func(app *CatalogApp) { app.Zip = "../app-todo-1.0.2.zip" },
		"bad hash":       func(app *CatalogApp) { app.SHA256 = "XYZ" },
		"zero size":      func(app *CatalogApp) { app.Size = 0 },
		"huge size":      func(app *CatalogApp) { app.Size = MaxAppZipBytes + 1 },
		"empty name":     func(app *CatalogApp) { app.Name = " " },
		"control name":   func(app *CatalogApp) { app.Name = "a\x00b" },
		"bad language":   func(app *CatalogApp) { app.Names = map[string]string{"Vietnamese": "x"} },
		"bad constraint": func(app *CatalogApp) { app.Requires = &CatalogRequires{Drawer: "^1.0.0"} },
		"bad version":    func(app *CatalogApp) { app.Version = "1.0" },
	}
	for name, mutate := range cases {
		app := sampleCatalogApp()
		mutate(&app)
		catalog := &Catalog{V: 1, Sequence: 1, Issued: "2026-01-01T00:00:00Z", Apps: []CatalogApp{app}}
		if catalog.Validate() == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	good := &Catalog{V: 1, Sequence: 1, Issued: "2026-01-01T00:00:00Z", Apps: []CatalogApp{sampleCatalogApp()}}
	if err := good.Validate(); err != nil {
		t.Fatal(err)
	}
	good.Sequence = 0
	if good.Validate() == nil {
		t.Error("sequence 0 accepted")
	}
	good.Sequence, good.V = 1, 2
	if good.Validate() == nil {
		t.Error("unknown format version accepted")
	}
}

func TestDrawerRequirementAndDisplayName(t *testing.T) {
	app := sampleCatalogApp()
	if ok, err := app.MeetsDrawerRequirement("1.0.0"); !ok || err != nil {
		t.Fatalf("no requirement: ok=%v err=%v", ok, err)
	}
	app.Requires = &CatalogRequires{Drawer: ">=1.0.0"}
	for version, want := range map[string]bool{"1.0.0": true, "1.0.1": true, "2.0.0": true, "1.0.0-beta.5": false, "0.9.9": false} {
		if got, _ := app.MeetsDrawerRequirement(version); got != want {
			t.Errorf("drawer %s: got %v, want %v", version, got, want)
		}
	}
	if app.DisplayName("vi_VN") != "Việc cần làm" || app.DisplayName("vi") != "Việc cần làm" || app.DisplayName("de") != "To-do" {
		t.Errorf("names: vi_VN=%q de=%q", app.DisplayName("vi_VN"), app.DisplayName("de"))
	}
}

func TestBuildCatalogDescribesZipsWithNamesTagsAndRequirements(t *testing.T) {
	tree := t.TempDir()
	writeTree(t, tree, map[string]string{
		".adds/apps/demo/app.json":        `{"id":"demo","name":"Demo","version":"1.2.3","min_host":"1.0.0","hidden":true}`,
		".adds/apps/demo/locales/vi.json": `{"name":"Mẫu"}`,
		".adds/apps/demo/locales/en.json": `{"name":"Demo"}`,
		".adds/apps/demo/bin/run.sh":      "run",
	})
	appsDir := t.TempDir()
	if _, err := PackApp(tree, "demo", appsDir); err != nil {
		t.Fatal(err)
	}
	catalog, err := BuildCatalog(appsDir, 7, time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	app, found := catalog.Find("demo")
	if !found || app.Tag != "demo-v1.2.3" || app.Name != "Demo" || app.Names["vi"] != "Mẫu" || !app.Hidden ||
		app.Requires == nil || app.Requires.Drawer != ">=1.0.0" || catalog.Sequence != 7 {
		t.Fatalf("catalog = %+v", catalog)
	}
	if _, hasEnglish := app.Names["en"]; hasEnglish {
		t.Error("English is the name field, not a translation")
	}
	data, err := MarshalCatalog(catalog)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseCatalog(data); err != nil {
		t.Fatal(err)
	}
	zipPath := filepath.Join(appsDir, "app-demo-1.2.3.zip")
	if err := VerifyCatalogZip(app, zipPath); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(zipPath)
	raw[len(raw)/2] ^= 0xff
	os.WriteFile(zipPath, raw, 0o644)
	if VerifyCatalogZip(app, zipPath) == nil {
		t.Error("a tampered zip matched its catalog entry")
	}
}

func TestAppZipRulesRefuseEveryUnsafeShape(t *testing.T) {
	link := zipFile{".adds/apps/demo/bin/link", "/etc/passwd", fs.ModeSymlink | 0o777}
	cases := map[string][]zipFile{
		"symlink":            append(goodAppFiles(), link),
		"other app":          append(goodAppFiles(), zipFile{".adds/apps/other/app.json", "x", 0o644}),
		"parent traversal":   append(goodAppFiles(), zipFile{".adds/apps/demo/../../koboapps/bin/appsctl", "x", 0o644}),
		"absolute":           append(goodAppFiles(), zipFile{"/.adds/apps/demo/x", "x", 0o644}),
		"backslash":          append(goodAppFiles(), zipFile{`.adds/apps/demo\x`, "x", 0o644}),
		"data folder":        append(goodAppFiles(), zipFile{".adds/apps/demo/data/token", "x", 0o644}),
		"nested data":        append(goodAppFiles(), zipFile{".adds/apps/demo/bin/data/x", "x", 0o644}),
		"preview state":      append(goodAppFiles(), zipFile{".adds/apps/demo/preview-state.json", "{}", 0o644}),
		"outside apps":       append(goodAppFiles(), zipFile{".kobo/KoboRoot.tgz", "x", 0o644}),
		"case duplicate":     append(goodAppFiles(), zipFile{".adds/apps/demo/BIN/run.sh", "x", 0o644}),
		"exact duplicate":    append(goodAppFiles(), goodAppFiles()[1]),
		"decomposed unicode": append(goodAppFiles(), zipFile{".adds/apps/demo/café.txt", "x", 0o644}),
		"control character":  append(goodAppFiles(), zipFile{".adds/apps/demo/a\x01b", "x", 0o644}),
		"no manifest":        {{".adds/apps/demo/bin/run.sh", "run", 0o755}},
		"device":             append(goodAppFiles(), zipFile{".adds/apps/demo/dev", "", fs.ModeDevice | 0o644}),
	}
	for name, files := range cases {
		_, err := CheckAppZip(makeZip(t, files), "demo")
		if !errors.Is(err, ErrUnsafeZip) {
			t.Errorf("%s: err = %v, want ErrUnsafeZip", name, err)
		}
	}
	entries, err := CheckAppZip(makeZip(t, goodAppFiles()), "demo")
	if err != nil || len(entries) != 2 {
		t.Fatalf("good zip: entries=%v err=%v", entries, err)
	}
	for _, entry := range entries {
		wantMode := fs.FileMode(0o644)
		if entry.Name == "bin/run.sh" {
			wantMode = 0o755
		}
		if entry.ModeOf() != wantMode {
			t.Errorf("%s mode = %o, want %o", entry.Name, entry.ModeOf(), wantMode)
		}
	}
}

func TestAppZipRefusesTooManyEntriesAndTooMuchData(t *testing.T) {
	many := goodAppFiles()
	for i := 0; i <= maxAppZipEntries; i++ {
		many = append(many, zipFile{".adds/apps/demo/f" + strings.Repeat("x", 1) + string(rune('a'+i%26)) + string(rune('a'+i/26%26)) + string(rune('a'+i/676%26)), "", 0o644})
	}
	if _, err := CheckAppZip(makeZip(t, many), "demo"); !errors.Is(err, ErrUnsafeZip) {
		t.Errorf("too many entries: %v", err)
	}
	big := goodAppFiles()
	chunk := strings.Repeat("0", 20<<20)
	for _, name := range []string{"a", "b", "c", "d"} {
		big = append(big, zipFile{".adds/apps/demo/" + name, chunk, 0o644})
	}
	if _, err := CheckAppZip(makeZip(t, big), "demo"); !errors.Is(err, ErrUnsafeZip) {
		t.Errorf("too much unpacked data: %v", err)
	}
}
