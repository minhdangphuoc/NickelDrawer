package main

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func gradient(t *testing.T, width, height int) []byte {
	t.Helper()
	picture := image.NewGray(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			picture.SetGray(x, y, color.Gray{Y: uint8(x * 255 / width)})
		}
	}
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, picture); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

type gallery struct {
	root, appDir string
	common       []string // flags every call needs
}

// importedGallery has three originals in Pictures (a, b, c) imported as 0001..0003.
func importedGallery(t *testing.T) gallery {
	t.Helper()
	root, appDir, versionFile := deviceRoot(t)
	write(t, filepath.Join(root, "Pictures", "b.png"), gradient(t, 80, 60))
	write(t, filepath.Join(root, "Pictures", "c.png"), gradient(t, 60, 80))
	g := gallery{root: root, appDir: appDir, common: []string{"--app-dir", appDir, "--locale", "en", "--version-file", versionFile,
		"--conf", filepath.Join(appDir, "Kobo eReader.conf"), "--progress-file", filepath.Join(appDir, "progress.json"),
		"--last-file", filepath.Join(appDir, "none.txt"), "--pid-file", filepath.Join(appDir, "none.pid")}}
	if status, output := g.run(t, "import"); status != 0 {
		t.Fatalf("import: %s", output)
	}
	return g
}

// run puts the flags first, then the positional arguments (the CLI's rule).
func (g gallery) run(t *testing.T, command string, rest ...string) (int, string) {
	t.Helper()
	var flags, positional []string
	for _, word := range rest {
		if strings.HasPrefix(word, "--") {
			flags = append(flags, word)
		} else {
			positional = append(positional, word)
		}
	}
	args := append([]string{command}, g.common...)
	return run(t, append(append(args, flags...), positional...)...)
}

func (g gallery) state(t *testing.T) stateReply {
	t.Helper()
	status, output := g.run(t, "state")
	if status != 0 {
		t.Fatalf("state: %s", output)
	}
	var reply stateReply
	if err := json.Unmarshal([]byte(output), &reply); err != nil {
		t.Fatal(err)
	}
	return reply
}

func TestStateListsPicturesWithThumbsBadgeAndInfo(t *testing.T) {
	g := importedGallery(t)
	reply := g.state(t)
	if reply.Count != 3 || len(reply.Pictures) != 3 || reply.CurrentID != "0003" || reply.Pending != 0 || reply.ImportStep != "" {
		t.Fatalf("state = %+v", reply)
	}
	for position, want := range []string{"0001", "0002", "0003"} {
		item := reply.Pictures[position]
		if item.ID != want {
			t.Fatalf("picture %d id = %s", position, item.ID)
		}
		if !filepath.IsAbs(item.Thumb) || !filepath.IsAbs(item.Image) {
			t.Errorf("paths must be absolute: %+v", item)
		}
		for _, path := range []string{item.Thumb, item.Image} {
			if _, err := os.Stat(path); err != nil {
				t.Errorf("%s: %v", path, err)
			}
		}
		if (item.Badge == "current") != (want == "0003") {
			t.Errorf("badge of %s = %q", want, item.Badge)
		}
	}
	first := reply.Pictures[0]
	if first.Label != "a" || !strings.HasPrefix(first.Info, "Name: a.png\nSize: 90 × 60\nAdded: ") {
		t.Errorf("label %q info %q", first.Label, first.Info)
	}
	if reply.Summary != "3 pictures · In use: c · Fill" || reply.SleepScreenLine != enSleepHint {
		t.Errorf("summary %q line %q", reply.Summary, reply.SleepScreenLine)
	}
}

func TestStateSaysReadyWhenNickelSettingsAreOn(t *testing.T) {
	g := importedGallery(t)
	write(t, filepath.Join(g.appDir, "Kobo eReader.conf"), []byte("[PowerOptions]\nShowBookCover=true\nFullscreenCoverStretch=true\n"))
	if reply := g.state(t); reply.SleepScreenLine != "Ready ✓" {
		t.Errorf("line = %q", reply.SleepScreenLine)
	}
}

func TestStateShowsImportProgress(t *testing.T) {
	g := importedGallery(t)
	write(t, filepath.Join(g.appDir, "progress.json"), []byte(`{"done":3,"total":7,"name":"bien-xanh.jpg"}`))
	reply := g.state(t)
	if reply.Status != "Importing 3/7" || reply.ImportStep != "Converting 3/7 · bien-xanh.jpg" || reply.Pending != 4 || !reply.Busy {
		t.Fatalf("state = %+v", reply)
	}
	write(t, filepath.Join(g.appDir, "progress.json"), []byte(`{"done":2,"total":7,"name":"x.png","kind":"refit"}`))
	if reply := g.state(t); reply.Status != "Converting 2/7" || reply.ImportStep != "Re-fitting 2/7 · x.png" {
		t.Fatalf("refit state = %+v", reply)
	}
}

func TestImportLeavesNoProgressFile(t *testing.T) {
	g := importedGallery(t)
	if _, err := os.Stat(filepath.Join(g.appDir, "progress.json")); err == nil {
		t.Error("progress file must be removed after the import")
	}
}

func TestStateWorksWhileAJobIsAlive(t *testing.T) {
	g := importedGallery(t)
	write(t, filepath.Join(g.appDir, "run.pid"), []byte(strconvPid()))
	status, output := run(t, "state", "--app-dir", g.appDir, "--locale", "en", "--pid-file", filepath.Join(g.appDir, "run.pid"), "--progress-file", filepath.Join(g.appDir, "none.json"))
	if status != 0 || !strings.Contains(output, `"busy":true`) {
		t.Fatalf("state = %d %s", status, output)
	}
}

func TestSetCurrentAcceptsPaddedAndPlainIds(t *testing.T) {
	g := importedGallery(t)
	for _, id := range []string{"0001", "2"} {
		if status, output := g.run(t, "set-current", id); status != 0 {
			t.Fatalf("set-current %s: %s", id, output)
		}
	}
	if reply := g.state(t); reply.CurrentID != "0002" {
		t.Errorf("current = %s", reply.CurrentID)
	}
	for _, bad := range []string{"9", "abc", "0"} {
		if status, output := g.run(t, "set-current", bad); status != 1 || !strings.HasPrefix(output, "Error: ") {
			t.Errorf("set-current %s = %d %q", bad, status, output)
		}
	}
}

func TestRemoveNeedsConfirmAndKeepsOriginal(t *testing.T) {
	g := importedGallery(t)
	status, output := g.run(t, "remove", "0003")
	if status != 1 || !strings.Contains(output, "--confirm") || g.state(t).Count != 3 {
		t.Fatalf("without --confirm: %d %q", status, output)
	}
	if status, output = g.run(t, "remove", "--confirm", "0003"); status != 0 {
		t.Fatalf("remove: %s", output)
	}
	reply := g.state(t)
	if reply.Count != 2 || reply.CurrentID != "0002" {
		t.Fatalf("state = %+v, want the previous picture current", reply)
	}
	if _, err := os.Stat(filepath.Join(g.root, "Pictures", "c.png")); err != nil {
		t.Error("the original must stay")
	}
	if _, err := os.Stat(filepath.Join(g.root, ".kobo", "screensaver", "nd-wallpaper-0003.png")); err == nil {
		t.Error("our screensaver file must go")
	}
	if status, _ = g.run(t, "import"); status != 0 || g.state(t).Count != 2 {
		t.Error("a removed picture must not be imported again")
	}
}

func TestRemoveDeleteOriginalOnlyWhenAsked(t *testing.T) {
	g := importedGallery(t)
	if status, output := g.run(t, "remove", "--confirm", "--delete-original", "0002"); status != 0 {
		t.Fatalf("remove: %s", output)
	}
	if _, err := os.Stat(filepath.Join(g.root, "Pictures", "b.png")); err == nil {
		t.Error("--delete-original must delete the matching original")
	}
	if _, err := os.Stat(filepath.Join(g.root, "Pictures", "c.png")); err != nil {
		t.Error("other originals must stay")
	}
}

func TestRotationThenModeBothPersist(t *testing.T) {
	g := importedGallery(t)
	if status, output := g.run(t, "rotation", "random"); status != 0 {
		t.Fatal(output)
	}
	if status, output := g.run(t, "mode", "fit"); status != 0 {
		t.Fatal(output)
	}
	if reply := g.state(t); reply.Rotation != "random" || reply.Mode != "fit" {
		t.Errorf("state = %+v", reply)
	}
}

func TestModeSaveOnlyStoresWithoutRefit(t *testing.T) {
	g := importedGallery(t)
	before, _ := os.ReadFile(filepath.Join(g.appDir, "data", "images", "0001.png"))
	if status, output := g.run(t, "mode", "--save-only", "fit"); status != 0 {
		t.Fatal(output)
	}
	after, _ := os.ReadFile(filepath.Join(g.appDir, "data", "images", "0001.png"))
	if !bytes.Equal(before, after) || g.state(t).Mode != "fit" {
		t.Error("save-only must store the mode and leave the pictures")
	}
}

func strconvPid() string { return strconv.Itoa(os.Getpid()) }
