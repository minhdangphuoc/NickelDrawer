package ghsync

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"koboapps/internal/i18n"
)

const testToken = "ghp_SECRET123"

type fakeGitHub struct {
	files      map[string]string // repo path -> content
	truncated  bool
	corrupt    map[string]bool // serve wrong bytes
	status     int             // forced status for every request
	flaky      int32           // 5xx responses to serve before succeeding
	downloads  int32
	server     *httptest.Server
	authHeader string
}

func blobSHA(content string) string {
	sum := sha1.Sum([]byte(fmt.Sprintf("blob %d\x00%s", len(content), content)))
	return hex.EncodeToString(sum[:])
}

func newFakeGitHub(t *testing.T, files map[string]string) *fakeGitHub {
	fake := &fakeGitHub{files: files, corrupt: map[string]bool{}}
	fake.server = httptest.NewServer(http.HandlerFunc(fake.handle))
	t.Cleanup(fake.server.Close)
	return fake
}

func (fake *fakeGitHub) handle(w http.ResponseWriter, r *http.Request) {
	fake.authHeader = r.Header.Get("Authorization")
	if fake.status != 0 {
		w.WriteHeader(fake.status)
		return
	}
	if atomic.AddInt32(&fake.flaky, -1) >= 0 {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	path := r.URL.Path
	switch {
	case path == "/repos/o/r":
		json.NewEncoder(w).Encode(map[string]string{"default_branch": "main"})
	case strings.HasPrefix(path, "/repos/o/r/git/trees/"):
		fake.serveTree(w)
	case strings.HasPrefix(path, "/repos/o/r/contents/"):
		atomic.AddInt32(&fake.downloads, 1)
		repoPath := strings.TrimPrefix(path, "/repos/o/r/contents/")
		content := fake.files[repoPath]
		if fake.corrupt[repoPath] {
			content = "tampered"
		}
		w.Write([]byte(content))
	default:
		http.NotFound(w, r)
	}
}

func (fake *fakeGitHub) serveTree(w http.ResponseWriter) {
	var entries []TreeEntry
	for path, content := range fake.files {
		entries = append(entries, TreeEntry{Path: path, Type: "blob", SHA: blobSHA(content), Size: int64(len(content))})
	}
	entries = append(entries, TreeEntry{Path: "books", Type: "tree"})
	json.NewEncoder(w).Encode(Tree{Entries: entries, Truncated: fake.truncated})
}

type env struct {
	paths Paths
	dest  string
	log   *bytes.Buffer
}

func newEnv(t *testing.T, configExtra string) env {
	dir := t.TempDir()
	dest := filepath.Join(dir, "dest")
	os.MkdirAll(filepath.Join(dir, "app", "data"), 0o755)
	paths := Paths{AppDir: filepath.Join(dir, "app")}
	config := fmt.Sprintf(`{"repo":"o/r","path":"books","dest":%q%s}`, dest, configExtra)
	must(t, os.WriteFile(paths.Config(), []byte(config), 0o644))
	must(t, os.WriteFile(paths.Token(), []byte(" "+testToken+"\n"), 0o600))
	return env{paths: paths, dest: dest, log: &bytes.Buffer{}}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func (e env) run(t *testing.T, fake *fakeGitHub) (Summary, error) {
	return Run(context.Background(), Options{
		Paths: e.paths, APIBase: fake.server.URL, Backoff: time.Millisecond,
		Log: NewLogger(e.log, testToken),
	})
}

func sampleFiles() map[string]string {
	return map[string]string{
		"books/a.epub":           "AAA",
		"books/sub/b.KEPUB.EPUB": "BBBB",
		"books/c.pdf":            "CCCCC",
		"books/notes.txt":        "ignore me",
		"other/d.epub":           "outside path",
	}
}

func TestListingFiltersByPathAndExtension(t *testing.T) {
	fake := newFakeGitHub(t, sampleFiles())
	e := newEnv(t, "")
	summary, err := e.run(t, fake)
	must(t, err)
	if summary.New != 3 {
		t.Fatalf("want 3 new, got %+v", summary)
	}
	for _, rel := range []string{"a.epub", "sub/b.KEPUB.EPUB", "c.pdf"} {
		if _, err := os.Stat(filepath.Join(e.dest, rel)); err != nil {
			t.Errorf("missing %s: %v", rel, err)
		}
	}
	if _, err := os.Stat(filepath.Join(e.dest, "notes.txt")); err == nil {
		t.Error("txt must be filtered out")
	}
	if fake.authHeader != "Bearer "+testToken {
		t.Errorf("token not trimmed/sent: %q", fake.authHeader)
	}
}

func TestRerunIsIdempotentAndStateRoundTrips(t *testing.T) {
	fake := newFakeGitHub(t, sampleFiles())
	e := newEnv(t, "")
	_, err := e.run(t, fake)
	must(t, err)
	before := atomic.LoadInt32(&fake.downloads)
	summary, err := e.run(t, fake)
	must(t, err)
	if summary.New != 0 || summary.Existing != 3 || atomic.LoadInt32(&fake.downloads) != before {
		t.Fatalf("rerun not idempotent: %+v", summary)
	}
	state, err := LoadState(e.paths.State())
	must(t, err)
	if len(state.Files) != 3 || state.Files["books/a.epub"].SHA != blobSHA("AAA") || state.LastRun == "" {
		t.Fatalf("bad state: %+v", state)
	}
}

func TestExistingFileIsNeverOverwrittenButRecorded(t *testing.T) {
	fake := newFakeGitHub(t, sampleFiles())
	e := newEnv(t, "")
	must(t, os.MkdirAll(e.dest, 0o755))
	must(t, os.WriteFile(filepath.Join(e.dest, "a.epub"), []byte("USER COPY"), 0o644))
	summary, err := e.run(t, fake)
	must(t, err)
	content, _ := os.ReadFile(filepath.Join(e.dest, "a.epub"))
	if string(content) != "USER COPY" {
		t.Fatal("existing file was overwritten")
	}
	state, _ := LoadState(e.paths.State())
	if summary.New != 2 || summary.Existing != 1 || state.Files["books/a.epub"].SHA == "" {
		t.Fatalf("unexpected: %+v %+v", summary, state)
	}
}

func TestDeletedLocalFileIsSkippedAndCounted(t *testing.T) {
	fake := newFakeGitHub(t, sampleFiles())
	e := newEnv(t, "")
	_, err := e.run(t, fake)
	must(t, err)
	must(t, os.Remove(filepath.Join(e.dest, "c.pdf")))
	summary, err := e.run(t, fake)
	must(t, err)
	if summary.New != 0 || summary.Deleted != 1 || summary.Existing != 2 {
		t.Fatalf("want 1 skipped deletion, got %+v", summary)
	}
	if _, err := os.Stat(filepath.Join(e.dest, "c.pdf")); err == nil {
		t.Fatal("deleted file was downloaded again")
	}
}

func TestRedownloadDeletedOverride(t *testing.T) {
	fake := newFakeGitHub(t, sampleFiles())
	e := newEnv(t, "")
	_, err := e.run(t, fake)
	must(t, err)
	must(t, os.Remove(filepath.Join(e.dest, "c.pdf")))
	summary, err := Run(context.Background(), Options{
		Paths: e.paths, APIBase: fake.server.URL, Backoff: time.Millisecond,
		Log: NewLogger(e.log, testToken), RedownloadDeleted: true,
	})
	must(t, err)
	if summary.New != 1 || summary.Deleted != 0 {
		t.Fatalf("want 1 re-download, got %+v", summary)
	}
}

func TestLogFileHoldsNoHandleBetweenWrites(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "log.txt")
	writer, err := NewLogFile(logPath)
	must(t, err)
	logger := NewLogger(writer, "")
	logger.Printf("one")
	must(t, os.Remove(logPath)) // a held handle would keep writing to the unlinked inode
	logger.Printf("two")
	content, err := os.ReadFile(logPath)
	must(t, err)
	if strings.Contains(string(content), "one") || !strings.Contains(string(content), "two") {
		t.Fatalf("log = %q", content)
	}
}

func TestRepoDeletionLeavesLocalFiles(t *testing.T) {
	fake := newFakeGitHub(t, sampleFiles())
	e := newEnv(t, "")
	_, err := e.run(t, fake)
	must(t, err)
	delete(fake.files, "books/a.epub")
	_, err = e.run(t, fake)
	must(t, err)
	if _, err := os.Stat(filepath.Join(e.dest, "a.epub")); err != nil {
		t.Fatal("local file must survive repo deletion")
	}
}

func TestShaMismatchDiscardsTempFile(t *testing.T) {
	fake := newFakeGitHub(t, sampleFiles())
	fake.corrupt["books/a.epub"] = true
	e := newEnv(t, "")
	summary, err := e.run(t, fake)
	must(t, err)
	if summary.Errors != 1 || summary.New != 2 {
		t.Fatalf("unexpected %+v", summary)
	}
	if _, err := os.Stat(filepath.Join(e.dest, "a.epub")); err == nil {
		t.Error("corrupt file must not be installed")
	}
	leftovers, _ := filepath.Glob(filepath.Join(e.dest, ".ghsync-*"))
	if len(leftovers) != 0 {
		t.Errorf("temp files left: %v", leftovers)
	}
}

func TestPathSanitizing(t *testing.T) {
	rejected := []string{"../x.epub", "a/../../x.epub", "/abs.epub", "a//b.epub", "a/./b.epub", "c:evil.epub", "a/b\x01.epub", ""}
	for _, input := range rejected {
		if got, err := sanitizeRelativePath(input); err == nil {
			t.Errorf("%q should be rejected, got %q", input, got)
		}
	}
	got, err := sanitizeRelativePath(`Sách/Tập 1? <hay>.epub`)
	must(t, err)
	if got != "Sách/Tập 1_ _hay_.epub" {
		t.Errorf("unexpected mapping %q", got)
	}
	local, err := localPathFor("/dest", "books", "books/x/y.epub")
	if err != nil || local != "/dest/x/y.epub" {
		t.Errorf("got %q %v", local, err)
	}
}

// hasMessageKey reports whether err carries the locale message key.
func hasMessageKey(err error, key string) bool {
	var messageErr *i18n.MessageError
	return errors.As(err, &messageErr) && messageErr.Key == key
}

func TestHTTPErrorsHaveClearMessagesAndNoToken(t *testing.T) {
	cases := map[int]string{401: "msg.err_http_401", 403: "msg.err_http_403", 404: "msg.err_http_404"}
	for status, want := range cases {
		fake := newFakeGitHub(t, sampleFiles())
		fake.status = status
		e := newEnv(t, "")
		_, err := e.run(t, fake)
		if !hasMessageKey(err, want) {
			t.Errorf("status %d: got %v", status, err)
		}
		if strings.Contains(err.Error(), testToken) || strings.Contains(e.log.String(), testToken) {
			t.Error("token leaked")
		}
	}
}

func TestRedactionOfTokenInNetworkErrors(t *testing.T) {
	client := NewClient("http://x", Token(testToken), time.Millisecond)
	if got := client.redact("boom " + testToken + " end"); strings.Contains(got, testToken) {
		t.Errorf("not redacted: %s", got)
	}
	logBuffer := &bytes.Buffer{}
	NewLogger(logBuffer, testToken).Printf("token=%s", testToken)
	if strings.Contains(logBuffer.String(), testToken) {
		t.Error("logger leaked token")
	}
	if fmt.Sprint(Token(testToken)) != "***" {
		t.Error("Token.String must hide value")
	}
}

func TestRetriesTransientServerErrors(t *testing.T) {
	fake := newFakeGitHub(t, sampleFiles())
	fake.flaky = 2
	e := newEnv(t, "")
	summary, err := e.run(t, fake)
	must(t, err)
	if summary.New != 3 {
		t.Fatalf("retry did not recover: %+v", summary)
	}
}

func TestGivesUpAfterThreeServerErrors(t *testing.T) {
	fake := newFakeGitHub(t, sampleFiles())
	fake.flaky = 100
	e := newEnv(t, "")
	if _, err := e.run(t, fake); err == nil {
		t.Fatal("expected failure")
	}
}

func TestTruncatedTreeWarns(t *testing.T) {
	fake := newFakeGitHub(t, sampleFiles())
	fake.truncated = true
	e := newEnv(t, "")
	summary, err := e.run(t, fake)
	must(t, err)
	if !summary.Truncated || !strings.Contains(e.log.String(), "truncated") {
		t.Fatalf("no warning: %+v %s", summary, e.log.String())
	}
}

func TestRateLimitStopsRun(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "0")
		json.NewEncoder(w).Encode(map[string]string{"default_branch": "main"})
	}))
	defer server.Close()
	e := newEnv(t, "")
	_, err := Run(context.Background(), Options{Paths: e.paths, APIBase: server.URL, Backoff: time.Millisecond, Log: NewLogger(e.log, testToken)})
	if !hasMessageKey(err, "msg.err_rate_limited") {
		t.Fatalf("want rate limit error, got %v", err)
	}
}

func TestDryRunWritesNothing(t *testing.T) {
	fake := newFakeGitHub(t, sampleFiles())
	e := newEnv(t, "")
	summary, err := Run(context.Background(), Options{Paths: e.paths, APIBase: fake.server.URL, DryRun: true, Log: NewLogger(e.log, testToken)})
	must(t, err)
	if summary.New != 3 || atomic.LoadInt32(&fake.downloads) != 0 {
		t.Fatalf("dry run misbehaved: %+v", summary)
	}
	if _, err := os.Stat(e.dest); err == nil {
		t.Error("dry run created dest")
	}
}

func TestConfigValidation(t *testing.T) {
	bad := []string{`{"repo":"nope"}`, `{"repo":"o/r","dest":"relative"}`, `{"repo":"o/r","extensions":["epub"]}`, `{"repo":"o/r","path":"../x"}`, `{`}
	for _, body := range bad {
		file := filepath.Join(t.TempDir(), "config.json")
		must(t, os.WriteFile(file, []byte(body), 0o644))
		if _, err := LoadConfig(file); err == nil {
			t.Errorf("%s should fail", body)
		}
	}
	file := filepath.Join(t.TempDir(), "config.json")
	must(t, os.WriteFile(file, []byte(`{"repo":"o/r"}`), 0o644))
	config, err := LoadConfig(file)
	must(t, err)
	if config.Dest != defaultDest() || len(config.Extensions) != 5 {
		t.Errorf("defaults missing: %+v", config)
	}
}

func TestDefaultDestinationIsBooksGitHubAndAnExplicitDestStillWorks(t *testing.T) {
	loadDest := func(body string) string {
		file := filepath.Join(t.TempDir(), "config.json")
		must(t, os.WriteFile(file, []byte(body), 0o644))
		config, err := LoadConfig(file)
		must(t, err)
		return config.Dest
	}
	if got := loadDest(`{"repo":"o/r"}`); got != "/mnt/onboard/Books/GitHub" {
		t.Errorf("default dest = %q, want /mnt/onboard/Books/GitHub", got)
	}
	if got := loadDest(`{"repo":"o/r","dest":"/mnt/onboard/GitHub"}`); got != "/mnt/onboard/GitHub" {
		t.Errorf("an explicit dest must be kept, got %q", got)
	}
	if got := loadDest(`{"repo":"o/r","dest":"/mnt/onboard/Books"}`); got != "/mnt/onboard/Books" {
		t.Errorf("an explicit dest must be kept, got %q", got)
	}
}

func TestTokenPermissionCheck(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "token")
	must(t, os.WriteFile(file, []byte("abc\n"), 0o644))
	must(t, os.Chmod(file, 0o644))
	if _, _, err := LoadToken(file); !hasMessageKey(err, "msg.err_token_permissions") {
		t.Errorf("want chmod hint, got %v", err)
	}
	must(t, os.Chmod(file, 0o600))
	token, warning, err := LoadToken(file)
	if err != nil || string(token) != "abc" || warning != "" {
		t.Errorf("got %q %q %v", token, warning, err)
	}
}

func TestStatusErrorIsSavedInTheUserLanguage(t *testing.T) {
	fake := newFakeGitHub(t, sampleFiles())
	fake.status = 404
	e := newEnv(t, "")
	messages := i18n.NewBundle(i18n.Catalog{"msg.err_http_404": "not found here"})
	_, err := Run(context.Background(), Options{
		Paths: e.paths, APIBase: fake.server.URL, Backoff: time.Millisecond,
		Log: NewLogger(e.log, testToken), Messages: messages,
	})
	if err == nil {
		t.Fatal("want an error")
	}
	state, loadErr := LoadState(e.paths.State())
	must(t, loadErr)
	if state.LastError != "not found here" {
		t.Fatalf("LastError = %q", state.LastError)
	}
}
