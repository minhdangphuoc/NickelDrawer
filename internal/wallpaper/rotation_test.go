package wallpaper

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

func entriesWithIDs(ids ...int) []Entry {
	entries := make([]Entry, 0, len(ids))
	for _, id := range ids {
		entries = append(entries, Entry{ID: id})
	}
	return entries
}

func seededIndex(seed int64) func(int) int {
	source := rand.New(rand.NewSource(seed))
	return source.Intn
}

func TestPickRandomNeverRepeatsCurrentOrPrevious(t *testing.T) {
	entries := entriesWithIDs(1, 2, 3, 4, 5)
	randomIndex := seededIndex(1)
	current, history := 1, []int(nil)
	for step := 0; step < 1000; step++ {
		history = rememberShown(history, current)
		picked := pickRandom(entries, current, history, randomIndex)
		for _, recent := range history {
			if picked == recent {
				t.Fatalf("step %d: picked %d, recent %v", step, picked, history)
			}
		}
		current = picked
	}
}

func TestPickRandomDistributionIsRoughlyUniform(t *testing.T) {
	entries := entriesWithIDs(1, 2, 3, 4, 5, 6)
	counts := map[int]int{}
	const draws = 60000
	for draw := 0; draw < draws; draw++ {
		counts[pickRandom(entries, 1, []int{1, 2}, cryptoIndex)]++
	}
	if counts[1] != 0 || counts[2] != 0 {
		t.Fatalf("avoided pictures were picked: %v", counts)
	}
	expected := draws / 4
	for _, id := range []int{3, 4, 5, 6} {
		if counts[id] < expected*9/10 || counts[id] > expected*11/10 {
			t.Errorf("picture %d picked %d times, want about %d (%v)", id, counts[id], expected, counts)
		}
	}
}

func TestPickRandomWithTwoPicturesAlternates(t *testing.T) {
	entries := entriesWithIDs(4, 9)
	current, history := 4, []int(nil)
	for step := 0; step < 10; step++ {
		history = rememberShown(history, current)
		picked := pickRandom(entries, current, history, cryptoIndex)
		if picked == current {
			t.Fatalf("step %d: repeated %d", step, picked)
		}
		current = picked
	}
}

func TestPickRandomWithOnePictureKeepsIt(t *testing.T) {
	if got := pickRandom(entriesWithIDs(7), 7, []int{7}, cryptoIndex); got != 7 {
		t.Errorf("picked %d, want 7", got)
	}
	if got := pickRandom(nil, 7, nil, cryptoIndex); got != 0 {
		t.Errorf("empty library picked %d", got)
	}
}

func TestRememberShownKeepsTheLastTwoMostRecentFirst(t *testing.T) {
	history := rememberShown(rememberShown(rememberShown(nil, 1), 2), 3)
	if fmt.Sprint(history) != "[3 2]" {
		t.Errorf("history = %v", history)
	}
	if got := rememberShown([]int{3, 2}, 3); fmt.Sprint(got) != "[3 2]" {
		t.Errorf("re-showing the same picture = %v", got)
	}
}

func importPictures(t *testing.T, r rig, count int) {
	t.Helper()
	for number := 0; number < count; number++ {
		put(t, filepath.Join(r.pictures, fmt.Sprintf("p%d.jpg", number)), solidJPEG(t, 30, 30, uint8(20*number)))
	}
	if _, _, err := r.service.ImportAndApply(); err != nil {
		t.Fatal(err)
	}
}

func TestRandomNextPublishesOnePictureAndTracksCurrent(t *testing.T) {
	r := newRig(t)
	importPictures(t, r, 4)
	r.service.RandomIndex = seededIndex(7)
	if _, err := r.service.SetRotation(RotationRandom); err != nil {
		t.Fatal(err)
	}
	previous, current := 0, 4 // the import made the newest picture current
	for step := 0; step < 30; step++ {
		entry, err := r.service.Next()
		if err != nil {
			t.Fatal(err)
		}
		if entry.ID == current || entry.ID == previous {
			t.Fatalf("step %d: picked %d after %d, %d", step, entry.ID, previous, current)
		}
		previous, current = current, entry.ID
		config, _ := LoadConfig(r.service.Paths.Config())
		if config.Current != entry.ID {
			t.Fatalf("config current = %d, want %d", config.Current, entry.ID)
		}
		got := listing(t, r.screensaver)
		if len(got) != 1 || got[0] != ownedFileName(entry.ID) {
			t.Fatalf("screensaver = %v, want only %s", got, ownedFileName(entry.ID))
		}
	}
}

func TestWakeFlagFollowsRotation(t *testing.T) {
	r := newRig(t)
	importPictures(t, r, 2)
	flag := r.service.Paths.WakeFlag()
	if _, err := os.Stat(flag); err == nil {
		t.Fatal("flag exists in fixed rotation")
	}
	if _, err := r.service.SetRotation(RotationRandom); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(flag); err != nil {
		t.Fatalf("random rotation without flag: %v", err)
	}
	if _, err := r.service.SetRotation(RotationFixed); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(flag); err == nil {
		t.Fatal("flag kept after switching back to fixed")
	}
}

func TestStateCreatesWakeFlagForOlderRandomConfig(t *testing.T) {
	r := newRig(t)
	importPictures(t, r, 2)
	if err := SaveConfig(r.service.Paths.Config(), Config{Mode: ModeFill, Rotation: RotationRandom, Current: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.service.ScreenState(ScreenStateFiles{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(r.service.Paths.WakeFlag()); err != nil {
		t.Fatalf("state did not create the flag: %v", err)
	}
}

func TestFixedNextStillCyclesInOrder(t *testing.T) {
	r := newRig(t)
	importPictures(t, r, 3)
	for _, want := range []int{1, 2, 3, 1} {
		entry, err := r.service.Next()
		if err != nil || entry.ID != want {
			t.Fatalf("Next = %v, %v, want %d", entry.ID, err, want)
		}
	}
	if _, err := os.Stat(r.service.Paths.RotationHistory()); err == nil {
		t.Error("fixed rotation must not write the random history")
	}
}
