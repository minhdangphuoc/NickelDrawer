package wallpaper

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSleepScreenReadyReadsBothNickelSettings(t *testing.T) {
	both := []string{"ShowBookCover", "FullscreenCoverStretch"}
	cases := []struct {
		name        string
		conf        *string // nil: no file
		wantReady   bool
		wantMissing []string
	}{
		{"both true", ptr("[PowerOptions]\nShowBookCover=true\nFullscreenCoverStretch=true\n"), true, nil},
		{"cover stretch false", ptr("[PowerOptions]\nShowBookCover=true\nFullscreenCoverStretch=false\n"), false, both[1:]},
		{"show cover false", ptr("[PowerOptions]\nShowBookCover=false\nFullscreenCoverStretch=true\n"), false, both[:1]},
		{"keys missing", ptr("[PowerOptions]\nOther=1\n"), false, both},
		{"keys in another section", ptr("[Other]\nShowBookCover=true\nFullscreenCoverStretch=true\n"), false, both},
		{"file missing", nil, false, both},
	}
	for _, c := range cases {
		path := filepath.Join(t.TempDir(), "Kobo eReader.conf")
		if c.conf != nil {
			if err := os.WriteFile(path, []byte(*c.conf), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		ready, missing := SleepScreenReady(path)
		if ready != c.wantReady || !reflect.DeepEqual(missing, c.wantMissing) {
			t.Errorf("%s: got ready=%v missing=%v, want ready=%v missing=%v", c.name, ready, missing, c.wantReady, c.wantMissing)
		}
	}
}

func TestSleepScreenReadyTreatsUnreadableFileAsUnknown(t *testing.T) {
	if ready, missing := SleepScreenReady(t.TempDir()); ready || len(missing) != 2 {
		t.Fatalf("a directory instead of the conf gave ready=%v missing=%v", ready, missing)
	}
}

func ptr(text string) *string { return &text }
