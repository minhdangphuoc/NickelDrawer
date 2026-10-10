package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"testing"

	"koboapps/internal/release"
)

// modArchive builds a KoboRoot.tgz holding the given regular files.
func modArchive(t *testing.T, names ...string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)
	for _, name := range names {
		content := []byte("library " + name)
		if err := tarWriter.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(content)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tarWriter.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

const modLibrary = "./usr/local/Kobo/imageformats/libnickeldrawer.so"

// publishWithMod serves version as the stable release with the given mod archive inside.
func (f *fixture) publishWithMod(version string, archive []byte) {
	dist := f.t.TempDir()
	writeVersionTree(f.t, dist, version, version)
	writeFile(f.t, dist, release.ModPath, string(archive), 0o644)
	f.github.publishStable(version, buildRelease(f.t, dist, version, f.privateKey, ""))
}

func (f *fixture) stagedMod() ([]byte, bool) {
	data, err := os.ReadFile(filepath.Join(f.root, ".kobo", "KoboRoot.tgz"))
	return data, err == nil
}

func TestUpdateStagesTheModForTheNextReboot(t *testing.T) {
	f := newFixture(t, "1.0.0")
	archive := modArchive(t, modLibrary)
	f.publishWithMod("1.1.0", archive)

	result, err := f.updater("1.0.0").Apply(context.Background())
	if err != nil || result.Key != "update.applied_reboot" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	staged, found := f.stagedMod()
	if !found || !bytes.Equal(staged, archive) {
		t.Fatalf("staged KoboRoot.tgz missing or different (found=%v)", found)
	}
	installed, err := os.ReadFile(filepath.Join(f.root, filepath.FromSlash(release.ModPath)))
	if err != nil || !bytes.Equal(installed, archive) {
		t.Fatalf("the record of the staged mod is missing: %v", err)
	}
}

func TestModWithForeignFilesIsRejectedBeforeAnythingChanges(t *testing.T) {
	f := newFixture(t, "1.0.0")
	f.publishWithMod("1.1.0", modArchive(t, modLibrary, "./etc/init.d/rcS"))
	before := snapshot(t, f.root, bookkeeping)

	_, err := f.updater("1.0.0").Apply(context.Background())
	if messageKey(err) != "update.bad_mod" {
		t.Fatalf("err = %v", err)
	}
	sameSnapshot(t, "tree", snapshot(t, f.root, bookkeeping), before)
	if _, found := f.stagedMod(); found {
		t.Fatal("a rejected mod was staged")
	}
}

func TestADifferentWaitingKoboRootBlocksTheUpdate(t *testing.T) {
	f := newFixture(t, "1.0.0")
	writeFile(t, f.root, ".kobo/KoboRoot.tgz", "someone else's update", 0o644)
	f.publishWithMod("1.1.0", modArchive(t, modLibrary))
	before := snapshot(t, f.root, bookkeeping)

	_, err := f.updater("1.0.0").Apply(context.Background())
	if messageKey(err) != "update.mod_busy" {
		t.Fatalf("err = %v", err)
	}
	sameSnapshot(t, "tree", snapshot(t, f.root, bookkeeping), before)
}

func TestAnUnchangedModIsNotStagedAgain(t *testing.T) {
	f := newFixture(t, "1.0.0")
	archive := modArchive(t, modLibrary)
	f.publishWithMod("1.1.0", archive)
	if _, err := f.updater("1.0.0").Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(f.root, ".kobo", "KoboRoot.tgz")); err != nil { // Nickel consumed it
		t.Fatal(err)
	}
	f.publishWithMod("1.2.0", archive)

	result, err := f.updater("1.1.0").Apply(context.Background())
	if err != nil || result.Key != "update.applied" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if _, found := f.stagedMod(); found {
		t.Fatal("the same mod was staged twice")
	}
}

func TestRollbackWithdrawsAModThatIsStillWaiting(t *testing.T) {
	f := newFixture(t, "1.0.0")
	f.publishWithMod("1.1.0", modArchive(t, modLibrary))
	if _, err := f.updater("1.0.0").Apply(context.Background()); err != nil {
		t.Fatal(err)
	}

	if _, err := f.updater("1.1.0").Rollback(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, found := f.stagedMod(); found {
		t.Fatal("rollback left the newer mod waiting for the next reboot")
	}
}

func TestCheckSavesWhatItFoundForTheDrawer(t *testing.T) {
	f := newFixture(t, "1.0.0")
	dist := t.TempDir()
	writeVersionTree(t, dist, "1.1.0", "1.1.0")
	writeFile(t, dist, release.ModPath, string(modArchive(t, modLibrary)), 0o644)
	f.github.publishStable("1.1.0", buildReleaseWithNotes(t, dist, "1.1.0", f.privateKey, "", "Faster library scan."))
	layout := Layout{OnboardRoot: f.root}
	before := snapshot(t, f.root, bookkeeping)

	if _, err := f.updater("1.0.0").Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	found, ok := ReadAvailability(layout)
	if !ok || !found.IsAvailable || found.Version != "1.1.0" || found.Notes != "Faster library scan." || !found.HasMod {
		t.Fatalf("availability = %+v ok=%v", found, ok)
	}
	if !found.IsNewerThan("1.0.0") || found.IsNewerThan("1.1.0") {
		t.Fatal("IsNewerThan must be true only for an older installed version")
	}
	sameSnapshot(t, "tree outside the bookkeeping", snapshot(t, f.root, bookkeeping), before)
	if _, err := f.updater("1.1.0").Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	if found, _ := ReadAvailability(layout); found.IsAvailable {
		t.Fatalf("an up-to-date check must clear the update: %+v", found)
	}
}

func TestCheckKeepsTheResultOfTheLastInstall(t *testing.T) {
	f := newFixture(t, "1.0.0")
	f.publishWithMod("1.1.0", modArchive(t, modLibrary))
	if _, err := f.updater("1.0.0").Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := f.updater("1.1.0").Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	status, _, _ := ReadStatus(Layout{OnboardRoot: f.root})
	if status.Report.Key != "update.applied_reboot" {
		t.Fatalf("a later check hid the restart reminder: %+v", status)
	}
}

func TestApplyForgetsTheCachedCheck(t *testing.T) {
	f := newFixture(t, "1.0.0")
	f.publishVersion("1.1.0", "1.1.0")
	if _, err := f.updater("1.0.0").Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := f.updater("1.0.0").Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, ok := ReadAvailability(Layout{OnboardRoot: f.root}); ok {
		t.Fatal("the cached check outlived the install")
	}
}
