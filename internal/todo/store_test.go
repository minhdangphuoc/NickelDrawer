package todo

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/text/unicode/norm"
)

func newStore(t *testing.T) Store {
	t.Helper()
	return Store{Dir: filepath.Join(t.TempDir(), "data")}
}

func mustLoad(t *testing.T, store Store) List {
	t.Helper()
	list, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	return list
}

func TestMissingFileIsAnEmptyList(t *testing.T) {
	list := mustLoad(t, newStore(t))
	if len(list.Tasks) != 0 || list.Recovered {
		t.Fatalf("list = %+v", list)
	}
}

func TestSavedListSurvivesANewStore(t *testing.T) {
	store := newStore(t)
	tasks, err := Add(nil, "Mua sữa")
	if err != nil {
		t.Fatal(err)
	}
	if tasks, err = Add(tasks, "Gọi cho Mai"); err != nil {
		t.Fatal(err)
	}
	if tasks, err = Toggle(tasks, "Mua sữa"); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(tasks); err != nil {
		t.Fatal(err)
	}
	reloaded := mustLoad(t, Store{Dir: store.Dir})
	if len(reloaded.Tasks) != 2 || !reloaded.Tasks[0].Done || reloaded.Tasks[1].Done || reloaded.Tasks[1].Text != "Gọi cho Mai" {
		t.Fatalf("reloaded = %+v", reloaded.Tasks)
	}
}

func TestSaveLeavesNoTempFileBehind(t *testing.T) {
	store := newStore(t)
	if err := store.Save([]Task{{Text: "a"}}); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(store.Dir)
	if err != nil || len(entries) != 1 || entries[0].Name() != fileName {
		t.Fatalf("entries = %v, err = %v", entries, err)
	}
}

func TestDamagedFileIsMovedAsideAndTheListStartsEmpty(t *testing.T) {
	damaged := map[string]string{
		"not json":       "{ this is not json",
		"wrong version":  `{"version":9,"tasks":[]}`,
		"empty text":     `{"version":1,"tasks":[{"text":"","done":false}]}`,
		"repeated text":  `{"version":1,"tasks":[{"text":"a"},{"text":"a"}]}`,
		"too many tasks": `{"version":1,"tasks":[` + strings.Repeat(`{"text":"x"},`, MaxTasks) + `{"text":"last"}]}`,
	}
	for name, content := range damaged {
		t.Run(name, func(t *testing.T) {
			store := newStore(t)
			if err := os.MkdirAll(store.Dir, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(store.path(), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
			list := mustLoad(t, store)
			if !list.Recovered || len(list.Tasks) != 0 || !store.HasRecoveredMarker() {
				t.Fatalf("list = %+v, marker = %v", list, store.HasRecoveredMarker())
			}
			kept, err := os.ReadFile(filepath.Join(store.Dir, corruptFileName))
			if err != nil || string(kept) != content {
				t.Fatalf("the damaged file must be kept unchanged: %q, %v", kept, err)
			}
			if again := mustLoad(t, store); again.Recovered {
				t.Fatal("the second load must see a clean, empty list")
			}
			if err := store.Save([]Task{{Text: "fresh"}}); err != nil || store.HasRecoveredMarker() {
				t.Fatalf("a save clears the marker: %v", err)
			}
		})
	}
}

func TestCleanTextNormalisesVietnameseToNFC(t *testing.T) {
	decomposed := norm.NFD.String("Đọc sách Việt Nam")
	if decomposed == "Đọc sách Việt Nam" {
		t.Fatal("test input must be decomposed")
	}
	if got := CleanText("  " + decomposed + "\t\n x "); got != "Đọc sách Việt Nam x" {
		t.Fatalf("got %q", got)
	}
}

func TestCleanTextCutsLongInputOnARuneBoundary(t *testing.T) {
	got := CleanText(strings.Repeat("ệ", MaxTextRunes+30))
	if len([]rune(got)) != MaxTextRunes {
		t.Fatalf("got %d runes", len([]rune(got)))
	}
}

func TestAddRules(t *testing.T) {
	tasks, err := Add(nil, "Nấu cơm")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Add(tasks, "   "); !errors.Is(err, ErrEmptyText) {
		t.Fatalf("empty: %v", err)
	}
	if _, err := Add(tasks, norm.NFD.String("Nấu cơm")); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("a decomposed copy is a duplicate: %v", err)
	}
	full := make([]Task, MaxTasks)
	for index := range full {
		full[index] = Task{Text: strings.Repeat("a", index+1)}
	}
	if _, err := Add(full, "one more"); !errors.Is(err, ErrFull) {
		t.Fatalf("full: %v", err)
	}
}

func TestToggleRemoveAndClearDone(t *testing.T) {
	tasks := []Task{{Text: "a"}, {Text: "b", Done: true}, {Text: "c", Done: true}}
	toggled, err := Toggle(tasks, "a")
	if err != nil || !toggled[0].Done || tasks[0].Done {
		t.Fatalf("toggle must change a copy: %+v %v", toggled, err)
	}
	if _, err := Toggle(tasks, "zzz"); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	removed, err := Remove(tasks, "b")
	if err != nil || len(removed) != 2 || removed[1].Text != "c" {
		t.Fatalf("removed = %+v %v", removed, err)
	}
	kept, count := ClearDone(tasks)
	if count != 2 || len(kept) != 1 || kept[0].Text != "a" {
		t.Fatalf("kept = %+v, count = %d", kept, count)
	}
	if open, done := Counts(tasks); open != 1 || done != 2 {
		t.Fatalf("counts = %d, %d", open, done)
	}
}

func TestStateJSONHasRowsCountsAndNotice(t *testing.T) {
	encoded, err := StateJSON([]Task{{Text: `say "hi" \ there`}, {Text: "xong", Done: true}}, "")
	if err != nil {
		t.Fatal(err)
	}
	want := `{"open":"1","done":"1","hasdone":true,"newitem":"","notice":"","items":[["say \"hi\" \\ there",""],["xong","✓"]]}`
	if string(encoded) != want {
		t.Fatalf("got %s\nwant %s", encoded, want)
	}
}
