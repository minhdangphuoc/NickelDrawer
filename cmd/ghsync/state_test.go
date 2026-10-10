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
		"--pair-info-file", env.pair("pair.json"), "--pair-result-file", env.pair("pair-result"),
		"--progress-file", env.pair("progress.json"), "--rescan-flag", env.pair("rescan.flag")}, &out)
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
	if variables.Configured || variables.CanSync || variables.Pairing || variables.Repo != "—" {
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
	if variables.Summary != "3 new · 120 already here · 1 errors" || variables.LastSync == "" || variables.Status != "Ready" {
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
	if !strings.HasPrefix(variables.PairExpires, "Open 5 more min, until ") {
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

func TestStateShowsTheFailureReasonButNeverTheToken(t *testing.T) {
	env := newStateEnv(t)
	env.configure()
	env.write("data/state.json", `{"last_run":"2026-10-08T21:14:00Z","last_error":"boom `+secretToken+`","files":{}}`, 0o600)
	raw, variables := env.state()
	if strings.Contains(raw, secretToken) || variables.LastError != "boom ***" || variables.Summary != "The last sync failed" {
		t.Fatalf("state = %s", raw)
	}
}

func TestStateCutsALongFailureReasonToFitTheScreen(t *testing.T) {
	env := newStateEnv(t)
	env.configure()
	long := strings.Repeat("Thư mục không có. ", 30)
	env.write("data/state.json", `{"last_run":"2026-10-08T21:14:00Z","last_error":"`+long+`","files":{}}`, 0o600)
	_, variables := env.state()
	if got := len([]rune(variables.LastError)); got != maxErrorChars || !strings.HasSuffix(variables.LastError, "…") {
		t.Fatalf("reason has %d characters: %q", got, variables.LastError)
	}
}

func TestStateShowsDownloadProgressOnlyWhileSyncing(t *testing.T) {
	env := newStateEnv(t)
	env.configure()
	env.write(env.pair("progress.json"), `{"done":2,"total":7,"file":"Dế Mèn.epub"}`, 0o644)
	if _, idle := env.state(); idle.Progress != "" || idle.StepDetail != "" {
		t.Fatalf("progress shown while idle: %+v", idle)
	}
	env.write(env.pair("sync.pid"), strconv.Itoa(os.Getpid()), 0o644)
	_, running := env.state()
	if running.Progress != "Downloading" || running.StepDone != "2" || running.StepTotal != "7" || running.StepDetail != "Dế Mèn.epub" {
		t.Fatalf("progress = %+v", running)
	}
	os.Remove(env.pair("progress.json"))
	if _, starting := env.state(); starting.Progress != "Contacting GitHub…" {
		t.Fatalf("before the first file the screen says it is contacting GitHub: %+v", starting)
	}
}

func TestStateAsksForARescanOnceAfterABookWasAdded(t *testing.T) {
	env := newStateEnv(t)
	env.configure()
	env.write(env.pair("rescan.flag"), "", 0o644)
	env.write(env.pair("sync.pid"), strconv.Itoa(os.Getpid()), 0o644)
	if _, whileRunning := env.state(); len(whileRunning.Requests) != 0 {
		t.Fatalf("no rescan while the sync still runs: %+v", whileRunning.Requests)
	}
	os.Remove(env.pair("sync.pid"))
	raw, first := env.state()
	if len(first.Requests) != 1 || first.Requests[0].Request != "rescan" || first.Requests[0].Mode != "quick" {
		t.Fatalf("want one quick rescan request: %s", raw)
	}
	if _, second := env.state(); len(second.Requests) != 0 {
		t.Fatalf("the host must be asked once, not on every refresh: %+v", second.Requests)
	}
}

func TestStateTellsHowTheLastPairingEnded(t *testing.T) {
	env := newStateEnv(t)
	env.configure()
	env.write(env.pair("pair-result"), "paired", 0o644)
	if _, variables := env.state(); variables.PairNote != "Paired with me/books. Tap 'Sync now'." {
		t.Fatalf("note = %q", variables.PairNote)
	}
	env.write(env.pair("pair-result"), "expired", 0o644)
	if _, variables := env.state(); variables.PairNote != "Pairing timed out. Start it again." {
		t.Fatalf("note = %q", variables.PairNote)
	}
	env.write(env.pair("pair-result"), "garbage", 0o644)
	if _, variables := env.state(); variables.PairNote != "" {
		t.Fatalf("an unknown result must say nothing: %q", variables.PairNote)
	}
}

func TestStateGuidesAnUnpairedUserToPairing(t *testing.T) {
	_, variables := newStateEnv(t).state()
	if !strings.Contains(variables.Guide, "Pair with your phone") || variables.CanSync {
		t.Fatalf("state = %+v", variables)
	}
	env := newStateEnv(t)
	env.configure()
	if _, paired := env.state(); paired.Guide != "" {
		t.Fatalf("the guide must disappear once paired: %q", paired.Guide)
	}
}
