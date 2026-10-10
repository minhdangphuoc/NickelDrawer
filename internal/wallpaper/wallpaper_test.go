package wallpaper

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"koboapps/internal/i18n"
)

var libra2 = Size{Width: 1264, Height: 1680}

func gradientPNG(t *testing.T, width, height int) []byte {
	t.Helper()
	picture := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			shade := uint8(x * 255 / max(width-1, 1))
			picture.Set(x, y, color.RGBA{shade, shade, shade, 255})
		}
	}
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, picture); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func solidJPEG(t *testing.T, width, height int, shade uint8) []byte {
	t.Helper()
	picture := image.NewGray(image.Rect(0, 0, width, height))
	for index := range picture.Pix {
		picture.Pix[index] = shade
	}
	var buffer bytes.Buffer
	if err := jpeg.Encode(&buffer, picture, nil); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func decodeGray(t *testing.T, content []byte) *image.Gray {
	t.Helper()
	decoded, err := png.Decode(bytes.NewReader(content))
	if err != nil {
		t.Fatal(err)
	}
	gray, isGray := decoded.(*image.Gray)
	if !isGray {
		t.Fatalf("output is %T, want *image.Gray", decoded)
	}
	return gray
}

func TestConvertProducesScreenSizedSixteenLevelGray(t *testing.T) {
	for _, mode := range []Mode{ModeFill, ModeFit} {
		output, err := Convert(gradientPNG(t, 640, 480), libra2, mode)
		if err != nil {
			t.Fatal(err)
		}
		gray := decodeGray(t, output)
		if gray.Rect.Dx() != 1264 || gray.Rect.Dy() != 1680 {
			t.Fatalf("%s size = %v", mode, gray.Rect)
		}
		levels := map[uint8]bool{}
		for _, pixel := range gray.Pix {
			if pixel%17 != 0 {
				t.Fatalf("%s: value %d is not one of 16 levels", mode, pixel)
			}
			levels[pixel] = true
		}
		if mode == ModeFill && len(levels) < 8 {
			t.Errorf("dithered gradient should use most levels, got %d", len(levels))
		}
	}
}

func TestDitheringKeepsMidGrayAverage(t *testing.T) {
	output, err := Convert(solidJPEG(t, 100, 100, 128), Size{Width: 200, Height: 200}, ModeFill)
	if err != nil {
		t.Fatal(err)
	}
	gray := decodeGray(t, output)
	sum, distinct := 0, map[uint8]bool{}
	for _, pixel := range gray.Pix {
		sum += int(pixel)
		distinct[pixel] = true
	}
	average := sum / len(gray.Pix)
	if average < 124 || average > 132 {
		t.Errorf("average = %d, want about 128", average)
	}
	if len(distinct) < 2 {
		t.Error("mid gray must be dithered between two levels")
	}
}

func TestFitLetterboxesOnWhiteAndFillCovers(t *testing.T) {
	black := solidJPEG(t, 400, 100, 0)
	fit := decodeGray(t, mustConvert(t, black, ModeFit))
	if fit.GrayAt(632, 5).Y != 255 || fit.GrayAt(632, 1674).Y != 255 {
		t.Error("fit must leave white bars above and below a wide picture")
	}
	if fit.GrayAt(632, 840).Y != 0 {
		t.Error("fit must show the picture in the middle")
	}
	fill := decodeGray(t, mustConvert(t, black, ModeFill))
	for _, corner := range [][2]int{{0, 0}, {1263, 0}, {0, 1679}, {1263, 1679}} {
		if fill.GrayAt(corner[0], corner[1]).Y != 0 {
			t.Errorf("fill must cover the corner %v", corner)
		}
	}
}

func mustConvert(t *testing.T, content []byte, mode Mode) []byte {
	t.Helper()
	output, err := Convert(content, libra2, mode)
	if err != nil {
		t.Fatal(err)
	}
	return output
}

func TestConvertRefusesNonImages(t *testing.T) {
	for name, content := range map[string][]byte{
		"text": []byte("hello world, not a picture"), "empty": nil,
		"gif": []byte("GIF89a\x01\x00\x01\x00"), "truncated png": gradientPNG(t, 10, 10)[:30],
	} {
		if _, err := Convert(content, libra2, ModeFill); err == nil {
			t.Errorf("%s must be refused", name)
		}
	}
}

func TestResolutionTable(t *testing.T) {
	size, err := ResolutionFor("00000000-0000-0000-0000-000000000388")
	if err != nil || size != libra2 {
		t.Fatalf("Libra 2 = %v, %v", size, err)
	}
	if _, err := ResolutionFor("00000000-0000-0000-0000-000000000999"); err == nil {
		t.Error("unknown device must be refused")
	}
}

func TestConfigRoundTripAndValidation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data", "config.json")
	if config, err := LoadConfig(path); err != nil || config != DefaultConfig() {
		t.Fatalf("missing file = %v, %v", config, err)
	}
	want := Config{Mode: ModeFit, Rotation: RotationRandom, Current: 7}
	if err := SaveConfig(path, want); err != nil {
		t.Fatal(err)
	}
	if got, err := LoadConfig(path); err != nil || got != want {
		t.Fatalf("round trip = %v, %v", got, err)
	}
	os.WriteFile(path, []byte(`{"mode":"stretch"}`), 0o644)
	if _, err := LoadConfig(path); err == nil {
		t.Error("unknown mode must be rejected")
	}
	if _, err := ParseRotation("shuffle"); err == nil {
		t.Error("unknown rotation must be rejected")
	}
}

type rig struct {
	service     *Service
	pictures    string
	screensaver string
}

func newRig(t *testing.T) rig {
	t.Helper()
	root := t.TempDir()
	r := rig{
		pictures:    filepath.Join(root, "Pictures"),
		screensaver: filepath.Join(root, ".kobo", "screensaver"),
	}
	service := NewService(Paths{AppDir: filepath.Join(root, "app")}, Size{Width: 120, Height: 160}, nil)
	service.Sources = []string{r.pictures}
	service.ScreensaverDir = r.screensaver
	r.service = service
	return r
}

func put(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
}

func listing(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}

func TestImportConvertsDedupesAndNeverTouchesOriginals(t *testing.T) {
	r := newRig(t)
	one, two := gradientPNG(t, 50, 70), solidJPEG(t, 60, 60, 90)
	put(t, filepath.Join(r.pictures, "one.png"), one)
	put(t, filepath.Join(r.pictures, "same-as-one.png"), one) // identical content in the same folder
	put(t, filepath.Join(r.pictures, "two.jpg"), two)
	put(t, filepath.Join(r.pictures, "book.epub"), []byte("PK\x03\x04 book"))
	put(t, filepath.Join(r.pictures, "Holiday", "deep.png"), one)

	imported, applied, err := r.service.ImportAndApply()
	if err != nil {
		t.Fatal(err)
	}
	if imported.Added != 2 || imported.Duplicates != 1 || applied.Shown != 1 {
		t.Fatalf("import = %+v, apply = %+v", imported, applied)
	}
	if got := listing(t, r.service.Paths.Images()); len(got) != 3 { // 2 pictures + index
		t.Errorf("images = %v", got)
	}
	if got := listing(t, r.screensaver); len(got) != 1 || got[0] != "nd-wallpaper-0002.png" {
		t.Errorf("screensaver = %v (fixed shows the newest import)", got)
	}
	again, _, err := r.service.ImportAndApply()
	if err != nil || again.Added != 0 {
		t.Errorf("second import = %+v, %v", again, err)
	}
	after, _ := os.ReadFile(filepath.Join(r.pictures, "one.png"))
	if !bytes.Equal(after, one) {
		t.Error("original was modified")
	}
}

func TestImportIgnoresSymlinkEscapeAndNonImages(t *testing.T) {
	r := newRig(t)
	secret := filepath.Join(t.TempDir(), "secret.png")
	put(t, secret, gradientPNG(t, 30, 30))
	put(t, filepath.Join(r.pictures, "ok.png"), solidJPEG(t, 30, 30, 200))
	if err := os.Symlink(secret, filepath.Join(r.pictures, "link.png")); err != nil {
		t.Skip(err)
	}
	put(t, filepath.Join(r.pictures, "notes.txt"), []byte("hello"))
	imported, _, err := r.service.ImportAndApply()
	if err != nil || imported.Added != 1 {
		t.Fatalf("import = %+v, %v", imported, err)
	}
}

func TestImportContinuesPastBrokenPicture(t *testing.T) {
	r := newRig(t)
	put(t, filepath.Join(r.pictures, "broken.png"), append([]byte("\x89PNG\r\n\x1a\n"), []byte("garbage")...))
	put(t, filepath.Join(r.pictures, "good.png"), gradientPNG(t, 30, 30))
	imported, _, err := r.service.ImportAndApply()
	if err != nil || imported.Added != 1 || imported.Failed != 1 {
		t.Fatalf("import = %+v, %v", imported, err)
	}
}

func TestApplyTouchesOnlyOwnedFiles(t *testing.T) {
	r := newRig(t)
	put(t, filepath.Join(r.pictures, "a.png"), gradientPNG(t, 30, 30))
	put(t, filepath.Join(r.pictures, "b.png"), solidJPEG(t, 30, 30, 50))
	foreign := map[string][]byte{"user.png": []byte("mine"), "nd-wallpaperish.txt": nil, "sleep.jpg": []byte("x")}
	for name, content := range foreign {
		put(t, filepath.Join(r.screensaver, name), content)
	}
	put(t, filepath.Join(r.screensaver, "nd-wallpaper-0099.png"), []byte("stale ours"))
	if _, _, err := r.service.ImportAndApply(); err != nil {
		t.Fatal(err)
	}
	got := listing(t, r.screensaver)
	for name, want := range foreign {
		content, err := os.ReadFile(filepath.Join(r.screensaver, name))
		if err != nil || !bytes.Equal(content, want) {
			t.Errorf("foreign file %s changed: %v", name, err)
		}
	}
	if !contains(got, "nd-wallpaper-0002.png") || contains(got, "nd-wallpaper-0099.png") || contains(got, "nd-wallpaper-0001.png") {
		t.Errorf("fixed must hold exactly the current picture, got %v", got)
	}

	if _, err := r.service.SetRotation(RotationRandom); err != nil {
		t.Fatal(err)
	}
	got = listing(t, r.screensaver)
	if !contains(got, "nd-wallpaper-0002.png") || contains(got, "nd-wallpaper-0001.png") {
		t.Errorf("random must also hold exactly the current picture (we choose, not Nickel), got %v", got)
	}

	removed, err := r.service.TurnOff()
	if err != nil || removed != 1 {
		t.Fatalf("off removed %d, %v", removed, err)
	}
	got = listing(t, r.screensaver)
	if len(got) != len(foreign) {
		t.Errorf("off must leave the foreign files, got %v", got)
	}
}

func contains(list []string, name string) bool {
	for _, item := range list {
		if item == name {
			return true
		}
	}
	return false
}

func TestOffNeverFollowsOrDeletesSymlinks(t *testing.T) {
	r := newRig(t)
	target := filepath.Join(t.TempDir(), "precious.png")
	put(t, target, []byte("keep"))
	os.MkdirAll(r.screensaver, 0o755)
	if err := os.Symlink(target, filepath.Join(r.screensaver, "nd-wallpaper-0001.png")); err != nil {
		t.Skip(err)
	}
	if _, err := r.service.TurnOff(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Error("symlink target was touched")
	}
}

func TestNextCyclesAndModeSwitchRefitsFromOriginals(t *testing.T) {
	r := newRig(t)
	put(t, filepath.Join(r.pictures, "a.png"), gradientPNG(t, 30, 60))
	put(t, filepath.Join(r.pictures, "b.png"), solidJPEG(t, 80, 20, 0))
	if _, _, err := r.service.ImportAndApply(); err != nil {
		t.Fatal(err)
	}
	for _, want := range []int{1, 2, 1} {
		entry, err := r.service.Next()
		if err != nil || entry.ID != want {
			t.Fatalf("Next = %v, %v, want %d", entry, err, want)
		}
	}
	before, _ := os.ReadFile(r.service.Paths.Image(2))
	report, applied, err := r.service.SetMode(ModeFit)
	if err != nil || report.Reconverted != 2 || applied.Shown != 1 {
		t.Fatalf("mode = %+v %+v %v", report, applied, err)
	}
	after, _ := os.ReadFile(r.service.Paths.Image(2))
	if bytes.Equal(before, after) {
		t.Error("fit must change the stored picture")
	}
	status, _ := r.service.Status()
	if status.Config.Mode != ModeFit || status.Pictures != 2 || status.Shown != 1 {
		t.Errorf("status = %+v", status)
	}
}

func TestApplyWithoutPicturesFails(t *testing.T) {
	r := newRig(t)
	if _, err := r.service.Apply(); err == nil {
		t.Fatal("expected an error")
	} else if _, isMessage := err.(*i18n.MessageError); !isMessage {
		t.Errorf("want a localized error, got %T", err)
	}
}

func TestDefaultSourcesAreTheStorageAllowlist(t *testing.T) {
	got := DefaultSources()
	if len(got) != 1 || got[0] != "/mnt/onboard/Pictures" {
		t.Errorf("sources = %v", got)
	}
}
