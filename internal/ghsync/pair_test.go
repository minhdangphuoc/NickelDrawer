package ghsync

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"koboapps/internal/i18n"
)

const (
	pairCode   = "123456"
	pairToken  = "ghp_" + "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	otherToken = "github_pat_" + "ZZZZZZZZZZZZZZZZZZZZZZZZZZ"
)

type pairEnv struct {
	session *pairSession
	server  *httptest.Server
	paths   Paths
	log     *bytes.Buffer
	clock   *time.Time
	github  *httptest.Server
	gitHits *[]string
}

// fakeRepo is a GitHub repository as the pairing check sees it. status forces one answer for
// every call; otherwise the other fields describe what exists.
type fakeRepo struct {
	status   int
	missing  bool            // the repository is unknown to the token (GitHub answers 404)
	empty    bool            // no commits, so no branches
	branches []string        // branch names; the first is the default
	folders  map[string]bool // folders that exist on every branch
	files    map[string]bool // paths that are single files
}

func healthyRepo() *fakeRepo {
	return &fakeRepo{branches: []string{"main", "dev"}, folders: map[string]bool{"books": true}}
}

func (repo *fakeRepo) handle(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/repos/me/books")
	switch {
	case repo.status != 0 && repo.status != http.StatusOK:
		http.Error(w, "nope", repo.status)
	case repo.missing:
		http.Error(w, `{"message":"Not Found"}`, http.StatusNotFound)
	case path == "":
		io.WriteString(w, `{"default_branch":"main"}`)
	case path == "/branches":
		repo.listBranches(w)
	case strings.HasPrefix(path, "/branches/"):
		repo.showBranch(w, strings.TrimPrefix(path, "/branches/"))
	case strings.HasPrefix(path, "/contents"):
		repo.listFolder(w, strings.TrimPrefix(strings.TrimPrefix(path, "/contents"), "/"))
	default:
		http.NotFound(w, r)
	}
}

func (repo *fakeRepo) listBranches(w http.ResponseWriter) {
	if repo.empty {
		io.WriteString(w, `[]`)
		return
	}
	io.WriteString(w, `[{"name":"`+repo.branches[0]+`"}]`)
}

func (repo *fakeRepo) showBranch(w http.ResponseWriter, name string) {
	for _, branch := range repo.branches {
		if branch == name {
			io.WriteString(w, `{"name":"`+name+`"}`)
			return
		}
	}
	http.Error(w, `{"message":"Branch not found"}`, http.StatusNotFound)
}

func (repo *fakeRepo) listFolder(w http.ResponseWriter, folder string) {
	switch {
	case repo.empty:
		http.Error(w, `{"message":"This repository is empty."}`, http.StatusNotFound)
	case folder == "" || repo.folders[folder]:
		io.WriteString(w, `[]`)
	case repo.files[folder]:
		io.WriteString(w, `{"type":"file"}`)
	default:
		http.Error(w, `{"message":"Not Found"}`, http.StatusNotFound)
	}
}

// newPairEnv wires a session to a fake GitHub API; githubStatus is the status for every API call.
func newPairEnv(t *testing.T, githubStatus int) pairEnv {
	t.Helper()
	repo := healthyRepo()
	repo.status = githubStatus
	return newPairEnvForRepo(t, repo)
}

func newPairEnvForRepo(t *testing.T, repo *fakeRepo) pairEnv {
	t.Helper()
	var hits []string
	github := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits = append(hits, r.URL.RequestURI())
		repo.handle(w, r)
	}))
	t.Cleanup(github.Close)

	appDir := filepath.Join(t.TempDir(), "app")
	if err := os.MkdirAll(filepath.Join(appDir, "data"), 0o755); err != nil {
		t.Fatal(err)
	}
	messages, err := i18n.LoadBundle("../../apps/ghsync/locales", "en")
	if err != nil {
		t.Fatal(err)
	}
	logBuffer := &bytes.Buffer{}
	clock := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	session, err := newPairSession(PairOptions{
		Paths: Paths{AppDir: appDir}, APIBase: github.URL, Messages: messages, Locale: "en",
		Log: NewLogger(logBuffer, ""), Backoff: time.Millisecond, Code: pairCode,
		Now: func() time.Time { return clock },
	})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(session)
	t.Cleanup(server.Close)
	return pairEnv{session, server, Paths{AppDir: appDir}, logBuffer, &clock, github, &hits}
}

func (e pairEnv) post(t *testing.T, fields map[string]string) (int, string) {
	t.Helper()
	values := url.Values{}
	for name, value := range fields {
		values.Set(name, value)
	}
	response, err := http.PostForm(e.server.URL+"/", values)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	return response.StatusCode, string(body)
}

func goodForm() map[string]string {
	return map[string]string{"code": pairCode, "repo": "me/books", "branch": "", "folder": "books", "token": pairToken}
}

func (e pairEnv) assertNoSecrets(t *testing.T, texts ...string) {
	t.Helper()
	for _, text := range append(texts, e.log.String()) {
		if strings.Contains(text, pairToken) || strings.Contains(text, otherToken) {
			t.Fatalf("token leaked into: %q", text)
		}
	}
}

func TestPairFormIsLocalizedAndSelfContained(t *testing.T) {
	e := newPairEnv(t, 200)
	response, err := http.Get(e.server.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	page := string(body)
	for _, want := range []string{`type="password"`, `name="code"`, `name="repo"`, `name="branch"`, `name="folder"`, "Use only on your own Wi-Fi", `<html lang="en">`} {
		if !strings.Contains(page, want) {
			t.Errorf("form misses %q", want)
		}
	}
	if strings.Contains(page, "http://") || strings.Contains(page, "https://") || strings.Contains(page, "<script") {
		t.Error("form must not load external resources or scripts")
	}
	if response.Header.Get("Cache-Control") != "no-store" {
		t.Error("form must not be cached")
	}
}

func TestPairFormFollowsVietnameseLocale(t *testing.T) {
	e := newPairEnv(t, 200)
	messages, _ := i18n.LoadBundle("../../apps/ghsync/locales", "vi")
	e.session.options.Messages = messages
	e.session.options.Locale = "vi"
	response, _ := http.Get(e.server.URL + "/")
	body, _ := io.ReadAll(response.Body)
	if !strings.Contains(string(body), "Chỉ dùng trên Wi-Fi nhà bạn") || !strings.Contains(string(body), `lang="vi"`) {
		t.Errorf("not Vietnamese: %s", body)
	}
}

func TestPairWrongCodeThenLockout(t *testing.T) {
	e := newPairEnv(t, 200)
	form := goodForm()
	form["code"] = "000000"
	for attempt := 1; attempt < maxCodeAttempts; attempt++ {
		status, body := e.post(t, form)
		if status != http.StatusForbidden || !strings.Contains(body, "Wrong code") {
			t.Fatalf("attempt %d: %d %s", attempt, status, body)
		}
	}
	if status, _ := e.post(t, form); status != http.StatusTooManyRequests {
		t.Fatalf("fifth wrong code must lock, got %d", status)
	}
	if outcome := <-e.session.outcome; outcome != PairLocked {
		t.Fatalf("outcome = %s", outcome)
	}
	// Even the right code is refused once locked, and nothing is saved.
	if status, _ := e.post(t, goodForm()); status != http.StatusTooManyRequests {
		t.Fatalf("locked session accepted a request: %d", status)
	}
	if _, err := os.Stat(e.paths.Token()); err == nil {
		t.Fatal("token was saved after lockout")
	}
}

func TestPairSuccessWritesPrivateFilesAndNeverEchoesToken(t *testing.T) {
	e := newPairEnv(t, 200)
	status, body := e.post(t, goodForm())
	if status != http.StatusOK || !strings.Contains(body, "me/books") {
		t.Fatalf("%d %s", status, body)
	}
	e.assertNoSecrets(t, body)

	for _, file := range []string{e.paths.Token(), e.paths.Config()} {
		info, err := os.Stat(file)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Errorf("%s mode = %v, want 600", file, info.Mode().Perm())
		}
	}
	savedToken, _ := os.ReadFile(e.paths.Token())
	if strings.TrimSpace(string(savedToken)) != pairToken {
		t.Error("token file content wrong")
	}
	var saved Config
	raw, _ := os.ReadFile(e.paths.Config())
	if err := json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Repo != "me/books" || saved.Path != "books" || saved.Branch != "" || saved.Dest != "" {
		t.Errorf("config = %+v", saved)
	}
	if _, err := LoadConfig(e.paths.Config()); err != nil {
		t.Errorf("saved config does not load: %v", err)
	}
	if outcome := <-e.session.outcome; outcome != PairPaired {
		t.Errorf("outcome = %s", outcome)
	}
	if status, _ := e.post(t, goodForm()); status != http.StatusGone {
		t.Errorf("a second submit after success must be refused, got %d", status)
	}
	leftovers, _ := filepath.Glob(filepath.Join(filepath.Dir(e.paths.Token()), ".ghsync-pair-*"))
	if len(leftovers) != 0 {
		t.Errorf("temp files left: %v", leftovers)
	}
}

func TestPairKeepsExistingDestAndExtensions(t *testing.T) {
	e := newPairEnv(t, 200)
	existing := `{"repo":"old/one","dest":"/mnt/onboard/Books","extensions":[".epub"]}`
	if err := os.WriteFile(e.paths.Config(), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}
	e.post(t, goodForm())
	config, err := LoadConfig(e.paths.Config())
	if err != nil || config.Dest != "/mnt/onboard/Books" || len(config.Extensions) != 1 || config.Repo != "me/books" {
		t.Errorf("config = %+v, err = %v", config, err)
	}
}

func TestPairRejectsInvalidInputWithoutCallingGitHub(t *testing.T) {
	cases := map[string]map[string]string{
		"repo":          {"repo": "not a repo"},
		"token prefix":  {"token": "abc_ABCDEFGHIJKLMNOPQRSTUVWXYZ"},
		"token short":   {"token": "ghp_short"},
		"token charset": {"token": "ghp_ABCDEFGHIJKLMNOPQRSTUV WXYZ"},
		"branch":        {"branch": "a..b"},
		"folder":        {"folder": "../etc"},
	}
	for name, override := range cases {
		t.Run(name, func(t *testing.T) {
			e := newPairEnv(t, 200)
			form := goodForm()
			for key, value := range override {
				form[key] = value
			}
			status, body := e.post(t, form)
			if status != http.StatusUnprocessableEntity || !strings.Contains(body, `class="error"`) {
				t.Fatalf("%d %s", status, body)
			}
			if len(*e.gitHits) != 0 {
				t.Errorf("GitHub was called: %v", *e.gitHits)
			}
			if _, err := os.Stat(e.paths.Token()); err == nil {
				t.Error("token saved despite invalid input")
			}
			if strings.Contains(body, `value="`+pairToken) {
				t.Error("token echoed")
			}
			// A mistake does not burn a code attempt.
			if e.session.attempts != 0 {
				t.Errorf("attempts = %d", e.session.attempts)
			}
		})
	}
}

func TestPairGitHubFailureShowsErrorAndSavesNothing(t *testing.T) {
	for status, want := range map[int]string{401: "bad or expired token", 404: "not found"} {
		e := newPairEnv(t, status)
		code, body := e.post(t, goodForm())
		if code != http.StatusUnprocessableEntity || !strings.Contains(body, want) {
			t.Errorf("status %d: %d %s", status, code, body)
		}
		e.assertNoSecrets(t, body)
		if _, err := os.Stat(e.paths.Config()); err == nil {
			t.Errorf("status %d: config saved", status)
		}
		// The form comes back filled in, minus the token.
		if !strings.Contains(body, `value="me/books"`) {
			t.Errorf("status %d: repo not kept", status)
		}
	}
}

func TestPairChecksRepoAndContents(t *testing.T) {
	e := newPairEnv(t, 200)
	e.post(t, goodForm())
	got := strings.Join(*e.gitHits, " ")
	if !strings.Contains(got, "/repos/me/books ") && !strings.HasPrefix(got, "/repos/me/books") {
		t.Errorf("repo not checked: %s", got)
	}
	if !strings.Contains(got, "/repos/me/books/contents/books?ref=main") {
		t.Errorf("contents not read from default branch: %s", got)
	}
}

// A pairing check must say which of repo, branch and folder is wrong: GitHub answers 404 for all
// three, and the first version of this check showed one vague message for every case.
func TestPairNamesWhatIsMissingInsteadOfAVague404(t *testing.T) {
	cases := []struct {
		name  string
		repo  *fakeRepo
		form  map[string]string
		want  string
		saved bool
	}{
		{"repository missing or hidden from the token", &fakeRepo{missing: true}, goodForm(), "Repository me/books not found, or the token cannot see it", false},
		{"branch missing", healthyRepo(), withField(goodForm(), "branch", "nope"), "Branch nope does not exist in me/books", false},
		{"folder missing", healthyRepo(), withField(goodForm(), "folder", "novels"), "Folder novels does not exist on branch main", false},
		{"folder missing on a chosen branch", healthyRepo(), withFields(goodForm(), map[string]string{"folder": "novels", "branch": "dev"}), "on branch dev", false},
		{"folder is a file", &fakeRepo{branches: []string{"main"}, files: map[string]bool{"books": true}}, goodForm(), "Folder books does not exist", false},
		{"empty repository", &fakeRepo{empty: true}, goodForm(), "", true},
		{"empty repository with a branch and folder typed", &fakeRepo{empty: true}, withField(goodForm(), "branch", "main"), "", true},
		{"existing branch and folder", healthyRepo(), withField(goodForm(), "branch", "dev"), "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := newPairEnvForRepo(t, c.repo)
			status, body := e.post(t, c.form)
			e.assertNoSecrets(t, body)
			_, configErr := os.Stat(e.paths.Config())
			if c.saved {
				if status != http.StatusOK || configErr != nil {
					t.Fatalf("an accessible repository must pair: %d %v\n%s", status, configErr, body)
				}
				return
			}
			if status != http.StatusUnprocessableEntity || !strings.Contains(body, c.want) || configErr == nil {
				t.Fatalf("want %q and nothing saved: %d %v\n%s", c.want, status, configErr, body)
			}
			if strings.Contains(body, "check the repo, branch and folder") {
				t.Errorf("the vague 404 text came back: %s", body)
			}
		})
	}
}

func withField(form map[string]string, name, value string) map[string]string {
	return withFields(form, map[string]string{name: value})
}

func withFields(form map[string]string, changes map[string]string) map[string]string {
	copied := map[string]string{}
	for name, value := range form {
		copied[name] = value
	}
	for name, value := range changes {
		copied[name] = value
	}
	return copied
}

func TestPairNetworkErrorRedactsToken(t *testing.T) {
	e := newPairEnv(t, 200)
	e.session.options.APIBase = "http://127.0.0.1:1/" + pairToken
	_, body := e.post(t, goodForm())
	e.assertNoSecrets(t, body)
}

func TestPairExpiry(t *testing.T) {
	e := newPairEnv(t, 200)
	*e.clock = e.clock.Add(DefaultPairLifetime + time.Second)
	if status, _ := e.post(t, goodForm()); status != http.StatusGone {
		t.Fatalf("expired session accepted a post: %d", status)
	}
	response, _ := http.Get(e.server.URL + "/")
	if response.StatusCode != http.StatusGone {
		t.Errorf("GET after expiry = %d", response.StatusCode)
	}
	if _, err := os.Stat(e.paths.Token()); err == nil {
		t.Error("token saved after expiry")
	}
	if outcome := <-e.session.outcome; outcome != PairExpired {
		t.Errorf("outcome = %s", outcome)
	}
}

func TestPairOtherPathsAndMethods(t *testing.T) {
	e := newPairEnv(t, 200)
	if response, _ := http.Get(e.server.URL + "/admin"); response.StatusCode != http.StatusNotFound {
		t.Errorf("/admin = %d", response.StatusCode)
	}
	request, _ := http.NewRequest(http.MethodPut, e.server.URL+"/", nil)
	if response, _ := http.DefaultClient.Do(request); response.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("PUT = %d", response.StatusCode)
	}
}

func TestPairServerBindsOnlyTheGivenAddressAndStopsOnContext(t *testing.T) {
	messages, _ := i18n.LoadBundle("../../apps/ghsync/locales", "en")
	out := &syncBuffer{}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan PairOutcome, 1)
	go func() {
		outcome, _ := Pair(ctx, PairOptions{
			Paths: Paths{AppDir: t.TempDir()}, IP: "127.0.0.1", Port: 18765, Messages: messages, Out: out, Code: pairCode,
		})
		done <- outcome
	}()
	waitForText(t, out, "127.0.0.1:18765")
	for _, want := range []string{"http://127.0.0.1:18765", "Code: " + pairCode, "10 minutes"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("announcement misses %q: %s", want, out)
		}
	}
	cancel()
	if outcome := <-done; outcome != PairStopped {
		t.Errorf("outcome = %s", outcome)
	}
}

// syncBuffer lets the test read what the pairing goroutine is still writing.
type syncBuffer struct {
	mutex  sync.Mutex
	buffer bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.buffer.Write(p)
}

func (b *syncBuffer) String() string {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.buffer.String()
}

func waitForText(t *testing.T, buffer *syncBuffer, text string) {
	t.Helper()
	for range 100 {
		if strings.Contains(buffer.String(), text) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("never saw %q", text)
}

func TestPairRefusesWildcardAndBadAddress(t *testing.T) {
	for _, address := range []string{"0.0.0.0", "::", "nonsense", "::1"} {
		if _, err := resolvePairIP(address); !hasMessageKey(err, "msg.err_pair_ip_invalid") {
			t.Errorf("%s accepted: %v", address, err)
		}
	}
}

func TestPairCodeIsSixDigits(t *testing.T) {
	for range 50 {
		code, err := newPairCode()
		if err != nil || len(code) != 6 || strings.Trim(code, "0123456789") != "" {
			t.Fatalf("code %q err %v", code, err)
		}
	}
}

func TestPairPublishesInfoForTheScreenWhileOpenOnly(t *testing.T) {
	messages, _ := i18n.LoadBundle("../../apps/ghsync/locales", "en")
	infoFile := filepath.Join(t.TempDir(), "pair.json")
	out := &syncBuffer{}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		Pair(ctx, PairOptions{
			Paths: Paths{AppDir: t.TempDir()}, IP: "127.0.0.1", Port: 18766, Messages: messages, Out: out,
			Code: pairCode, InfoFile: infoFile,
		})
	}()
	waitForText(t, out, "127.0.0.1:18766")
	info, ok := readPairInfo(infoFile, time.Now())
	if !ok || info.URL != "http://127.0.0.1:18766" || info.Code != pairCode {
		t.Fatalf("info = %+v, ok=%v", info, ok)
	}
	if stat, _ := os.Stat(infoFile); stat.Mode().Perm() != 0o600 {
		t.Errorf("info file mode = %v, want 600", stat.Mode().Perm())
	}
	cancel()
	<-done
	if _, err := os.Stat(infoFile); !os.IsNotExist(err) {
		t.Errorf("the info file must be removed when pairing ends (err=%v)", err)
	}
}
