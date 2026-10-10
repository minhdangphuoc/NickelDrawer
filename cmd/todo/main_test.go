package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runTodo(t *testing.T, dataDir string, args ...string) (output string, status int) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	status = run(append(args, "--data-dir", dataDir), &stdout, &stderr)
	return strings.TrimSpace(stdout.String()), status
}

func TestEditsPrintOutcomeWords(t *testing.T) {
	dir := t.TempDir()
	steps := []struct {
		args []string
		want string
	}{
		{[]string{"add", "--text", "Mua sữa"}, "added"},
		{[]string{"add", "--text", "Mua sữa"}, "duplicate"},
		{[]string{"add", "--text", "  "}, "empty"},
		{[]string{"toggle", "--text", "Mua sữa"}, "toggled"},
		{[]string{"toggle", "--text", "không có"}, "missing"},
		{[]string{"clear-done"}, "cleared 1"},
		{[]string{"clear-done"}, "nothing_done"},
		{[]string{"add", "--text", "Gọi Mai"}, "added"},
		{[]string{"remove", "--text", "Gọi Mai"}, "removed"},
	}
	for _, step := range steps {
		if got, status := runTodo(t, dir, step.args...); got != step.want || status != 0 {
			t.Fatalf("%v: got %q (status %d), want %q", step.args, got, status, step.want)
		}
	}
}

func TestDamagedListReportsRecoveredOnceAndStateShowsTheNoticeUntilTheNextChange(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "todo.json"), []byte("garbage"), 0o644); err != nil {
		t.Fatal(err)
	}
	stateOutput, status := runTodo(t, dir, "state", "--notice", "List was damaged")
	var state struct{ Notice string }
	if err := json.Unmarshal([]byte(stateOutput), &state); err != nil || status != 0 || state.Notice != "List was damaged" {
		t.Fatalf("state = %q (%v, status %d)", stateOutput, err, status)
	}
	if got, _ := runTodo(t, dir, "add", "--text", "again"); got != "added" {
		t.Fatalf("the damage was already reported by the screen state, got %q", got)
	}
	stateOutput, _ = runTodo(t, dir, "state", "--notice", "List was damaged")
	if strings.Contains(stateOutput, "damaged") {
		t.Fatalf("the notice must go after a change: %s", stateOutput)
	}
}

func TestActionThatFindsTheDamageReportsRecovered(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "todo.json"), []byte("garbage"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, _ := runTodo(t, dir, "add", "--text", "again"); got != "recovered" {
		t.Fatalf("got %q", got)
	}
	if got, _ := runTodo(t, dir, "toggle", "--text", "again"); got != "toggled" {
		t.Fatalf("the edit must still have been applied: %q", got)
	}
}

func TestUnknownCommandIsAUsageError(t *testing.T) {
	if _, status := runTodo(t, t.TempDir(), "frobnicate"); status != exitUsage {
		t.Fatalf("status = %d", status)
	}
}
