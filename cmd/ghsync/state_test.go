package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

const secretToken = "github_pat_" + "SECRETSECRETSECRETSECRET"

type stateEnv struct {
	t      *testing.T
	appDir string
	tmp    string
}

func newStateEnv(t *testing.T) stateEnv {
	t.Helper()
	appDir := t.TempDir()
	if err := os.Symlink(mustAbs(t, "../../apps/ghsync/locales"), filepath.Join(appDir, "locales")); err != nil {
		t.Fatal(err)
	}
	return stateEnv{t: t, appDir: appDir, tmp: t.TempDir()}
}

func mustAbs(t *testing.T, path string) string {
	t.Helper()
	absolute, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	return absolute
}

func (env stateEnv) write(relative, content string, mode os.FileMode) string {
	env.t.Helper()
	path := filepath.Join(env.appDir, relative)
	if filepath.IsAbs(relative) {
		path = relative
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		env.t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		env.t.Fatal(err)
	}
	return path
}

func (env stateEnv) pair(relative string) string { return filepath.Join(env.tmp, relative) }

// state runs `ghsync state` and returns the raw output and the decoded variables.
func (env stateEnv) state() (string, screenVariables) {
	env.t.Helper()
	var out bytes.Buffer
	status := execute([]string{"state", "--app-dir", env.appDir, "--locale", "en",
		"--sync-pid-file", env.pair("sync.pid"), "--pair-pid-file", env.pair("pair.pid"),
		"--pair-info-file", env.pair("pair.json")}, &out)
	if status != 0 {
		env.t.Fatalf("state exit %d: %s", status, out.String())
	}
	if strings.Count(strings.TrimSpace(out.String()), "\n") != 0 {
		env.t.Fatalf("state must be one line: %q", out.String())
	}
	var variables screenVariables
	if err := json.Unmarshal(out.Bytes(), &variables); err != nil {
		env.t.Fatalf("not JSON: %v: %s", err, out.String())
	}
	return out.String(), variables
}

func (env stateEnv) configure() {
	env.write("data/config.json", `{"repo":"me/books","path":"epub"}`, 0o600)
	env.write("data/token", secretToken+"\n", 0o600)
}

func TestStateBeforePairingIsEmptyButUsable(t *testing.T) {
	_, variables := newStateEnv(t).state()
	if variables.Configured || variables.CanSync || variables.Pairing || variables.Repo != "" {
		t.Fatalf("state = %+v", variables)
	}
	if variables.Status != "Not paired" || variables.Summary != "Never synced" || variables.PairHint == "" {
		t.Fatalf("texts = %+v", variables)
	}
}

func TestStateConfiguredShowsRepoAndLastResultButNeverTheToken(t *testing.T) {
	env := newStateEnv(t)
	env.configure()
	env.write("data/state.json", `{"last_run":"2026-10-08T21:14:00Z","last_new":3,"last_existing":120,"last_errors":1,"files":{}}`, 0o600)
	raw, variables := env.state()
	if strings.Contains(raw, secretToken) || strings.Contains(raw, "SECRET") || strings.Contains(raw, "token") {
		t.Fatalf("the token leaked into the state: %s", raw)
	}
	if !variables.Configured || !variables.CanSync || variables.Repo != "me/books" || variables.Folder != "epub" || variables.Branch != "Default" {
		t.Fatalf("state = %+v", variables)
	}
	if variables.Summary != "3 new, 120 already here, 1 errors" || variables.LastSync == "" || variables.Status != "Ready" {
		t.Fatalf("last sync = %+v", variables)
	}
}

func TestStateWithoutTokenFileIsNotConfigured(t *testing.T) {
	env := newStateEnv(t)
	env.write("data/config.json", `{"repo":"me/books"}`, 0o600)
	if _, variables := env.state(); variables.Configured {
		t.Fatalf("a config without a token must not count as paired: %+v", variables)
	}
}

func TestStateRunningSyncBlocksSyncButton(t *testing.T) {
	env := newStateEnv(t)
	env.configure()
	env.write(env.pair("sync.pid"), strconv.Itoa(os.Getpid()), 0o644)
	_, variables := env.state()
	if !variables.Running || variables.CanSync || variables.Status != "Syncing…" {
		t.Fatalf("state = %+v", variables)
	}
}

func TestStateOpenPairingShowsAddressAndCode(t *testing.T) {
	env := newStateEnv(t)
	env.write(env.pair("pair.pid"), strconv.Itoa(os.Getpid()), 0o644)
	expires := time.Now().Add(5 * time.Minute).UTC().Format(time.RFC3339)
	env.write(env.pair("pair.json"), `{"url":"http://192.168.1.23:8765","code":"482913","expires":"`+expires+`"}`, 0o600)
	_, variables := env.state()
	if !variables.Pairing || variables.PairURL != "http://192.168.1.23:8765" || variables.PairCode != "482913" {
		t.Fatalf("state = %+v", variables)
	}
	if !strings.HasPrefix(variables.PairExpires, "Valid until ") {
		t.Fatalf("expires = %q", variables.PairExpires)
	}
}

func TestStateHidesStaleOrExpiredPairing(t *testing.T) {
	env := newStateEnv(t)
	expired := time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)
	env.write(env.pair("pair.json"), `{"url":"http://x:1","code":"111111","expires":"`+expired+`"}`, 0o600)
	env.write(env.pair("pair.pid"), strconv.Itoa(os.Getpid()), 0o644)
	if raw, variables := env.state(); variables.Pairing || strings.Contains(raw, "111111") {
		t.Fatalf("expired pairing must be hidden: %s", raw)
	}
	env.write(env.pair("pair.pid"), "2147483646", 0o644)
	future := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	env.write(env.pair("pair.json"), `{"url":"http://x:1","code":"222222","expires":"`+future+`"}`, 0o600)
	if raw, variables := env.state(); variables.Pairing || strings.Contains(raw, "222222") {
		t.Fatalf("pairing without a live process must be hidden: %s", raw)
	}
}

func TestStateReportsFailedLastSyncWithoutTheErrorText(t *testing.T) {
	env := newStateEnv(t)
	env.configure()
	env.write("data/state.json", `{"last_run":"2026-10-08T21:14:00Z","last_error":"boom `+secretToken+`","files":{}}`, 0o600)
	raw, variables := env.state()
	if strings.Contains(raw, "boom") || strings.Contains(raw, secretToken) || variables.Summary != "Last sync failed. See data/log.txt." {
		t.Fatalf("state = %s", raw)
	}
}
