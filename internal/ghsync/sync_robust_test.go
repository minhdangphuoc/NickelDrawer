package ghsync

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"koboapps/internal/inbox"
	"koboapps/internal/storage"
)

func (e env) exists(relative string) bool {
	_, err := os.Stat(filepath.Join(e.dest, relative))
	return err == nil
}

func (e env) leftoverParts() []string {
	var found []string
	filepath.WalkDir(e.dest, func(path string, entry os.DirEntry, err error) error {
		if err == nil && strings.HasPrefix(entry.Name(), ".ghsync-") {
			found = append(found, path)
		}
		return nil
	})
	return found
}

func (e env) writeConfig(t *testing.T, body string) {
	t.Helper()
	must(t, os.WriteFile(e.paths.Config(), []byte(body), 0o644))
}

func TestEmptyRepositoryIsASuccessfulSyncOfZeroBooks(t *testing.T) {
	fake := newFakeGitHub(t, nil)
	fake.empty = true
	e := newEnv(t, "")
	summary, err := e.run(t, fake)
	must(t, err)
	if summary.New != 0 || summary.Existing != 0 || summary.Errors != 0 {
		t.Fatalf("summary = %+v", summary)
	}
	state, loadErr := LoadState(e.paths.State())
	must(t, loadErr)
	if state.LastRun == "" || state.LastError != "" {
		t.Fatalf("an empty repo is a clean run: %+v", state)
	}
}

func TestMissingBranchIsNamedNotAGeneric404(t *testing.T) {
	fake := newFakeGitHub(t, sampleFiles())
	fake.noBranch = true
	e := newEnv(t, `,"branch":"gone"`)
	_, err := e.run(t, fake)
	if !hasMessageKey(err, "msg.err_branch_not_found") {
		t.Fatalf("got %v", err)
	}
}

func TestMissingFolderSyncsZeroBooksAndSaysSo(t *testing.T) {
	fake := newFakeGitHub(t, sampleFiles())
	e := newEnv(t, "")
	e.writeConfig(t, `{"repo":"o/r","path":"novels","dest":"`+e.dest+`"}`)
	summary, err := e.run(t, fake)
	must(t, err)
	if !summary.FolderMissing || summary.New != 0 {
		t.Fatalf("summary = %+v", summary)
	}
}

func TestOnlyAllowlistedFilesAreDownloaded(t *testing.T) {
	files := sampleFiles()
	files["books/tool.exe"] = "MZ not a book"
	files["books/notes.txt"] = "plain text is on the allowlist but not in this config"
	fake := newFakeGitHub(t, files)
	e := newEnv(t, `,"extensions":[".epub",".exe"]`)
	summary, err := e.run(t, fake)
	must(t, err)
	if e.exists("tool.exe") || e.exists("notes.txt") {
		t.Fatal("a file type that is not on the allowlist was downloaded")
	}
	if summary.New != 2 || atomic.LoadInt32(&fake.downloads) != 2 {
		t.Fatalf("only the 2 epubs may be fetched: %+v, %d downloads", summary, fake.downloads)
	}
}

func TestContentThatDoesNotMatchItsExtensionIsDiscardedOnceAndRemembered(t *testing.T) {
	files := map[string]string{"books/fake.epub": "<html>an error page saved as .epub</html>", "books/real.epub": epubContent("R")}
	fake := newFakeGitHub(t, files)
	e := newEnv(t, "")
	summary, err := e.run(t, fake)
	must(t, err)
	if summary.New != 1 || summary.Skipped != 1 || summary.Errors != 0 {
		t.Fatalf("summary = %+v", summary)
	}
	if e.exists("fake.epub") || !e.exists("real.epub") || len(e.leftoverParts()) != 0 {
		t.Fatalf("fake installed or temp left: %v", e.leftoverParts())
	}
	before := atomic.LoadInt32(&fake.downloads)
	summary, err = e.run(t, fake)
	must(t, err)
	if summary.Skipped != 1 || atomic.LoadInt32(&fake.downloads) != before {
		t.Fatalf("the same bad file was fetched again: %+v", summary)
	}
}

func TestFilesAboveTheTypeSizeCapAreSkippedBeforeDownloading(t *testing.T) {
	fake := newFakeGitHub(t, sampleFiles())
	e := newEnv(t, "")
	rules := testRules(t)
	for index := range rules.Rules {
		rules.Rules[index].MaxBytes = 100
	}
	files := sampleFiles()
	files["books/big.epub"] = epubContent(strings.Repeat("x", 500))
	fake.files = files
	options := e.options(t, fake)
	options.Rules = rules
	summary, err := Run(context.Background(), options)
	must(t, err)
	if summary.Skipped != 1 || e.exists("big.epub") || summary.New != 3 {
		t.Fatalf("summary = %+v", summary)
	}
}

func TestEveryTypeIsRoutedToItsLibraryFolderWhenNoDestIsConfigured(t *testing.T) {
	root := t.TempDir()
	defer storage.SetRoot(root)()
	files := map[string]string{
		"books/a.epub":    epubContent("A"),
		"books/sub/b.pdf": pdfContent("B"),
		"books/c.cbz":     "PK\x03\x04comic",
	}
	fake := newFakeGitHub(t, files)
	e := newEnv(t, "")
	e.writeConfig(t, `{"repo":"o/r","path":"books"}`)
	_, err := e.run(t, fake)
	must(t, err)
	for _, want := range []string{"Books/GitHub/a.epub", "Documents/GitHub/sub/b.pdf", "Comics/GitHub/c.cbz"} {
		if _, err := os.Stat(filepath.Join(root, want)); err != nil {
			t.Errorf("missing %s: %v", want, err)
		}
	}
}

func TestVietnameseNamesAreStoredInNFCAndNotDuplicated(t *testing.T) {
	decomposed := "books/Vie\u0302\u0323t Nam su\u031b\u0309.epub"
	composed := "Việt Nam sử.epub"
	if decomposed == "books/"+composed {
		t.Fatal("test needs a decomposed name")
	}
	fake := newFakeGitHub(t, map[string]string{decomposed: epubContent("V")})
	e := newEnv(t, "")
	summary, err := e.run(t, fake)
	must(t, err)
	if summary.New != 1 || !e.exists(composed) {
		t.Fatalf("name not in NFC: %+v", summary)
	}
	summary, err = e.run(t, fake)
	must(t, err)
	if summary.New != 0 || summary.Existing != 1 {
		t.Fatalf("the NFC copy must count as the same book: %+v", summary)
	}
}

func TestInterruptedDownloadLeavesNoFinalFileAndTheNextRunCompletesIt(t *testing.T) {
	fake := newFakeGitHub(t, sampleFiles())
	fake.abort = map[string]bool{"books/c.pdf": true}
	e := newEnv(t, "")
	summary, err := e.run(t, fake)
	must(t, err)
	if summary.Errors != 1 || summary.New != 2 {
		t.Fatalf("summary = %+v", summary)
	}
	if e.exists("c.pdf") || len(e.leftoverParts()) != 0 {
		t.Fatalf("a partial file was kept: %v", e.leftoverParts())
	}
	fake.abort = nil
	summary, err = e.run(t, fake)
	must(t, err)
	if summary.New != 1 || !e.exists("c.pdf") {
		t.Fatalf("resume failed: %+v", summary)
	}
}

func TestUnfinishedDownloadsOfAKilledRunAreCleanedUp(t *testing.T) {
	fake := newFakeGitHub(t, sampleFiles())
	e := newEnv(t, "")
	must(t, os.MkdirAll(filepath.Join(e.dest, "sub"), 0o755))
	stale := []string{filepath.Join(e.dest, ".ghsync-123.part"), filepath.Join(e.dest, "sub", ".ghsync-456.part")}
	for _, path := range stale {
		must(t, os.WriteFile(path, []byte("PK"), 0o644))
	}
	userFile := filepath.Join(e.dest, "mine.part")
	must(t, os.WriteFile(userFile, []byte("keep"), 0o644))
	_, err := e.run(t, fake)
	must(t, err)
	if len(e.leftoverParts()) != 0 {
		t.Fatalf("stale parts remain: %v", e.leftoverParts())
	}
	if _, err := os.Stat(userFile); err != nil {
		t.Fatal("a file that is not ours was removed")
	}
}

func TestSyncNeverRemovesOrChangesAnythingItDidNotCreate(t *testing.T) {
	fake := newFakeGitHub(t, sampleFiles())
	e := newEnv(t, "")
	must(t, os.MkdirAll(e.dest, 0o755))
	mine := filepath.Join(e.dest, "mine.epub")
	must(t, os.WriteFile(mine, []byte("MINE"), 0o644))
	fake.files = map[string]string{}
	_, err := e.run(t, fake)
	must(t, err)
	if content, _ := os.ReadFile(mine); string(content) != "MINE" {
		t.Fatalf("local file changed: %q", content)
	}
}

func TestUnchangedListingIsAnsweredWith304AndDownloadsNothing(t *testing.T) {
	fake := newFakeGitHub(t, sampleFiles())
	fake.etag = `W/"tree-1"`
	e := newEnv(t, "")
	_, err := e.run(t, fake)
	must(t, err)
	before := atomic.LoadInt32(&fake.downloads)
	summary, err := e.run(t, fake)
	must(t, err)
	if !summary.Unchanged || summary.New != 0 || atomic.LoadInt32(&fake.notModified) != 1 || atomic.LoadInt32(&fake.downloads) != before {
		t.Fatalf("summary = %+v", summary)
	}
}

func TestChangedSettingsOrAnEarlierFailureIgnoreTheValidator(t *testing.T) {
	fake := newFakeGitHub(t, sampleFiles())
	fake.etag = `W/"tree-1"`
	e := newEnv(t, "")
	_, err := e.run(t, fake)
	must(t, err)
	e.writeConfig(t, `{"repo":"o/r","path":"books/sub","dest":"`+e.dest+`"}`)
	summary, err := e.run(t, fake)
	must(t, err)
	if summary.Unchanged {
		t.Fatal("another folder must not be answered with 'unchanged'")
	}
	fake.abort = map[string]bool{"books/sub/b.KEPUB.EPUB": true}
	fake.etag = `W/"tree-2"`
	os.Remove(filepath.Join(e.dest, "b.KEPUB.EPUB"))
	state, _ := LoadState(e.paths.State())
	delete(state.Files, "books/sub/b.KEPUB.EPUB")
	must(t, state.Save(e.paths.State()))
	summary, err = e.run(t, fake)
	must(t, err)
	if summary.Errors != 1 {
		t.Fatalf("setup: want one failed download, got %+v", summary)
	}
	fake.abort = nil
	summary, err = e.run(t, fake)
	must(t, err)
	if summary.Unchanged || summary.New != 1 {
		t.Fatalf("a run that had errors must not be skipped next time: %+v", summary)
	}
}

func TestRedownloadDeletedBypassesTheValidator(t *testing.T) {
	fake := newFakeGitHub(t, sampleFiles())
	fake.etag = `W/"tree-1"`
	e := newEnv(t, "")
	_, err := e.run(t, fake)
	must(t, err)
	must(t, os.Remove(filepath.Join(e.dest, "c.pdf")))
	options := e.options(t, fake)
	options.RedownloadDeleted = true
	summary, err := Run(context.Background(), options)
	must(t, err)
	if summary.Unchanged || summary.New != 1 {
		t.Fatalf("summary = %+v", summary)
	}
}

func TestProgressIsPublishedWhileRunningAndRemovedAfterwards(t *testing.T) {
	fake := newFakeGitHub(t, sampleFiles())
	e := newEnv(t, "")
	progressPath := filepath.Join(t.TempDir(), "progress.json")
	var seen []Progress
	fake.onDownload = func(string) {
		raw, err := os.ReadFile(progressPath)
		var progress Progress
		if err == nil && json.Unmarshal(raw, &progress) == nil {
			seen = append(seen, progress)
		}
	}
	options := e.options(t, fake)
	options.ProgressFile = progressPath
	_, err := Run(context.Background(), options)
	must(t, err)
	if len(seen) != 3 || seen[0].Total != 3 || seen[2].Done != 2 || seen[0].File == "" {
		t.Fatalf("progress = %+v", seen)
	}
	if _, err := os.Stat(progressPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("the progress file must be removed when the run ends")
	}
}

func TestRescanMarkerIsLeftOnlyWhenBooksWereAdded(t *testing.T) {
	fake := newFakeGitHub(t, sampleFiles())
	e := newEnv(t, "")
	marker := filepath.Join(t.TempDir(), "rescan.flag")
	options := e.options(t, fake)
	options.RescanFlag = marker
	_, err := Run(context.Background(), options)
	must(t, err)
	if _, err := os.Stat(marker); err != nil {
		t.Fatal("books were added but no marker was left")
	}
	must(t, os.Remove(marker))
	_, err = Run(context.Background(), options)
	must(t, err)
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("a run that added nothing must not ask for a rescan")
	}
}

func TestBrokenRoutingRulesStopTheRunInsteadOfDownloadingUnchecked(t *testing.T) {
	root := t.TempDir()
	defer storage.SetRoot(root)()
	fake := newFakeGitHub(t, sampleFiles())
	e := newEnv(t, "")
	options := e.options(t, fake)
	options.Rules = nil
	_, err := Run(context.Background(), options)
	if !hasMessageKey(err, "msg.err_rules_unreadable") || atomic.LoadInt32(&fake.downloads) != 0 {
		t.Fatalf("got %v, %d downloads", err, fake.downloads)
	}
	var _ = inbox.Rules{}
}
