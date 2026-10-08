package inbox

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"koboapps/internal/storage"
)

const shippedRules = "../../host/config/routing.json"

// fakeKobo is a temp /mnt/onboard with the shipped rules installed.
type fakeKobo struct {
	t      *testing.T
	root   string
	router *Router
}

func newFakeKobo(t *testing.T) *fakeKobo {
	t.Helper()
	root := t.TempDir()
	t.Cleanup(storage.SetRoot(root))
	rules, overrideErr, err := LoadRules(shippedRules, filepath.Join(root, "none.json"))
	if err != nil || overrideErr != nil {
		t.Fatalf("shipped rules: %v / %v", err, overrideErr)
	}
	kobo := &fakeKobo{t: t, root: root, router: NewRouter(rules, JournalPath())}
	must(t, os.MkdirAll(storage.Path(storage.Inbox), 0o755))
	return kobo
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func (kobo *fakeKobo) put(folder, name string, content []byte) string {
	kobo.t.Helper()
	path := filepath.Join(folder, name)
	must(kobo.t, os.MkdirAll(folder, 0o755))
	must(kobo.t, os.WriteFile(path, content, 0o644))
	return path
}

func (kobo *fakeKobo) inbox(name string, content []byte) string {
	kobo.t.Helper()
	return kobo.put(storage.Path(storage.Inbox), name, content)
}

func (kobo *fakeKobo) route(path string) Outcome {
	kobo.t.Helper()
	outcome, err := kobo.router.Route(path, "test")
	if err != nil {
		kobo.t.Fatalf("Route(%s): %v", path, err)
	}
	return outcome
}

func (kobo *fakeKobo) listing(folder storage.Folder) []string {
	kobo.t.Helper()
	entries, err := os.ReadDir(storage.Path(folder))
	if os.IsNotExist(err) {
		return nil
	}
	must(kobo.t, err)
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	return names
}

func epubBytes(t *testing.T, mimetype string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	header := &zip.FileHeader{Name: "mimetype", Method: zip.Store}
	entry, err := archive.CreateHeader(header)
	must(t, err)
	_, err = entry.Write([]byte(mimetype))
	must(t, err)
	other, err := archive.Create("OEBPS/content.opf")
	must(t, err)
	_, err = other.Write([]byte("<package/>"))
	must(t, err)
	must(t, archive.Close())
	return buffer.Bytes()
}

func plainZip(t *testing.T) []byte {
	t.Helper()
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	entry, err := archive.Create("page1.jpg")
	must(t, err)
	_, err = entry.Write([]byte("x"))
	must(t, err)
	must(t, archive.Close())
	return buffer.Bytes()
}

func mobiBytes() []byte {
	content := make([]byte, 100)
	copy(content[60:], "BOOKMOBI")
	return content
}

func sampleContent(t *testing.T) map[string][]byte {
	return map[string][]byte{
		"epub": epubBytes(t, "application/epub+zip"),
		"mobi": mobiBytes(),
		"pdf":  []byte("%PDF-1.7\n..."),
		"txt":  []byte("Xin chào, Việt Nam\n"),
		"html": []byte("\n <html><body>hi</body></html>"),
		"rtf":  []byte(`{\rtf1\ansi hello}`),
		"cbz":  plainZip(t),
		"cbr4": []byte("Rar!\x1a\x07\x00rest"),
		"cbr5": []byte("Rar!\x1a\x07\x01\x00rest"),
		"png":  []byte("\x89PNG\r\n\x1a\nrest"),
		"jpeg": []byte{0xFF, 0xD8, 0xFF, 0xE0, 0, 0},
		"gif":  []byte("GIF89a......"),
	}
}

func TestShippedRulesCoverTheArchitectureAllowlist(t *testing.T) {
	rules, overrideErr, err := LoadRules(shippedRules, "/no/such/override.json")
	if err != nil || overrideErr != nil {
		t.Fatalf("LoadRules: %v / %v", err, overrideErr)
	}
	want := map[string]storage.Folder{
		"a.epub": storage.Books, "a.kepub.epub": storage.Books, "a.mobi": storage.Books,
		"a.pdf": storage.Documents, "a.txt": storage.Documents, "a.html": storage.Documents,
		"a.htm": storage.Documents, "a.rtf": storage.Documents,
		"a.cbz": storage.Comics, "a.cbr": storage.Comics,
		"a.png": storage.Pictures, "a.jpg": storage.Pictures, "a.jpeg": storage.Pictures, "a.gif": storage.Pictures,
	}
	for name, folder := range want {
		rule, _, found := rules.Match(name)
		if !found || rule.Destination != folder {
			t.Errorf("%s -> %v (found %v), want %s", name, rule.Destination, found, folder)
		}
	}
	if _, _, found := rules.Match("a.exe"); found {
		t.Error(".exe must not match")
	}
	if rule, ext, _ := rules.Match("Book.KEPUB.EPUB"); ext != ".kepub.epub" || rule.Type != "epub" {
		t.Errorf("longest extension must win, got %q", ext)
	}
}

func TestRoutesEachAllowedTypeToItsFolder(t *testing.T) {
	kobo := newFakeKobo(t)
	content := sampleContent(t)
	cases := []struct {
		name     string
		data     []byte
		folder   storage.Folder
		wantType string
	}{
		{"a.epub", content["epub"], storage.Books, "epub"},
		{"b.kepub.epub", content["epub"], storage.Books, "epub"},
		{"c.mobi", content["mobi"], storage.Books, "mobi"},
		{"d.pdf", content["pdf"], storage.Documents, "pdf"},
		{"e.txt", content["txt"], storage.Documents, "txt"},
		{"f.html", content["html"], storage.Documents, "html"},
		{"g.htm", content["html"], storage.Documents, "html"},
		{"h.rtf", content["rtf"], storage.Documents, "rtf"},
		{"i.cbz", content["cbz"], storage.Comics, "cbz"},
		{"j.cbr", content["cbr4"], storage.Comics, "cbr"},
		{"k.cbr", content["cbr5"], storage.Comics, "cbr"},
		{"l.png", content["png"], storage.Pictures, "png"},
		{"m.jpg", content["jpeg"], storage.Pictures, "jpeg"},
		{"n.JPEG", content["jpeg"], storage.Pictures, "jpeg"},
		{"o.gif", content["gif"], storage.Pictures, "gif"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			outcome := kobo.route(kobo.inbox(c.name, c.data))
			if outcome.Destination != c.folder || outcome.Type != c.wantType || outcome.IsRejected() {
				t.Fatalf("outcome = %+v", outcome)
			}
			got, err := os.ReadFile(outcome.Path)
			if err != nil || !bytes.Equal(got, c.data) {
				t.Fatalf("stored content differs: %v", err)
			}
			if filepath.Dir(outcome.Path) != storage.Path(c.folder) {
				t.Errorf("stored in %s", outcome.Path)
			}
		})
	}
	if left := kobo.listing(storage.Inbox); len(left) != 0 {
		t.Errorf("inbox must be empty, has %v", left)
	}
}

func TestRejectsExtensionContentMismatch(t *testing.T) {
	kobo := newFakeKobo(t)
	content := sampleContent(t)
	deflatedEpub := epubBytes(t, "application/x-something-else")
	cases := []struct {
		name string
		data []byte
	}{
		{"fake.pdf", content["cbz"]},
		{"plain.epub", content["cbz"]},
		{"wrongmime.epub", deflatedEpub},
		{"short.epub", []byte("PK\x03\x04")},
		{"fake.mobi", []byte("not a mobi at all")},
		{"nul.txt", []byte("hello\x00world")},
		{"latin1.txt", []byte("caf\xe9")},
		{"cut.txt", []byte("ok \xe1\xbb")},
		{"png.jpg", content["png"]},
		{"jpg.png", content["jpeg"]},
		{"zip.cbr", content["cbz"]},
		{"rar.cbz", content["cbr4"]},
		{"text.html", []byte("just text")},
		{"text.rtf", []byte("just text")},
		{"gif.gif", []byte("GIF90a")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			outcome := kobo.route(kobo.inbox(c.name, c.data))
			if !outcome.IsRejected() || outcome.Reason != ReasonContentMismatch {
				t.Fatalf("outcome = %+v", outcome)
			}
			if _, err := os.Stat(filepath.Join(storage.Path(storage.Rejected), outcome.StoredAs)); err != nil {
				t.Errorf("rejected file missing: %v", err)
			}
		})
	}
	for _, folder := range []storage.Folder{storage.Books, storage.Documents, storage.Comics, storage.Pictures} {
		if got := kobo.listing(folder); len(got) != 0 {
			t.Errorf("%s must stay empty, has %v", folder, got)
		}
	}
}

func TestRejectsDisallowedTypes(t *testing.T) {
	kobo := newFakeKobo(t)
	for _, name := range []string{"virus.exe", "run.sh", "archive.zip", "noextension", "app.apk", "x.epub.exe", "x.pdf.sh", "readme"} {
		outcome := kobo.route(kobo.inbox(name, []byte("%PDF-1.4 whatever")))
		if !outcome.IsRejected() || outcome.Reason != ReasonTypeNotAllowed {
			t.Errorf("%s: outcome = %+v", name, outcome)
		}
	}
	if got := len(kobo.listing(storage.Rejected)); got != 8 {
		t.Errorf("rejected has %d files, want 8", got)
	}
}

func TestEnforcesSizeCapAndRejectsEmptyFiles(t *testing.T) {
	kobo := newFakeKobo(t)
	for index := range kobo.router.rules.Rules {
		if kobo.router.rules.Rules[index].Type == "pdf" {
			kobo.router.rules.Rules[index].MaxBytes = 20
		}
	}
	ok := kobo.route(kobo.inbox("small.pdf", []byte("%PDF-1.7 12345")))
	big := kobo.route(kobo.inbox("big.pdf", []byte("%PDF-1.7 "+strings.Repeat("x", 100))))
	empty := kobo.route(kobo.inbox("empty.pdf", nil))
	if ok.IsRejected() || big.Reason != ReasonTooLarge || empty.Reason != ReasonEmpty {
		t.Fatalf("ok=%+v big=%+v empty=%+v", ok, big, empty)
	}
}

func TestRefusesSymlinksAndNonRegularFiles(t *testing.T) {
	kobo := newFakeKobo(t)
	secret := filepath.Join(t.TempDir(), "secret.pdf")
	must(t, os.WriteFile(secret, []byte("%PDF-1.7 secret"), 0o644))
	link := filepath.Join(storage.Path(storage.Inbox), "link.pdf")
	must(t, os.Symlink(secret, link))
	fifo := filepath.Join(storage.Path(storage.Inbox), "pipe.pdf")
	must(t, syscall.Mkfifo(fifo, 0o644))

	linkOutcome := kobo.route(link)
	fifoOutcome := kobo.route(fifo)

	if linkOutcome.Reason != ReasonSymlink || fifoOutcome.Reason != ReasonNotRegular {
		t.Fatalf("link=%+v fifo=%+v", linkOutcome, fifoOutcome)
	}
	if got := kobo.listing(storage.Documents); len(got) != 0 {
		t.Errorf("nothing may reach documents, got %v", got)
	}
	if content, err := os.ReadFile(secret); err != nil || string(content) != "%PDF-1.7 secret" {
		t.Errorf("link target was touched: %q %v", content, err)
	}
	if _, err := os.Stat(secret); err != nil {
		t.Error("link target must still exist")
	}
}

func TestRefusesPathsOutsideTheAllowedSources(t *testing.T) {
	kobo := newFakeKobo(t)
	elsewhere := t.TempDir()
	outside := filepath.Join(elsewhere, "a.pdf")
	must(t, os.WriteFile(outside, []byte("%PDF-1.7 x"), 0o644))
	must(t, os.MkdirAll(storage.Path(storage.Books), 0o755))
	inLibrary := kobo.put(storage.Path(storage.Books), "b.pdf", []byte("%PDF-1.7 x"))
	inRejected := kobo.put(storage.Path(storage.Rejected), "c.pdf", []byte("%PDF-1.7 x"))
	traversal := filepath.Join(storage.Path(storage.Inbox), "..", "..", "..", "Books", "b.pdf")

	for _, path := range []string{outside, inLibrary, inRejected, traversal, "<clipboard>", "", storage.Path(storage.Inbox)} {
		if _, err := kobo.router.Route(path, "test"); err == nil {
			t.Errorf("Route(%q) must fail", path)
		}
	}
	if _, err := os.Stat(outside); err != nil {
		t.Error("a file outside the sources must not move")
	}
	if _, err := kobo.router.Route(filepath.Join(storage.Path(storage.Inbox), "rejected"), "t"); !errors.Is(err, ErrIsFolder) {
		t.Errorf("a folder must give ErrIsFolder, got %v", err)
	}
}

func TestSanitizeName(t *testing.T) {
	decomposed := "Việt Nam.epub" // "Việt" typed as base letters plus combining marks
	long := strings.Repeat("ệ", 200) + ".kepub.epub"
	cases := []struct{ name, extension, want string }{
		{decomposed, ".epub", "Việt Nam.epub"},
		{"Truyện Kiều.pdf", ".pdf", "Truyện Kiều.pdf"},
		{"../../etc/passwd.pdf", ".pdf", "etcpasswd.pdf"},
		{`..\..\evil.pdf`, ".pdf", "evil.pdf"},
		{"a/b/c.txt", ".txt", "abc.txt"},
		{".hidden.pdf", ".pdf", "hidden.pdf"},
		{"...pdf", ".pdf", "file.pdf"},
		{".pdf", ".pdf", "file.pdf"},
		{"  spaced  .pdf", ".pdf", "spaced.pdf"},
		{"tab\tnew\nline\x00.txt", ".txt", "tabnewline.txt"},
		{"what?*<>|:\".txt", ".txt", "what_______.txt"},
		{"trailing dots...", "", "trailing dots"},
		{"BOOK.PDF", ".pdf", "BOOK.pdf"},
		{"x.KEPUB.epub", ".kepub.epub", "x.kepub.epub"},
		{"bad\xffbyte.pdf", ".pdf", "bad_byte.pdf"},
		{"", "", "file"},
		{"noext", "", "noext"},
	}
	for _, c := range cases {
		if got := SanitizeName(c.name, c.extension); got != c.want {
			t.Errorf("SanitizeName(%q, %q) = %q, want %q", c.name, c.extension, got, c.want)
		}
	}
	got := SanitizeName(long, ".kepub.epub")
	if len(got) > maxNameBytes || !strings.HasSuffix(got, ".kepub.epub") || !strings.HasPrefix(got, "ệ") {
		t.Errorf("long name = %q (%d bytes)", got, len(got))
	}
	if !strings.HasPrefix(got, strings.Repeat("ệ", 10)) || strings.ContainsRune(got, '�') {
		t.Errorf("truncation must not split characters: %q", got)
	}
}

func TestRoutedFileGetsTheSanitizedName(t *testing.T) {
	kobo := newFakeKobo(t)
	outcome := kobo.route(kobo.inbox(".Việt\\Nam.EPUB", epubBytes(t, "application/epub+zip")))
	if outcome.StoredAs != "ViệtNam.epub" {
		t.Errorf("stored as %q", outcome.StoredAs)
	}
}

func TestCollisionsNeverOverwrite(t *testing.T) {
	kobo := newFakeKobo(t)
	var stored []string
	for index := 1; index <= 3; index++ {
		content := []byte(fmt.Sprintf("%%PDF-1.7 copy %d", index))
		stored = append(stored, kobo.route(kobo.inbox("same.pdf", content)).StoredAs)
	}
	for index := 1; index <= 2; index++ {
		stored = append(stored, kobo.route(kobo.inbox("k.kepub.epub", epubBytes(t, "application/epub+zip"))).StoredAs)
	}
	want := "same.pdf,same (2).pdf,same (3).pdf,k.kepub.epub,k (2).kepub.epub"
	if strings.Join(stored, ",") != want {
		t.Fatalf("stored %v, want %s", stored, want)
	}
	for index, name := range []string{"same.pdf", "same (2).pdf", "same (3).pdf"} {
		got, _ := os.ReadFile(filepath.Join(storage.Path(storage.Documents), name))
		if string(got) != fmt.Sprintf("%%PDF-1.7 copy %d", index+1) {
			t.Errorf("%s holds %q", name, got)
		}
	}
}

func TestExistingLibraryFileIsNeverReplaced(t *testing.T) {
	kobo := newFakeKobo(t)
	kobo.put(storage.Path(storage.Documents), "mine.pdf", []byte("%PDF-1.7 original"))
	outcome := kobo.route(kobo.inbox("mine.pdf", []byte("%PDF-1.7 incoming")))
	original, _ := os.ReadFile(filepath.Join(storage.Path(storage.Documents), "mine.pdf"))
	if string(original) != "%PDF-1.7 original" || outcome.StoredAs != "mine (2).pdf" {
		t.Errorf("original=%q stored=%q", original, outcome.StoredAs)
	}
}

func TestConcurrentRoutesLoseNothingAndKeepTheJournalIntact(t *testing.T) {
	kobo := newFakeKobo(t)
	const count = 20
	// Distinct names, and names that all sanitize to "same.pdf" (leading spaces are stripped).
	var paths []string
	for index := 0; index < count; index++ {
		paths = append(paths, kobo.inbox(fmt.Sprintf("distinct-%d.pdf", index), []byte(fmt.Sprintf("%%PDF-1.7 distinct %d", index))))
		paths = append(paths, kobo.inbox(strings.Repeat(" ", index)+"same.pdf", []byte(fmt.Sprintf("%%PDF-1.7 same %d", index))))
	}
	var group sync.WaitGroup
	errs := make(chan error, len(paths))
	for _, path := range paths {
		group.Add(1)
		go func() {
			defer group.Done()
			_, err := kobo.router.Route(path, "race")
			errs <- err
		}()
	}
	group.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Error(err)
		}
	}

	seen := map[string]bool{}
	for _, name := range kobo.listing(storage.Documents) {
		content, err := os.ReadFile(filepath.Join(storage.Path(storage.Documents), name))
		must(t, err)
		seen[string(content)] = true
	}
	if len(seen) != 2*count {
		t.Errorf("%d distinct documents arrived, want %d", len(seen), 2*count)
	}
	entries, skipped, err := ReadEntries(JournalPath())
	if err != nil || skipped != 0 || len(entries) != 2*count {
		t.Errorf("journal: %d entries, %d corrupt lines, err %v", len(entries), skipped, err)
	}
	if left := kobo.listing(storage.Inbox); len(left) != 0 {
		t.Errorf("inbox still holds %v", left)
	}
}

func TestTwoRoutersFightingOverOneFileMoveItOnce(t *testing.T) {
	kobo := newFakeKobo(t)
	path := kobo.inbox("once.pdf", []byte("%PDF-1.7 once"))
	var group sync.WaitGroup
	results := make(chan error, 8)
	for range 8 {
		group.Add(1)
		go func() {
			defer group.Done()
			_, err := kobo.router.Route(path, "race")
			results <- err
		}()
	}
	group.Wait()
	close(results)
	delivered := 0
	for err := range results {
		switch {
		case err == nil:
			delivered++
		case !errors.Is(err, ErrGone):
			t.Error(err)
		}
	}
	if delivered != 1 || len(kobo.listing(storage.Documents)) != 1 {
		t.Errorf("delivered %d times, documents %v", delivered, kobo.listing(storage.Documents))
	}
}

func TestJournalLinesDescribeEachFile(t *testing.T) {
	kobo := newFakeKobo(t)
	kobo.route(kobo.inbox("ok.pdf", []byte("%PDF-1.7 x")))
	kobo.route(kobo.inbox("bad.exe", []byte("MZ")))
	data, err := os.ReadFile(JournalPath())
	must(t, err)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("journal = %q", data)
	}
	var routed, refused Entry
	must(t, json.Unmarshal([]byte(lines[0]), &routed))
	must(t, json.Unmarshal([]byte(lines[1]), &refused))
	if routed.Source != "test" || routed.Name != "ok.pdf" || routed.Type != "pdf" || routed.Destination != "documents" || routed.Time.IsZero() {
		t.Errorf("routed = %+v", routed)
	}
	if refused.Destination != "rejected" || refused.Reason != ReasonTypeNotAllowed || refused.Name != "bad.exe" {
		t.Errorf("refused = %+v", refused)
	}
}

func TestIncompleteUploadsAreDeletedNotKept(t *testing.T) {
	kobo := newFakeKobo(t)
	outcome := kobo.route(kobo.inbox("half.epub.tmp", []byte("PK")))
	if outcome.Reason != ReasonIncomplete || outcome.Path != "" {
		t.Fatalf("outcome = %+v", outcome)
	}
	if len(kobo.listing(storage.Inbox)) != 0 || len(kobo.listing(storage.Rejected)) != 0 {
		t.Error("a .tmp leftover must be deleted, not kept")
	}
}

func TestSortFolderRoutesEverythingAndSkipsHiddenAndFolders(t *testing.T) {
	kobo := newFakeKobo(t)
	legacy := storage.LegacyLocalSendDir()
	kobo.put(legacy, "a.pdf", []byte("%PDF-1.7 a"))
	kobo.put(legacy, "b.exe", []byte("MZ"))
	kobo.put(legacy, ".DS_Store", []byte("junk"))
	kobo.put(filepath.Join(legacy, "sub"), "c.pdf", []byte("%PDF-1.7 c"))

	report, err := kobo.router.SortFolder(legacy, "migrate")
	if err != nil || report.Routed != 1 || report.Rejected != 1 || len(report.Failures) != 0 {
		t.Fatalf("report = %+v, err %v", report, err)
	}
	for _, kept := range []string{".DS_Store", "sub"} {
		if _, err := os.Stat(filepath.Join(legacy, kept)); err != nil {
			t.Errorf("%s must stay: %v", kept, err)
		}
	}
	must(t, os.MkdirAll(storage.Path(storage.Books), 0o755))
	if _, err := kobo.router.SortFolder(storage.Path(storage.Books), "x"); !errors.Is(err, ErrOutsideSource) {
		t.Errorf("sorting the library must be refused, got %v", err)
	}
	if _, err := kobo.router.SortFolder(t.TempDir(), "x"); !errors.Is(err, ErrOutsideSource) {
		t.Errorf("sorting a random folder must be refused, got %v", err)
	}
	report, err = kobo.router.SortFolder(filepath.Join(kobo.root, "Missing"), "x")
	if err != nil || report.Routed != 0 {
		t.Errorf("a missing folder is normal: %+v %v", report, err)
	}
}

func TestPurgeDeletesOnlyOldFilesInsideRejected(t *testing.T) {
	kobo := newFakeKobo(t)
	rejected := storage.Path(storage.Rejected)
	old := time.Now().Add(-8 * 24 * time.Hour)
	fresh := time.Now().Add(-6 * 24 * time.Hour)
	oldInRejected := kobo.put(rejected, "old.exe", []byte("x"))
	freshInRejected := kobo.put(rejected, "fresh.exe", []byte("x"))
	oldInInbox := kobo.inbox("old.pdf", []byte("%PDF-1.7 x"))
	oldInBooks := kobo.put(storage.Path(storage.Books), "old.epub", []byte("x"))
	oldSubfolderFile := kobo.put(filepath.Join(rejected, "sub"), "old.txt", []byte("x"))
	target := filepath.Join(t.TempDir(), "target.txt")
	must(t, os.WriteFile(target, []byte("keep me"), 0o644))
	oldLink := filepath.Join(rejected, "oldlink")
	must(t, os.Symlink(target, oldLink))
	for _, path := range []string{oldInRejected, oldInInbox, oldInBooks, oldSubfolderFile, target} {
		must(t, os.Chtimes(path, old, old))
	}
	must(t, os.Chtimes(freshInRejected, fresh, fresh))
	must(t, os.Lchown(oldLink, os.Getuid(), os.Getgid()))

	deleted, err := kobo.router.PurgeRejected(7 * 24 * time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if deleted < 1 {
		t.Errorf("deleted %d", deleted)
	}
	if _, err := os.Stat(oldInRejected); !os.IsNotExist(err) {
		t.Error("old rejected file must be deleted")
	}
	for _, kept := range []string{freshInRejected, oldInInbox, oldInBooks, oldSubfolderFile, target} {
		if _, err := os.Stat(kept); err != nil {
			t.Errorf("%s must survive: %v", kept, err)
		}
	}
	if content, _ := os.ReadFile(target); string(content) != "keep me" {
		t.Error("a link target must never be touched")
	}
}

func TestRejectionIsStampedSoPurgeCountsFromRejectionDay(t *testing.T) {
	kobo := newFakeKobo(t)
	path := kobo.inbox("ancient.exe", []byte("x"))
	ancient := time.Now().Add(-400 * 24 * time.Hour)
	must(t, os.Chtimes(path, ancient, ancient))
	outcome := kobo.route(path)
	deleted, err := kobo.router.PurgeRejected(7 * 24 * time.Hour)
	if err != nil || deleted != 0 {
		t.Errorf("a file rejected just now must survive purge: deleted %d, %v", deleted, err)
	}
	if _, err := os.Stat(outcome.Path); err != nil {
		t.Error(err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	must(t, os.WriteFile(path, []byte(content), 0o644))
}

func TestOverrideCanOnlyRemoveTypesOrRemapToKnownFolders(t *testing.T) {
	dir := t.TempDir()
	override := filepath.Join(dir, "routing.json")
	good := map[string]string{
		"remove": `{"remove":["pdf","gif"]}`,
		"remap":  `{"destinations":{"txt":"books","png":"documents"}}`,
		"both":   `{"remove":["pdf"],"destinations":{"cbz":"books"}}`,
		"empty":  `{}`,
	}
	for name, content := range good {
		writeFile(t, override, content)
		rules, overrideErr, err := LoadRules(shippedRules, override)
		if err != nil || overrideErr != nil {
			t.Errorf("%s: %v / %v", name, err, overrideErr)
			continue
		}
		switch name {
		case "remove":
			if _, _, found := rules.Match("a.pdf"); found {
				t.Error("pdf must be removed")
			}
			if _, _, found := rules.Match("a.epub"); !found {
				t.Error("epub must stay")
			}
		case "remap":
			if rule, _, _ := rules.Match("a.txt"); rule.Destination != storage.Books {
				t.Errorf("txt -> %s", rule.Destination)
			}
		}
	}

	defaults, _, _ := LoadRules(shippedRules, filepath.Join(dir, "absent.json"))
	bad := map[string]string{
		"add a type":           `{"add":[{"type":"exe","extensions":[".exe"],"check":"pdf","max_bytes":1,"destination":"books"}]}`,
		"unknown field":        `{"remove":["pdf"],"max_bytes":5}`,
		"remove unknown":       `{"remove":["exe"]}`,
		"remap unknown type":   `{"destinations":{"exe":"books"}}`,
		"remap to path":        `{"destinations":{"pdf":"/etc"}}`,
		"remap to traversal":   `{"destinations":{"pdf":"../.."}}`,
		"remap to inbox":       `{"destinations":{"pdf":"inbox"}}`,
		"remap to rejected":    `{"destinations":{"pdf":"rejected"}}`,
		"remap to screensaver": `{"destinations":{"pdf":"screensaver"}}`,
		"remap to unknown":     `{"destinations":{"pdf":"secrets"}}`,
		"broken json":          `{"remove":`,
		"trailing data":        `{} {}`,
		"wrong type":           `{"remove":"pdf"}`,
	}
	for name, content := range bad {
		writeFile(t, override, content)
		rules, overrideErr, err := LoadRules(shippedRules, override)
		var rulesErr *RulesError
		if err != nil || !errors.As(overrideErr, &rulesErr) || rulesErr.File != override {
			t.Errorf("%s: want an override error, got err=%v overrideErr=%v", name, err, overrideErr)
		}
		if !reflect.DeepEqual(rules, defaults) {
			t.Errorf("%s: must fall back to the defaults", name)
		}
	}
}

func TestBrokenOrMissingShippedRulesFailClosed(t *testing.T) {
	dir := t.TempDir()
	if _, _, err := LoadRules(filepath.Join(dir, "missing.json"), ""); err == nil {
		t.Error("missing shipped rules must be an error")
	}
	broken := map[string]string{
		"unknown check":   `{"version":1,"rules":[{"type":"x","extensions":[".x"],"check":"magic","max_bytes":1,"destination":"books"}]}`,
		"bad destination": `{"version":1,"rules":[{"type":"x","extensions":[".x"],"check":"pdf","max_bytes":1,"destination":"/etc"}]}`,
		"duplicate ext":   `{"version":1,"rules":[{"type":"x","extensions":[".x"],"check":"pdf","max_bytes":1,"destination":"books"},{"type":"y","extensions":[".x"],"check":"pdf","max_bytes":1,"destination":"books"}]}`,
		"no size cap":     `{"version":1,"rules":[{"type":"x","extensions":[".x"],"check":"pdf","max_bytes":0,"destination":"books"}]}`,
		"upper case ext":  `{"version":1,"rules":[{"type":"x","extensions":[".X"],"check":"pdf","max_bytes":1,"destination":"books"}]}`,
		"wrong version":   `{"version":2,"rules":[]}`,
	}
	for name, content := range broken {
		path := filepath.Join(dir, "r.json")
		writeFile(t, path, content)
		if _, _, err := LoadRules(path, ""); err == nil {
			t.Errorf("%s must be rejected", name)
		}
	}
}

func TestSummaryCountsTheLastSessionOnly(t *testing.T) {
	base := time.Date(2026, 10, 8, 9, 0, 0, 0, time.UTC)
	at := func(minutes int, destination string, reason Reason) Entry {
		return Entry{Time: base.Add(time.Duration(minutes) * time.Minute), Destination: destination, Reason: reason}
	}
	entries := []Entry{
		at(0, "books", ""), at(1, "books", ""),
		at(600, "books", ""), at(601, "documents", ""), at(603, "pictures", ""), at(604, "comics", ""),
		at(605, "rejected", ReasonTooLarge), at(606, "rejected", ReasonTypeNotAllowed), at(607, "rejected", ReasonTypeNotAllowed),
	}
	summary := SummarizeLastSession(entries)
	if summary.Routed[storage.Books] != 1 || summary.Routed[storage.Documents] != 1 ||
		summary.Routed[storage.Pictures] != 1 || summary.Routed[storage.Comics] != 1 || summary.Rejected != 3 {
		t.Fatalf("summary = %+v", summary)
	}
	if len(summary.Reasons) != 2 || summary.Reasons[0] != (ReasonCount{ReasonTypeNotAllowed, 2}) {
		t.Errorf("reasons = %+v", summary.Reasons)
	}
	if !SummarizeLastSession(nil).IsEmpty() {
		t.Error("no entries means an empty summary")
	}
}

func TestReadEntriesSkipsCorruptLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "j.jsonl")
	good, _ := json.Marshal(Entry{Time: time.Now(), Destination: "books"})
	writeFile(t, path, string(good)+"\n{\"time\":\n\nnot json\n"+string(good)+"\n")
	entries, skipped, err := ReadEntries(path)
	if err != nil || len(entries) != 2 || skipped != 2 {
		t.Errorf("entries=%d skipped=%d err=%v", len(entries), skipped, err)
	}
	if entries, _, err := ReadEntries(filepath.Join(t.TempDir(), "absent")); err != nil || entries != nil {
		t.Errorf("a missing journal is empty, got %v %v", entries, err)
	}
}

func TestTextCheckHandlesCharactersCutByChunkBoundaries(t *testing.T) {
	valid := strings.Repeat("ệ", textChunkLength) // 3 bytes each, crosses many chunk edges at odd offsets
	for _, c := range []struct {
		name string
		data string
		want bool
	}{
		{"valid across chunks", valid, true},
		{"invalid at the end", valid + "\xe1\xbb", false},
		{"nul after chunks", valid + "\x00", false},
		{"invalid in the middle", valid + "\xff" + valid, false},
		{"empty is vacuously fine", "", true},
	} {
		got, err := isCleanUTF8(strings.NewReader(c.data))
		if err != nil || got != c.want {
			t.Errorf("%s: got %v, %v", c.name, got, err)
		}
	}
}
