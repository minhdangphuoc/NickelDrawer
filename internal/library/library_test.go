package library

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/text/unicode/norm"

	"koboapps/internal/i18n"
	"koboapps/internal/storage"
)

const (
	vietnameseText = "Tôi yêu tiếng Việt vì nó có nhiều dấu thanh và những âm điệu đẹp đẽ. Đắc nhân tâm là một cuốn sách quen thuộc. "
	englishText    = "The quick brown fox jumps over the lazy dog while the sun sets slowly behind the hills. "
	frenchText     = "Où est passé l'été dernier, déjà fini ? Les élèves étaient très préoccupés par les événements. "
)

type epubSpec struct {
	title, author, language, text string
	xmlLang                       string // lang attribute written on the chapter
	omitLanguage                  bool
}

func vietnameseSpec(title, language string) epubSpec {
	return epubSpec{title: title, author: "Nguyễn Văn A", language: language, text: strings.Repeat(vietnameseText, 8), xmlLang: language}
}

func buildEpub(t *testing.T, spec epubSpec) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	add := func(name, content string, method uint16) {
		entry, err := writer.CreateHeader(&zip.FileHeader{Name: name, Method: method})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	language := fmt.Sprintf("<dc:language>%s</dc:language>", spec.language)
	if spec.omitLanguage {
		language = ""
	}
	add("mimetype", "application/epub+zip", zip.Store)
	add("META-INF/container.xml", `<?xml version="1.0"?><container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles></container>`, zip.Deflate)
	add("OEBPS/content.opf", `<?xml version="1.0" encoding="utf-8"?><package xmlns="http://www.idpf.org/2007/opf" version="3.0" xml:lang="`+spec.xmlLang+`"><metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>`+spec.title+`</dc:title><dc:creator>`+spec.author+`</dc:creator>`+language+`</metadata><manifest><item id="c1" href="chapter%201.xhtml" media-type="application/xhtml+xml"/></manifest><spine><itemref idref="c1"/></spine></package>`, zip.Deflate)
	add("OEBPS/chapter 1.xhtml", `<html xmlns="http://www.w3.org/1999/xhtml" xml:lang="`+spec.xmlLang+`" lang="`+spec.xmlLang+`"><head><title>x</title></head><body><p hreflang="en">`+spec.text+`</p></body></html>`, zip.Deflate)
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

type fixture struct {
	root    string
	service *Service
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	root := t.TempDir()
	t.Cleanup(storage.SetRoot(root))
	for _, folder := range []string{"Books", "Documents", "Comics"} {
		if err := os.MkdirAll(filepath.Join(root, folder), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	appDir := filepath.Join(t.TempDir(), "library")
	if err := os.MkdirAll(filepath.Join(appDir, "data"), 0o755); err != nil {
		t.Fatal(err)
	}
	return fixture{root: root, service: NewService(Paths{AppDir: appDir})}
}

func (f fixture) write(t *testing.T, relative string, content []byte) string {
	t.Helper()
	full := filepath.Join(f.root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, content, 0o644); err != nil {
		t.Fatal(err)
	}
	return full
}

func (f fixture) scan(t *testing.T) Report {
	t.Helper()
	report, err := f.service.Scan()
	if err != nil {
		t.Fatal(err)
	}
	return report
}

func findBook(t *testing.T, report Report, relative string) Book {
	t.Helper()
	for _, book := range report.Books {
		if book.Path == relative {
			return book
		}
	}
	t.Fatalf("%s not in report", relative)
	return Book{}
}

func TestScanReadsMetadataAndDetectsLanguage(t *testing.T) {
	f := newFixture(t)
	f.write(t, "Books/vi-wrong.epub", buildEpub(t, vietnameseSpec("Đắc Nhân Tâm", "en")))
	f.write(t, "Books/vi-right.epub", buildEpub(t, vietnameseSpec("Truyện Kiều", "vi-VN")))
	f.write(t, "Books/english.epub", buildEpub(t, epubSpec{title: "Dog", author: "Fox", language: "en", text: strings.Repeat(englishText, 8)}))
	f.write(t, "Documents/french.epub", buildEpub(t, epubSpec{title: "Été", author: "Élève", language: "fr", text: strings.Repeat(frenchText, 8)}))
	f.write(t, "Books/no-language.epub", buildEpub(t, func() epubSpec { s := vietnameseSpec("Không ngôn ngữ", ""); s.omitLanguage = true; return s }()))

	report := f.scan(t)

	wrong := findBook(t, report, "Books/vi-wrong.epub")
	if wrong.Title != "Đắc Nhân Tâm" || wrong.Author != "Nguyễn Văn A" || wrong.Language != "en" || !wrong.Vietnamese || !wrong.NeedsLanguageFix() {
		t.Errorf("wrong-language book not detected: %+v", wrong)
	}
	if findBook(t, report, "Books/vi-right.epub").NeedsLanguageFix() {
		t.Error("a book tagged vi-VN must not need a fix")
	}
	if book := findBook(t, report, "Books/english.epub"); book.Vietnamese || book.NeedsLanguageFix() {
		t.Errorf("English text taken for Vietnamese: %+v", book)
	}
	if book := findBook(t, report, "Documents/french.epub"); book.Vietnamese {
		t.Errorf("French text taken for Vietnamese (share %.2f)", book.VietnameseShare)
	}
	if !findBook(t, report, "Books/no-language.epub").NeedsLanguageFix() {
		t.Error("a Vietnamese book without dc:language needs a fix")
	}
	if report.Counts.WrongLanguage != 2 || report.Counts.Files != 5 || report.Counts.NotNFC != 0 {
		t.Errorf("unexpected counts: %+v", report.Counts)
	}
}

func TestScanDetectsNonNFCText(t *testing.T) {
	f := newFixture(t)
	decomposed := vietnameseSpec("Sách", "vi")
	decomposed.text = norm.NFD.String(decomposed.text)
	f.write(t, "Books/nfd.epub", buildEpub(t, decomposed))
	f.write(t, "Books/nfc.epub", buildEpub(t, vietnameseSpec("Sách khác", "vi")))

	report := f.scan(t)

	if !findBook(t, report, "Books/nfd.epub").NotNFC || findBook(t, report, "Books/nfc.epub").NotNFC {
		t.Errorf("NFC detection wrong: %+v", report.Books)
	}
	if !findBook(t, report, "Books/nfd.epub").Vietnamese {
		t.Error("decomposed Vietnamese text must still count as Vietnamese")
	}
	if report.Counts.NotNFC != 1 {
		t.Errorf("not_nfc = %d, want 1", report.Counts.NotNFC)
	}
}

func TestScanFindsIdenticalAndProbableDuplicates(t *testing.T) {
	f := newFixture(t)
	original := buildEpub(t, vietnameseSpec("Dế Mèn Phiêu Lưu Ký", "vi"))
	f.write(t, "Books/a.epub", original)
	f.write(t, "Books/GitHub/a copy.epub", original)
	other := vietnameseSpec("DE MEN phieu luu ky", "vi")
	other.text += " thêm một câu khác"
	f.write(t, "Documents/other.epub", buildEpub(t, other))
	f.write(t, "Books/unique.epub", buildEpub(t, vietnameseSpec("Một sách khác hẳn", "vi")))

	report := f.scan(t)

	if len(report.Duplicates) != 1 || report.Counts.DuplicateGroups != 1 {
		t.Fatalf("want one group, got %+v", report.Duplicates)
	}
	group := report.Duplicates[0]
	if len(group.Paths) != 3 || group.Kind != KindProbable {
		t.Errorf("group = %+v, want 3 paths of kind probable", group)
	}
	if findBook(t, report, "Books/a.epub").SHA256 == "" || findBook(t, report, "Books/unique.epub").SHA256 != "" {
		t.Error("only files sharing a size are hashed")
	}

	identicalOnly := newFixture(t)
	identicalOnly.write(t, "Books/x.epub", original)
	identicalOnly.write(t, "Comics/x.epub", original)
	if group := identicalOnly.scan(t).Duplicates[0]; group.Kind != KindIdentical {
		t.Errorf("kind = %s, want identical", group.Kind)
	}
}

func TestScanSkipsUnsafeAndOversizedFiles(t *testing.T) {
	f := newFixture(t)
	outside := filepath.Join(t.TempDir(), "outside.epub")
	if err := os.WriteFile(outside, buildEpub(t, vietnameseSpec("Ngoài", "en")), 0o644); err != nil {
		t.Fatal(err)
	}
	outsideDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(outsideDir, "inner.epub"), buildEpub(t, vietnameseSpec("Trong", "en")), 0o644); err != nil {
		t.Fatal(err)
	}
	for link, target := range map[string]string{"Books/link.epub": outside, "Books/linkdir": outsideDir} {
		if err := os.Symlink(target, filepath.Join(f.root, link)); err != nil {
			t.Fatal(err)
		}
	}
	f.write(t, "Books/.hidden.epub", buildEpub(t, vietnameseSpec("Ẩn", "en")))
	f.write(t, "Books/_private.epub", buildEpub(t, vietnameseSpec("Riêng", "en")))
	f.write(t, "Books/.cache/inner.epub", buildEpub(t, vietnameseSpec("Ẩn", "en")))
	f.write(t, "Books/notes.exe", []byte("MZ"))
	huge := f.write(t, "Books/huge.epub", nil)
	if err := os.Truncate(huge, maxScanBytes+1); err != nil {
		t.Fatal(err)
	}
	f.write(t, "Books/ok.epub", buildEpub(t, vietnameseSpec("Được", "vi")))
	f.write(t, "LocalSend/old.epub", buildEpub(t, vietnameseSpec("Cũ", "en")))
	f.write(t, "Pictures/p.epub", buildEpub(t, vietnameseSpec("Ảnh", "en")))

	report := f.scan(t)

	if report.Counts.Files != 1 || report.Books[0].Path != "Books/ok.epub" {
		t.Errorf("only Books/ok.epub should be scanned, got %+v", report.Books)
	}
	if report.Counts.Skipped != 1 {
		t.Errorf("skipped = %d, want 1 (oversized file)", report.Counts.Skipped)
	}
}

func TestScanReportsMalformedEpubsWithoutFailing(t *testing.T) {
	f := newFixture(t)
	f.write(t, "Books/garbage.epub", []byte("this is not a zip file"))
	var noContainer bytes.Buffer
	writer := zip.NewWriter(&noContainer)
	entry, _ := writer.Create("mimetype")
	_, _ = entry.Write([]byte("application/epub+zip"))
	_ = writer.Close()
	f.write(t, "Books/empty.epub", noContainer.Bytes())
	f.write(t, "Books/fine.epub", buildEpub(t, vietnameseSpec("Tốt", "vi")))

	report := f.scan(t)

	if report.Counts.Unreadable != 2 || findBook(t, report, "Books/garbage.epub").Error == "" {
		t.Errorf("malformed files not reported: %+v", report.Counts)
	}
	if findBook(t, report, "Books/fine.epub").Error != "" {
		t.Error("good book flagged")
	}
	saved, ok, err := LoadReport(f.service.Paths)
	if err != nil || !ok || saved.Counts != report.Counts {
		t.Errorf("report.json round trip failed: %v %v", ok, err)
	}
	if summary, ok, _ := LoadSummary(f.service.Paths); !ok || len(summary.Books) != 0 {
		t.Error("summary.json must exist without books")
	}
}

func openEpub(t *testing.T, path string) *zip.ReadCloser {
	t.Helper()
	archive, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("%s is not a valid zip: %v", path, err)
	}
	t.Cleanup(func() { archive.Close() })
	return archive
}

func entryText(t *testing.T, archive *zip.ReadCloser, name string) string {
	t.Helper()
	for _, file := range archive.File {
		if file.Name == name {
			content, err := readFile(file)
			if err != nil {
				t.Fatal(err)
			}
			return string(content)
		}
	}
	t.Fatalf("no entry %s", name)
	return ""
}

func TestFixLanguageProducesValidEpubAndBacksUp(t *testing.T) {
	f := newFixture(t)
	original := buildEpub(t, vietnameseSpec("Đắc Nhân Tâm", "en-US"))
	path := f.write(t, "Books/book.epub", original)
	untouched := buildEpub(t, vietnameseSpec("Đúng rồi", "vi"))
	untouchedPath := f.write(t, "Books/fine.epub", untouched)
	f.scan(t)

	result, err := f.service.FixLanguage(false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Fixed != 1 || result.Failed != 0 || result.Candidates != 1 {
		t.Fatalf("result = %+v", result)
	}

	archive := openEpub(t, path)
	if first := archive.File[0]; first.Name != "mimetype" || first.Method != zip.Store {
		t.Errorf("first entry = %s method %d, want stored mimetype", first.Name, first.Method)
	}
	opf := entryText(t, archive, "OEBPS/content.opf")
	if !strings.Contains(opf, "<dc:language>vi</dc:language>") || strings.Contains(opf, `lang="en`) {
		t.Errorf("package document not fixed: %s", opf)
	}
	chapter := entryText(t, archive, "OEBPS/chapter 1.xhtml")
	if strings.Contains(chapter, `xml:lang="en`) || !strings.Contains(chapter, `xml:lang="vi"`) || !strings.Contains(chapter, `<html xmlns="http://www.w3.org/1999/xhtml" xml:lang="vi" lang="vi">`) {
		t.Errorf("chapter language not fixed: %s", chapter)
	}
	if !strings.Contains(chapter, `hreflang="en"`) || !strings.Contains(chapter, vietnameseText[:40]) {
		t.Errorf("unrelated markup or text changed: %s", chapter)
	}
	reread, err := ReadEpub(path)
	if err != nil || reread.Language != "vi" || reread.Title != "Đắc Nhân Tâm" {
		t.Errorf("fixed book unreadable or changed: %+v %v", reread, err)
	}
	backup := filepath.Join(f.service.Paths.Backups(), result.Date, "Books", "book.epub")
	if saved, err := os.ReadFile(backup); err != nil || !bytes.Equal(saved, original) {
		t.Errorf("backup missing or different: %v", err)
	}
	if content, _ := os.ReadFile(untouchedPath); !bytes.Equal(content, untouched) {
		t.Error("a correct book was modified")
	}
	if leftovers, _ := filepath.Glob(filepath.Join(f.root, "Books", ".library-*")); len(leftovers) != 0 {
		t.Errorf("temp files left: %v", leftovers)
	}
	report, _, _ := LoadReport(f.service.Paths)
	if report.Counts.WrongLanguage != 0 {
		t.Errorf("report still lists wrong language: %+v", report.Counts)
	}
}

func TestFixLanguageAddsMissingLanguageElement(t *testing.T) {
	f := newFixture(t)
	spec := vietnameseSpec("Không ngôn ngữ", "")
	spec.omitLanguage = true
	path := f.write(t, "Books/missing.epub", buildEpub(t, spec))

	if result, err := f.service.FixLanguage(false); err != nil || result.Fixed != 1 {
		t.Fatalf("%+v %v", result, err)
	}
	if info, err := ReadEpub(path); err != nil || info.Language != "vi" {
		t.Errorf("language not added: %+v %v", info, err)
	}
}

func TestFixLanguageDryRunWritesNothing(t *testing.T) {
	f := newFixture(t)
	original := buildEpub(t, vietnameseSpec("Thử", "en"))
	path := f.write(t, "Books/dry.epub", original)

	result, err := f.service.FixLanguage(true)
	if err != nil || result.Candidates != 1 || result.Fixed != 1 {
		t.Fatalf("%+v %v", result, err)
	}
	if content, _ := os.ReadFile(path); !bytes.Equal(content, original) {
		t.Error("dry run changed the book")
	}
	if _, err := os.Stat(f.service.Paths.Backups()); !os.IsNotExist(err) {
		t.Error("dry run created a backup")
	}
}

func TestFixLanguageSkipsLargeBooksAndChecksFreeSpace(t *testing.T) {
	f := newFixture(t)
	original := buildEpub(t, vietnameseSpec("To", "en"))
	path := f.write(t, "Books/big.epub", original)
	previous := maxFixBytes
	maxFixBytes = int64(len(original)) - 1
	defer func() { maxFixBytes = previous }()

	result, err := f.service.FixLanguage(false)
	if err != nil || result.Skipped != 1 || result.Fixed != 0 {
		t.Fatalf("%+v %v", result, err)
	}
	if content, _ := os.ReadFile(path); !bytes.Equal(content, original) {
		t.Error("a skipped book was modified")
	}
	if _, err := os.Stat(f.service.Paths.Backups()); !os.IsNotExist(err) {
		t.Error("a skipped book was backed up")
	}
	if err := ensureFreeSpace(f.root, 1<<62); err == nil || !strings.Contains(err.Error(), "msg.err_no_space") {
		t.Errorf("free-space check did not fail: %v", err)
	}
}

func TestUndoRestoresOriginalsByteForByte(t *testing.T) {
	f := newFixture(t)
	first := buildEpub(t, vietnameseSpec("Một", "en"))
	second := buildEpub(t, vietnameseSpec("Hai", "en"))
	path1 := f.write(t, "Books/one.epub", first)
	path2 := f.write(t, "Books/sub/two.epub", second)
	result, err := f.service.FixLanguage(false)
	if err != nil || result.Fixed != 2 {
		t.Fatalf("%+v %v", result, err)
	}
	if content, _ := os.ReadFile(path1); bytes.Equal(content, first) {
		t.Fatal("fix did not change the file")
	}

	restored, err := f.service.Undo(result.Date)
	if err != nil || restored != 2 {
		t.Fatalf("restored %d: %v", restored, err)
	}
	for path, want := range map[string][]byte{path1: first, path2: second} {
		if content, _ := os.ReadFile(path); !bytes.Equal(content, want) {
			t.Errorf("%s not restored", path)
		}
	}
	if _, err := os.Stat(filepath.Join(f.service.Paths.Backups(), result.Date)); !os.IsNotExist(err) {
		t.Error("restored backups should be removed")
	}
	if _, err := f.service.Undo(""); err == nil {
		t.Error("undo with no backups must fail")
	}
}

func TestUndoRefusesBadDatesAndEscapingBackups(t *testing.T) {
	f := newFixture(t)
	if _, err := f.service.Undo("../../etc"); err == nil {
		t.Error("a path as date must be refused")
	}
	escape := filepath.Join(f.service.Paths.Backups(), "2026-01-01", "Pictures")
	if err := os.MkdirAll(escape, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(escape, "x.epub"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if restored, err := f.service.Undo("2026-01-01"); err == nil || restored != 0 {
		t.Errorf("backup outside the library folders restored %d (%v)", restored, err)
	}
	if _, err := os.Stat(filepath.Join(f.root, "Pictures", "x.epub")); err == nil {
		t.Error("file written outside the scanned folders")
	}
}

func duplicateFixture(t *testing.T) (fixture, []byte) {
	f := newFixture(t)
	content := buildEpub(t, vietnameseSpec("Trùng", "vi"))
	f.write(t, "Books/a.epub", content)
	f.write(t, "Books/b.epub", content)
	f.scan(t)
	return f, content
}

func TestRemoveDuplicateMovesToTrashAndRestores(t *testing.T) {
	f, content := duplicateFixture(t)

	if err := f.service.RemoveDuplicate("Books/b.epub", false); err == nil {
		t.Fatal("removal without confirmation must fail")
	}
	if err := f.service.RemoveDuplicate("Books/b.epub", true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(f.root, "Books", "b.epub")); !os.IsNotExist(err) {
		t.Error("duplicate still in the library")
	}
	trashed := filepath.Join(f.service.Paths.Trash(), today(), "Books", "b.epub")
	if saved, err := os.ReadFile(trashed); err != nil || !bytes.Equal(saved, content) {
		t.Errorf("trashed copy missing: %v", err)
	}
	report, _, _ := LoadReport(f.service.Paths)
	if len(report.Duplicates) != 0 || report.Counts.Files != 1 {
		t.Errorf("report not updated: %+v", report.Counts)
	}
	if err := f.service.RemoveDuplicate("Books/a.epub", true); err == nil {
		t.Error("the last copy must never be removed")
	}

	restored, err := f.service.RestoreTrash("")
	if err != nil || restored != 1 {
		t.Fatalf("restore: %d %v", restored, err)
	}
	if saved, _ := os.ReadFile(filepath.Join(f.root, "Books", "b.epub")); !bytes.Equal(saved, content) {
		t.Error("restored file differs")
	}
}

func TestRemoveDuplicateOnlyTouchesGroupMembers(t *testing.T) {
	f, _ := duplicateFixture(t)
	f.write(t, "Books/solo.epub", buildEpub(t, vietnameseSpec("Một mình", "vi")))
	outside := filepath.Join(t.TempDir(), "victim.epub")
	_ = os.WriteFile(outside, []byte("keep"), 0o644)
	for _, path := range []string{"Books/solo.epub", "../victim.epub", outside, "Books/../../x.epub", "Books/.hidden.epub", "/etc/passwd"} {
		if err := f.service.RemoveDuplicate(path, true); err == nil {
			t.Errorf("%q was accepted", path)
		}
	}
	if _, err := os.Stat(outside); err != nil {
		t.Error("file outside the library was touched")
	}
}

func TestRemoveDuplicateRefusesSymlinkedMember(t *testing.T) {
	f, _ := duplicateFixture(t)
	target := filepath.Join(t.TempDir(), "elsewhere.epub")
	_ = os.WriteFile(target, []byte("keep"), 0o644)
	_ = os.Remove(filepath.Join(f.root, "Books", "b.epub"))
	if err := os.Symlink(target, filepath.Join(f.root, "Books", "b.epub")); err != nil {
		t.Fatal(err)
	}
	if err := f.service.RemoveDuplicate("Books/b.epub", true); err == nil {
		t.Error("a symlink must not be moved")
	}
	if _, err := os.Stat(target); err != nil {
		t.Error("symlink target removed")
	}
}

func TestPurgeTrashDeletesOnlyOldDaysInsideTrash(t *testing.T) {
	f := newFixture(t)
	old := time.Now().Add(-31 * 24 * time.Hour).Format(dateLayout)
	recent := time.Now().Add(-5 * 24 * time.Hour).Format(dateLayout)
	for _, day := range []string{old, recent} {
		dir := filepath.Join(f.service.Paths.Trash(), day, "Books")
		_ = os.MkdirAll(dir, 0o755)
		_ = os.WriteFile(filepath.Join(dir, "x.epub"), []byte("x"), 0o644)
	}
	stranger := filepath.Join(f.service.Paths.Trash(), "not-a-date")
	_ = os.MkdirAll(stranger, 0o755)
	bystander := f.write(t, "Books/bystander.epub", []byte("x"))

	f.service.PurgeTrash()

	if _, err := os.Stat(filepath.Join(f.service.Paths.Trash(), old)); !os.IsNotExist(err) {
		t.Error("old trash not purged")
	}
	for _, kept := range []string{filepath.Join(f.service.Paths.Trash(), recent), stranger, bystander} {
		if _, err := os.Stat(kept); err != nil {
			t.Errorf("%s was removed", kept)
		}
	}
}

func TestLockRefusesSecondRunAndTakesOverStaleLock(t *testing.T) {
	f := newFixture(t)
	release, err := Acquire(f.service.Paths)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Acquire(f.service.Paths); err == nil {
		t.Error("second run must be refused")
	}
	if !IsRunning(f.service.Paths) {
		t.Error("IsRunning must be true while locked")
	}
	release()
	if IsRunning(f.service.Paths) {
		t.Error("IsRunning must be false after release")
	}
	_ = os.WriteFile(f.service.Paths.Lock(), []byte("2147483646"), 0o644)
	release, err = Acquire(f.service.Paths)
	if err != nil {
		t.Fatalf("stale lock not taken over: %v", err)
	}
	release()
}

func TestStateReportsCountsPagesAndSelection(t *testing.T) {
	f := newFixture(t)
	for index := 0; index < 5; index++ {
		content := buildEpub(t, vietnameseSpec(fmt.Sprintf("Sách số %d", index), "en"))
		f.write(t, fmt.Sprintf("Books/s%d-a.epub", index), content)
		f.write(t, fmt.Sprintf("Books/s%d-b.epub", index), content)
	}
	messages := i18n.NewBundle()
	if state := f.service.State(messages); state["books"] != "0" || state["can_fix"] != false {
		t.Errorf("state before any scan: %+v", state)
	}
	f.scan(t)

	state := f.service.State(messages)
	if state["books"] != "10" || state["wrong"] != "10" || state["groups"] != "5" || state["can_fix"] != true || state["has_pages"] != true || state["running"] != false {
		t.Errorf("state = %+v", state)
	}
	if rows := state["duplicates"].([][]string); len(rows) != 2*groupsPerPage {
		t.Errorf("first page has %d rows, want %d", len(rows), 2*groupsPerPage)
	}
	if err := f.service.NextPage(); err != nil {
		t.Fatal(err)
	}
	if rows := f.service.State(messages)["duplicates"].([][]string); len(rows) != 2 {
		t.Errorf("second page has %d rows, want 2", len(rows))
	}
	if err := f.service.Select("Books/s0-a.epub"); err != nil {
		t.Fatal(err)
	}
	if state := f.service.State(messages); state["has_selection"] != true {
		t.Error("selection not reflected")
	}
	if err := f.service.Select("Books/nope.epub"); err == nil {
		t.Error("selecting a non-duplicate must fail")
	}
}

func TestVietnameseDetectionThresholds(t *testing.T) {
	for name, text := range map[string]string{"english": englishText, "french": strings.Repeat(frenchText, 3), "short": "Việt"} {
		if _, isVietnamese := LooksVietnamese(strings.Repeat(text, 3)); isVietnamese && name != "short" {
			t.Errorf("%s detected as Vietnamese", name)
		}
	}
	if _, isVietnamese := LooksVietnamese(strings.Repeat(vietnameseText, 3)); !isVietnamese {
		t.Error("Vietnamese not detected")
	}
	if _, isVietnamese := LooksVietnamese("Việt"); isVietnamese {
		t.Error("too little text must not count")
	}
}

func TestStripTagsDropsScriptsAndDecodesEntities(t *testing.T) {
	got := StripTags([]byte(`<head><title>T</title></head><body><p>Xin&nbsp;ch&#224;o<b>!</b></p><script>var x="<p>no</p>";</script><p>b&#7841;n`))
	if strings.Contains(got, "no") || strings.Contains(got, "T") && strings.Contains(got, "title") || !strings.Contains(got, "chào!") || !strings.Contains(got, "bạn") {
		t.Errorf("StripTags = %q", got)
	}
}

func TestValidateRelativeRejectsEscapes(t *testing.T) {
	for _, bad := range []string{"", ".", "Books", "Books/", "Books/../x", "/Books/x.epub", "Pictures/x.epub", "Books//x.epub", "Books/.h/x.epub", "../Books/x.epub"} {
		if validateRelative(bad) == nil {
			t.Errorf("%q accepted", bad)
		}
	}
	if err := validateRelative("Books/GitHub/x y.epub"); err != nil {
		t.Errorf("good path rejected: %v", err)
	}
}
