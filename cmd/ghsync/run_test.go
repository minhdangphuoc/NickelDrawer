package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"koboapps/internal/storage"
)

// runEnv is a fake Kobo tree with the shipped routing rules, so `ghsync run` loads its allowlist
// from the place it does on the device.
func newRunEnv(t *testing.T) stateEnv {
	t.Helper()
	root := t.TempDir()
	t.Cleanup(storage.SetRoot(root))
	rules, err := os.ReadFile("../../host/config/routing.json")
	if err != nil {
		t.Fatal(err)
	}
	rulesPath := filepath.Join(root, ".adds", "koboapps", "config", "routing.json")
	if err := os.MkdirAll(filepath.Dir(rulesPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rulesPath, rules, 0o644); err != nil {
		t.Fatal(err)
	}
	env := newStateEnv(t)
	env.configure()
	return env
}

func TestRunPrintsAClearErrorAndNeverTheToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad credentials "+r.Header.Get("Authorization"), http.StatusUnauthorized)
	}))
	defer server.Close()
	env := newRunEnv(t)
	var out bytes.Buffer
	status := execute([]string{"run", "--app-dir", env.appDir, "--locale", "en", "--api-base", server.URL,
		"--progress-file", env.pair("progress.json"), "--rescan-flag", env.pair("rescan.flag")}, &out)
	logText, _ := os.ReadFile(filepath.Join(env.appDir, "data", "log.txt"))
	stateText, _ := os.ReadFile(filepath.Join(env.appDir, "data", "state.json"))
	for name, text := range map[string]string{"stdout": out.String(), "log": string(logText), "state": string(stateText)} {
		if strings.Contains(text, "SECRETSECRET") {
			t.Errorf("%s holds the token: %s", name, text)
		}
	}
	if status != 1 || !strings.Contains(out.String(), "bad or expired token") {
		t.Fatalf("status %d, output %q", status, out.String())
	}
}

func TestRunOnAnEmptyRepositoryEndsCleanlyWithZeroBooks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/git/trees/") {
			http.Error(w, `{"message":"Git Repository is empty."}`, http.StatusConflict)
			return
		}
		w.Write([]byte(`{"default_branch":"main"}`))
	}))
	defer server.Close()
	env := newRunEnv(t)
	var out bytes.Buffer
	status := execute([]string{"run", "--app-dir", env.appDir, "--locale", "en", "--api-base", server.URL,
		"--progress-file", env.pair("progress.json"), "--rescan-flag", env.pair("rescan.flag")}, &out)
	if status != 0 || !strings.Contains(out.String(), "Synced: 0 new, 0 already here") {
		t.Fatalf("status %d, output %q", status, out.String())
	}
	if _, err := os.Stat(env.pair("rescan.flag")); err == nil {
		t.Error("nothing was added, so no rescan must be requested")
	}
}
