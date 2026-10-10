package library

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/text/unicode/norm"

	"koboapps/internal/i18n"
)

// decomposedName is a file name whose tone marks are separate characters (as typed by some
// keyboards and sent by some phones).
func decomposedName(name string) string { return norm.NFD.String(name) }

// hasName reports whether the folder lists exactly this file name. Unlike os.Stat it tells the
// decomposed and the composed spelling apart on a file system (APFS) that treats them as one.
func hasName(t *testing.T, folder, name string) bool {
	t.Helper()
	entries, err := os.ReadDir(folder)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() == name {
			return true
		}
	}
	return false
}

// foldsNormalization is true on a file system (APFS) where the decomposed and the composed
// spelling of a name are one file. The Kobo's vfat keeps them apart, so name assertions that
// need two files are skipped there.
func foldsNormalization(t *testing.T, folder string) bool {
	t.Helper()
	probe := filepath.Join(folder, ".probe-"+norm.NFD.String("é"))
	if err := os.WriteFile(probe, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(probe)
	_, err := os.Stat(filepath.Join(folder, ".probe-"+norm.NFC.String("é")))
	return err == nil
}

func TestNormalizeNFCRenamesFilesAndRewritesMetadata(t *testing.T) {
	f := newFixture(t)
	spec := vietnameseSpec(norm.NFD.String("Đắc Nhân Tâm"), "vi")
	spec.author = norm.NFD.String("Nguyễn Văn A")
	original := buildEpub(t, spec)
	oldName := "Books/" + decomposedName("Đắc nhân tâm.epub")
	path := f.write(t, oldName, original)
	otherName := "Documents/" + decomposedName("Ghi chú.txt")
	f.write(t, otherName, []byte("ghi chú"))
	fine := buildEpub(t, vietnameseSpec("Truyện Kiều", "vi"))
	finePath := f.write(t, "Books/fine.epub", fine)
	report := f.scan(t)
	if report.Counts.NotNFC != 2 {
		t.Fatalf("not_nfc = %d, want 2 (%+v)", report.Counts.NotNFC, report.Books)
	}

	result, err := f.service.NormalizeNFC(false)
	if err != nil || result.Fixed != 2 || result.Failed != 0 {
		t.Fatalf("result = %+v err = %v", result, err)
	}

	newName := "Books/" + norm.NFC.String("Đắc nhân tâm.epub")
	if hasName(t, filepath.Dir(path), filepath.Base(path)) {
		t.Error("the old, decomposed name is still there")
	}
	reread, err := ReadEpub(filepath.Join(f.root, newName))
	if err != nil || reread.Title != norm.NFC.String("Đắc Nhân Tâm") || reread.Author != norm.NFC.String("Nguyễn Văn A") {
		t.Fatalf("metadata not normalised: %+v %v", reread, err)
	}
	if !norm.NFC.IsNormalString(reread.Title) || !norm.NFC.IsNormalString(reread.Author) {
		t.Error("title or author still decomposed")
	}
	if content, _ := os.ReadFile(finePath); !bytes.Equal(content, fine) {
		t.Error("a book that was fine was modified")
	}
	if saved, err := os.ReadFile(filepath.Join(f.service.Paths.Backups(), result.Date, filepath.FromSlash(oldName))); err != nil || !bytes.Equal(saved, original) {
		t.Errorf("backup of the original missing or different: %v", err)
	}
	after, _, _ := LoadReport(f.service.Paths)
	if after.Counts.NotNFC != 0 {
		t.Errorf("report still lists not-NFC items: %+v", after.Counts)
	}
	if findBook(t, after, newName).Path != newName {
		t.Error("the report row did not follow the rename")
	}
	requests := f.service.takeRequests()
	if len(requests) != 1 || requests[0].Request != "rescan" || requests[0].Mode != "quick" {
		t.Errorf("a quick rescan must be queued, got %+v", requests)
	}
}

func TestNormalizeNFCIsUndoable(t *testing.T) {
	f := newFixture(t)
	spec := vietnameseSpec(norm.NFD.String("Sách"), "vi")
	original := buildEpub(t, spec)
	contentOldName := "Books/" + decomposedName("Truyện.epub")
	f.write(t, contentOldName, original)
	renameOnlyName := "Documents/" + decomposedName("Hướng dẫn.txt")
	f.write(t, renameOnlyName, []byte("nội dung"))
	f.scan(t)
	result, err := f.service.NormalizeNFC(false)
	if err != nil || result.Fixed != 2 {
		t.Fatalf("result = %+v err = %v", result, err)
	}

	restored, err := f.service.Undo("")
	if err != nil || restored != 2 {
		t.Fatalf("undo restored %d: %v", restored, err)
	}
	if content, err := os.ReadFile(filepath.Join(f.root, filepath.FromSlash(contentOldName))); err != nil || !bytes.Equal(content, original) {
		t.Errorf("the original book is not back byte for byte: %v", err)
	}
	if content, err := os.ReadFile(filepath.Join(f.root, filepath.FromSlash(renameOnlyName))); err != nil || string(content) != "nội dung" {
		t.Errorf("the renamed file did not get its old name back: %v", err)
	}
	for _, entry := range []string{"Books/" + norm.NFC.String("Truyện.epub"), "Documents/" + norm.NFC.String("Hướng dẫn.txt")} {
		if foldsNormalization(t, f.root) {
			break
		}
		if hasName(t, filepath.Join(f.root, filepath.Dir(filepath.FromSlash(entry))), filepath.Base(entry)) {
			t.Errorf("%s is still there after the undo", entry)
		}
	}
	if _, err := os.Stat(f.service.Paths.RenameJournal(result.Date)); !os.IsNotExist(err) {
		t.Error("the rename journal outlived the undo")
	}
	if _, err := f.service.Undo(""); err == nil {
		t.Error("a second undo has nothing left to restore")
	}
}

func TestNormalizeNFCNeverOverwritesAnExistingFile(t *testing.T) {
	f := newFixture(t)
	decomposed := "Books/" + decomposedName("Trùng tên.txt")
	f.write(t, decomposed, []byte("decomposed"))
	composed := f.write(t, "Books/"+norm.NFC.String("Trùng tên.txt"), []byte("composed"))
	first, _ := os.Lstat(filepath.Join(f.root, filepath.FromSlash(decomposed)))
	second, _ := os.Lstat(composed)
	if os.SameFile(first, second) {
		t.Skip("this file system treats both spellings as one name (APFS); the Kobo's vfat keeps them apart")
	}
	f.scan(t)

	result, err := f.service.NormalizeNFC(false)
	if err != nil || result.Failed != 1 || result.Fixed != 0 {
		t.Fatalf("result = %+v err = %v", result, err)
	}
	if content, _ := os.ReadFile(composed); string(content) != "composed" {
		t.Error("the file that already had the NFC name was overwritten")
	}
	if _, err := os.Stat(filepath.Join(f.root, filepath.FromSlash(decomposed))); err != nil {
		t.Error("the clashing file must stay where it was")
	}
}

func TestNormalizeNFCDryRunWritesNothing(t *testing.T) {
	f := newFixture(t)
	name := "Books/" + decomposedName("Thử.txt")
	f.write(t, name, []byte("x"))
	f.scan(t)

	result, err := f.service.NormalizeNFC(true)
	if err != nil || result.Candidates != 1 || len(result.Paths) != 1 {
		t.Fatalf("result = %+v err = %v", result, err)
	}
	if _, err := os.Stat(filepath.Join(f.root, filepath.FromSlash(name))); err != nil {
		t.Error("a dry run renamed a file")
	}
	if f.service.HasBackups() {
		t.Error("a dry run wrote a backup")
	}
}

func TestCancelStopsAFixAfterTheCurrentBook(t *testing.T) {
	f := newFixture(t)
	for _, name := range []string{"a", "b", "c"} {
		f.write(t, "Books/"+name+".epub", buildEpub(t, vietnameseSpec("Sách "+name, "en")))
	}
	if err := writeFileAtomic(f.service.Paths.Cancel(), []byte("1")); err != nil {
		t.Fatal(err)
	}

	result, err := f.service.FixLanguage(false)
	if err != nil || !result.Cancelled || result.Fixed != 0 {
		t.Fatalf("a run that starts stopped must do nothing: %+v %v", result, err)
	}
	if _, err := f.service.Scan(); err != ErrCancelled {
		t.Fatalf("scan err = %v, want ErrCancelled", err)
	}
	if _, ok, _ := LoadReport(f.service.Paths); ok {
		t.Error("a stopped scan must not save a partial report")
	}
}

func TestRequestCancelOnlyWorksWhileSomethingRuns(t *testing.T) {
	f := newFixture(t)
	paths := f.service.Paths
	if wasRunning, err := RequestCancel(paths); err != nil || wasRunning {
		t.Fatalf("nothing runs: %v %v", wasRunning, err)
	}
	release, err := Acquire(paths)
	if err != nil {
		t.Fatal(err)
	}
	if wasRunning, err := RequestCancel(paths); err != nil || !wasRunning || !f.service.Cancelled() {
		t.Fatalf("running: %v %v cancelled=%v", wasRunning, err, f.service.Cancelled())
	}
	if !IsCancelling(paths) {
		t.Error("the screen must see that a stop is pending")
	}
	release()
	if f.service.Cancelled() {
		t.Error("a finished run must not leave its stop request for the next one")
	}
}

func TestStateHandsOverQueuedRequestsOnce(t *testing.T) {
	f := newFixture(t)
	f.write(t, "Books/a.epub", buildEpub(t, vietnameseSpec("Một", "en")))
	f.scan(t)
	if result, err := f.service.FixLanguage(false); err != nil || result.Fixed != 1 {
		t.Fatalf("%+v %v", result, err)
	}
	messages := i18n.NewBundle()

	first := f.service.State(messages)
	requests, ok := first["requests"].([]hostRequest)
	if !ok || len(requests) != 1 || requests[0].Request != "rescan" || requests[0].Mode != "quick" {
		t.Fatalf("first state requests = %+v", first["requests"])
	}
	if _, again := f.service.State(messages)["requests"]; again {
		t.Error("the same request was handed over twice")
	}
}

func TestStateWaitsForTheJobBeforeHandingOverRequests(t *testing.T) {
	f := newFixture(t)
	f.service.queueRescan()
	release, err := Acquire(f.service.Paths)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if _, early := f.service.State(i18n.NewBundle())["requests"]; early {
		t.Error("requests must wait until the job has ended")
	}
	release()
	if _, late := f.service.State(i18n.NewBundle())["requests"]; !late {
		t.Error("the request was lost while the job ran")
	}
}

func TestTrashRestoreOffersAndQueuesRescan(t *testing.T) {
	f, _ := duplicateFixture(t)
	messages := i18n.NewBundle()
	if f.service.State(messages)["can_restore"] != false {
		t.Fatal("nothing to restore yet")
	}
	if err := f.service.RemoveDuplicate("Books/b.epub", true); err != nil {
		t.Fatal(err)
	}
	f.service.takeRequests() // the trash itself queued one
	if f.service.State(messages)["can_restore"] != true {
		t.Fatal("the screen must offer to restore")
	}
	if _, err := f.service.RestoreTrash(""); err != nil {
		t.Fatal(err)
	}
	if len(f.service.takeRequests()) != 1 {
		t.Error("restoring books must queue a rescan")
	}
	if f.service.State(messages)["can_restore"] != false {
		t.Error("the restored day is gone from the trash")
	}
}

func TestStateOffersNormalizeAndStop(t *testing.T) {
	f := newFixture(t)
	f.write(t, "Books/"+decomposedName("Thử.txt"), []byte("x"))
	f.scan(t)
	state := f.service.State(i18n.NewBundle())
	if state["can_normalize"] != true || state["can_stop"] != false {
		t.Errorf("state = %+v", state)
	}
	release, err := Acquire(f.service.Paths)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	state = f.service.State(i18n.NewBundle())
	if state["can_normalize"] != false || state["can_stop"] != true {
		t.Errorf("while running: %+v", state)
	}
	if _, err := RequestCancel(f.service.Paths); err != nil {
		t.Fatal(err)
	}
	if f.service.State(i18n.NewBundle())["can_stop"] != false {
		t.Error("the stop button must wait once a stop is pending")
	}
}
