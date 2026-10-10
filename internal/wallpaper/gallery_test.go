package wallpaper

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// rigWithPictures imports three pictures: ids 0001 (a), 0002 (b), 0003 (c).
func rigWithPictures(t *testing.T) rig {
	t.Helper()
	r := newRig(t)
	put(t, filepath.Join(r.pictures, "a.png"), gradientPNG(t, 50, 70))
	put(t, filepath.Join(r.pictures, "b.jpg"), solidJPEG(t, 60, 60, 90))
	put(t, filepath.Join(r.pictures, "c.png"), gradientPNG(t, 70, 40))
	if _, _, err := r.service.ImportAndApply(); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestImportWritesThumbnailsFittedInside300x400(t *testing.T) {
	r := rigWithPictures(t)
	for id := 1; id <= 3; id++ {
		content, err := os.ReadFile(r.service.Paths.Thumb(id))
		if err != nil {
			t.Fatalf("thumbnail %d: %v", id, err)
		}
		config, err := png.DecodeConfig(bytes.NewReader(content))
		if err != nil || config.Width > 300 || config.Height > 400 {
			t.Fatalf("thumbnail %d = %+v, %v", id, config, err)
		}
		if config.Width != 300 && config.Height != 400 {
			t.Errorf("thumbnail %d %dx%d does not touch the 300x400 box", id, config.Width, config.Height)
		}
		gray := decodeGray(t, content)
		for _, pixel := range gray.Pix {
			if pixel%17 != 0 {
				t.Fatalf("thumbnail %d value %d is not one of 16 levels", id, pixel)
			}
		}
	}
}

func TestEntriesRecordOriginalSizeAndAddedTime(t *testing.T) {
	r := rigWithPictures(t)
	library, err := LoadLibrary(r.service.Paths)
	if err != nil {
		t.Fatal(err)
	}
	entry, _ := library.find(1)
	if entry.Width != 50 || entry.Height != 70 || entry.Added == "" {
		t.Errorf("entry = %+v", entry)
	}
}

func TestStateCreatesAtMostTwoMissingThumbnailsPerCall(t *testing.T) {
	r := rigWithPictures(t)
	for id := 1; id <= 3; id++ {
		os.Remove(r.service.Paths.Thumb(id))
	}
	missing := func(state ScreenState) int {
		count := 0
		for _, picture := range state.Pictures {
			if picture.Thumb == "" {
				count++
			}
		}
		return count
	}
	state, err := r.service.ScreenState(ScreenStateFiles{})
	if err != nil || missing(state) != 3-maxLazyThumbnails {
		t.Fatalf("first state: %d missing, %v", missing(state), err)
	}
	state, err = r.service.ScreenState(ScreenStateFiles{})
	if err != nil || missing(state) != 0 {
		t.Fatalf("second state: %d missing, %v", missing(state), err)
	}
}

func TestModeRefitDeletesThumbnailAndStateRegeneratesIt(t *testing.T) {
	r := rigWithPictures(t)
	if _, _, err := r.service.SetMode(ModeFit); err != nil {
		t.Fatal(err)
	}
	for id := 1; id <= 3; id++ {
		if _, err := os.Stat(r.service.Paths.Thumb(id)); err == nil {
			t.Errorf("thumbnail %d must be dropped by the re-fit", id)
		}
	}
	if _, err := r.service.ScreenState(ScreenStateFiles{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(r.service.Paths.Thumb(1)); err != nil {
		t.Errorf("state must regenerate a thumbnail: %v", err)
	}
}

func TestProgressFileIsWrittenDuringImportAndRemovedAtTheEnd(t *testing.T) {
	r := newRig(t)
	r.service.ProgressFile = filepath.Join(t.TempDir(), "progress.json")
	put(t, filepath.Join(r.pictures, "a.png"), gradientPNG(t, 50, 70))
	put(t, filepath.Join(r.pictures, "b.png"), gradientPNG(t, 70, 50))
	library, _, _ := r.service.load()
	if _, err := r.service.importWithProgress(library, ModeFill, ProgressImport); err != nil {
		t.Fatal(err)
	}
	progress, found, isFresh := ReadProgress(r.service.ProgressFile)
	if !found || !isFresh || progress.Done != 2 || progress.Total != 2 || progress.Name != "b.png" {
		t.Fatalf("progress = %+v found=%v fresh=%v", progress, found, isFresh)
	}
	if _, _, err := r.service.ImportAndApply(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(r.service.ProgressFile); err == nil {
		t.Error("the progress file must be removed when the import ends")
	}
}

func TestSetCurrentAppliesAndRejectsUnknown(t *testing.T) {
	r := rigWithPictures(t)
	if _, err := r.service.SetCurrent(1); err != nil {
		t.Fatal(err)
	}
	if got := listing(t, r.screensaver); len(got) != 1 || got[0] != "nd-wallpaper-0001.png" {
		t.Errorf("screensaver = %v", got)
	}
	if _, err := r.service.SetCurrent(9); err == nil {
		t.Error("an unknown id must fail")
	}
}

func TestRemoveMovesCurrentToNextElsePreviousAndKeepsOriginal(t *testing.T) {
	r := rigWithPictures(t) // current is 3, the newest
	r.service.SetCurrent(2)
	if _, err := r.service.Remove(2, false); err != nil {
		t.Fatal(err)
	}
	config, _ := LoadConfig(r.service.Paths.Config())
	if config.Current != 3 {
		t.Errorf("current = %d, want the next picture 3", config.Current)
	}
	if _, err := r.service.Remove(3, false); err != nil {
		t.Fatal(err)
	}
	config, _ = LoadConfig(r.service.Paths.Config())
	if config.Current != 1 {
		t.Errorf("current = %d, want the previous picture 1", config.Current)
	}
	for _, path := range []string{r.service.Paths.Image(2), r.service.Paths.Thumb(2), filepath.Join(r.screensaver, "nd-wallpaper-0002.png")} {
		if _, err := os.Stat(path); err == nil {
			t.Errorf("%s must be gone", path)
		}
	}
	for _, name := range []string{"a.png", "b.jpg", "c.png"} {
		if _, err := os.Stat(filepath.Join(r.pictures, name)); err != nil {
			t.Errorf("original %s must stay: %v", name, err)
		}
	}
	if got := listing(t, r.screensaver); len(got) != 1 || got[0] != "nd-wallpaper-0001.png" {
		t.Errorf("screensaver = %v", got)
	}
}

func TestRemovedPictureIsNotImportedAgainButNewOnesAre(t *testing.T) {
	r := rigWithPictures(t)
	if _, err := r.service.Remove(2, false); err != nil {
		t.Fatal(err)
	}
	put(t, filepath.Join(r.pictures, "d.png"), gradientPNG(t, 40, 80))
	report, _, err := r.service.ImportAndApply()
	if err != nil || report.Added != 1 || report.Skipped != 1 {
		t.Fatalf("import = %+v, %v", report, err)
	}
	library, _ := LoadLibrary(r.service.Paths)
	if _, found := library.find(2); found {
		t.Error("the removed picture came back")
	}
}

func TestRemoveOfLastPictureTurnsSleepScreenOffAndLeavesForeignFiles(t *testing.T) {
	r := newRig(t)
	put(t, filepath.Join(r.pictures, "only.png"), gradientPNG(t, 50, 70))
	put(t, filepath.Join(r.screensaver, "mine.png"), []byte("user file"))
	if _, _, err := r.service.ImportAndApply(); err != nil {
		t.Fatal(err)
	}
	if _, err := r.service.Remove(1, false); err != nil {
		t.Fatal(err)
	}
	if got := listing(t, r.screensaver); len(got) != 1 || got[0] != "mine.png" {
		t.Errorf("screensaver = %v, only the foreign file may remain", got)
	}
	content, _ := os.ReadFile(filepath.Join(r.screensaver, "mine.png"))
	if string(content) != "user file" {
		t.Error("foreign file changed")
	}
}

func TestDeleteOriginalOnlyForMatchingDirectChildRegularFile(t *testing.T) {
	r := rigWithPictures(t)
	report, err := r.service.Remove(1, true)
	if err != nil || !report.OriginalDeleted {
		t.Fatalf("report = %+v, %v", report, err)
	}
	if _, err := os.Stat(filepath.Join(r.pictures, "a.png")); err == nil {
		t.Error("the matching original must be deleted")
	}
	if _, err := os.Stat(filepath.Join(r.pictures, "b.jpg")); err != nil {
		t.Error("other originals must stay")
	}
}

func TestDeleteOriginalKeepsFileWhoseContentChanged(t *testing.T) {
	r := rigWithPictures(t)
	put(t, filepath.Join(r.pictures, "a.png"), gradientPNG(t, 33, 33)) // same name, other picture
	report, err := r.service.Remove(1, true)
	if err != nil || report.OriginalDeleted || !report.OriginalNotFound {
		t.Fatalf("report = %+v, %v", report, err)
	}
	if _, err := os.Stat(filepath.Join(r.pictures, "a.png")); err != nil {
		t.Error("a file with another hash must stay")
	}
}

func TestDeleteOriginalRefusesSymlinks(t *testing.T) {
	r := newRig(t)
	content := gradientPNG(t, 50, 70)
	put(t, filepath.Join(r.pictures, "real.png"), content)
	if _, _, err := r.service.ImportAndApply(); err != nil {
		t.Fatal(err)
	}
	// The imported file is renamed away and replaced by a link of the same name to a copy elsewhere.
	outside := filepath.Join(t.TempDir(), "elsewhere.png")
	put(t, outside, content)
	os.Remove(filepath.Join(r.pictures, "real.png"))
	if err := os.Symlink(outside, filepath.Join(r.pictures, "real.png")); err != nil {
		t.Skip(err)
	}
	report, err := r.service.Remove(1, true)
	if err != nil || report.OriginalDeleted {
		t.Fatalf("report = %+v, %v", report, err)
	}
	if _, err := os.Stat(outside); err != nil {
		t.Error("the link target was deleted")
	}
	if _, err := os.Lstat(filepath.Join(r.pictures, "real.png")); err != nil {
		t.Error("the link was deleted")
	}
}

func TestDeleteOriginalRefusesLinkInsideSameFolder(t *testing.T) {
	r := newRig(t)
	content := gradientPNG(t, 50, 70)
	put(t, filepath.Join(r.pictures, "real.png"), content)
	if _, _, err := r.service.ImportAndApply(); err != nil {
		t.Fatal(err)
	}
	// real.png (the entry's name) becomes a link to another file in the same folder.
	put(t, filepath.Join(r.pictures, "copy.png"), content)
	os.Remove(filepath.Join(r.pictures, "real.png"))
	if err := os.Symlink("copy.png", filepath.Join(r.pictures, "real.png")); err != nil {
		t.Skip(err)
	}
	report, _ := r.service.Remove(1, true)
	if report.OriginalDeleted {
		t.Error("a link must not count as the original")
	}
	if _, err := os.Stat(filepath.Join(r.pictures, "copy.png")); err != nil {
		t.Error("the link target was deleted")
	}
}

func TestRotationThenModeBothPersist(t *testing.T) {
	r := rigWithPictures(t)
	if _, err := r.service.SetRotation(RotationRandom); err != nil {
		t.Fatal(err)
	}
	if _, _, err := r.service.SetMode(ModeFit); err != nil {
		t.Fatal(err)
	}
	config, _ := LoadConfig(r.service.Paths.Config())
	if config.Rotation != RotationRandom || config.Mode != ModeFit {
		t.Errorf("config = %+v", config)
	}
}

func TestModeChangedDuringImportIsRefittedWhenItEnds(t *testing.T) {
	r := rigWithPictures(t)
	library, _ := LoadLibrary(r.service.Paths)
	if err := r.service.SaveMode(ModeFit); err != nil { // the user tapped Fit while a fill job ran
		t.Fatal(err)
	}
	report, err := r.service.importSettled(library, ModeFill, ProgressImport)
	if err != nil || report.Reconverted != 3 {
		t.Fatalf("report = %+v, %v", report, err)
	}
	library, _ = LoadLibrary(r.service.Paths)
	for _, entry := range library.Entries {
		if entry.Mode != ModeFit {
			t.Errorf("entry %d is %s, want fit", entry.ID, entry.Mode)
		}
	}
}

func TestScreenStateReadsProgressOnlyWhileFreshOrAlive(t *testing.T) {
	r := rigWithPictures(t)
	progressFile := filepath.Join(t.TempDir(), "progress.json")
	if err := writeProgress(progressFile, Progress{Done: 3, Total: 7, Name: "x.png"}); err != nil {
		t.Fatal(err)
	}
	state, err := r.service.ScreenState(ScreenStateFiles{ProgressFile: progressFile})
	if err != nil || state.Progress == nil || !state.Busy || state.Progress.Done != 3 {
		t.Fatalf("state = %+v, %v", state, err)
	}
	stale := filepath.Join(t.TempDir(), "stale.json")
	if err := writeProgress(stale, Progress{Done: 1, Total: 2}); err != nil {
		t.Fatal(err)
	}
	long := staleProgressAfter + 1
	if err := os.Chtimes(stale, timeAgo(long), timeAgo(long)); err != nil {
		t.Fatal(err)
	}
	state, _ = r.service.ScreenState(ScreenStateFiles{ProgressFile: stale})
	if state.Progress != nil || state.Busy {
		t.Errorf("a stale progress file must be ignored: %+v", state)
	}
}

func timeAgo(age time.Duration) time.Time { return time.Now().Add(-age) }
