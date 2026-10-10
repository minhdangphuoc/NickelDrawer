package ghsync

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"koboapps/internal/i18n"
)

// leakyToken is long enough to be found by a plain substring search.
const leakyToken = "github_pat_" + "LEAKCHECKLEAKCHECKLEAKCHECK0123456789"

// everyTextOfARun collects what a run may show or keep: the error in both forms, the log, and
// every file the run wrote below the app folder except the token file itself.
func everyTextOfARun(t *testing.T, e env, err error, messages *i18n.Bundle) string {
	t.Helper()
	var texts []string
	if err != nil {
		texts = append(texts, err.Error(), messages.Describe(err))
	}
	texts = append(texts, e.log.String())
	filepath.WalkDir(e.paths.AppDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() || path == e.paths.Token() {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr == nil {
			texts = append(texts, string(content))
		}
		return nil
	})
	return strings.Join(texts, "\n")
}

func TestTheTokenNeverReachesErrorsLogsOrStateWhateverGoesWrong(t *testing.T) {
	messages, err := i18n.LoadBundle("../../apps/ghsync/locales", "en")
	if err != nil {
		t.Fatal(err)
	}
	statuses := []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusTeapot, http.StatusBadGateway}
	for _, status := range statuses {
		fake := newFakeGitHub(t, sampleFiles())
		fake.status = status
		e := newEnv(t, "")
		must(t, os.WriteFile(e.paths.Token(), []byte(leakyToken+"\n"), 0o600))
		_, runErr := Run(context.Background(), withToken(e.options(t, fake), e))
		if text := everyTextOfARun(t, e, runErr, messages); strings.Contains(text, leakyToken) {
			t.Errorf("status %d leaked the token:\n%s", status, text)
		}
	}
}

func TestTheTokenIsRemovedFromNetworkErrorsEvenWhenTheServerAddressHoldsIt(t *testing.T) {
	messages, _ := i18n.LoadBundle("../../apps/ghsync/locales", "en")
	e := newEnv(t, "")
	must(t, os.WriteFile(e.paths.Token(), []byte(leakyToken+"\n"), 0o600))
	options := e.options(t, newFakeGitHub(t, nil))
	options.APIBase = "http://127.0.0.1:1/" + leakyToken // nothing listens: the error quotes the URL
	_, err := Run(context.Background(), withToken(options, e))
	if err == nil {
		t.Fatal("want a network error")
	}
	if text := everyTextOfARun(t, e, err, messages); strings.Contains(text, leakyToken) {
		t.Fatalf("leaked:\n%s", text)
	}
}

func TestScrubbedKeepsTheMessageKeyButDropsTheToken(t *testing.T) {
	inner := errors.New("write failed for " + leakyToken)
	err := i18n.NewError("msg.err_write", "extra", leakyToken).WithCause(inner)
	clean := scrubbed(err, Token(leakyToken))
	if strings.Contains(clean.Error(), leakyToken) || !hasMessageKey(clean, "msg.err_write") {
		t.Fatalf("scrubbed = %v", clean)
	}
	untouched := i18n.NewError("msg.err_timeout")
	if scrubbed(untouched, Token(leakyToken)) != error(untouched) {
		t.Error("an error without the token must be returned as it is")
	}
}

func TestTokenAndConfigAreWrittenPrivateAndTheTokenIsInNoOtherFile(t *testing.T) {
	e := newPairEnv(t, 200)
	form := goodForm()
	form["token"] = leakyToken
	status, body := e.post(t, form)
	if status != http.StatusOK || strings.Contains(body, leakyToken) {
		t.Fatalf("%d %s", status, body)
	}
	appDir := filepath.Dir(filepath.Dir(e.paths.Token()))
	err := filepath.WalkDir(appDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return walkErr
		}
		info, _ := entry.Info()
		if info.Mode().Perm()&0o077 != 0 {
			t.Errorf("%s is readable by others: %v", path, info.Mode().Perm())
		}
		content, _ := os.ReadFile(path)
		if strings.Contains(string(content), leakyToken) && path != e.paths.Token() {
			t.Errorf("%s holds the token", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(e.log.String(), leakyToken) {
		t.Error("the log holds the token")
	}
}

// A fake that echoes the Authorization header into its error body must not get that text shown.
func TestAnErrorBodyThatEchoesTheTokenIsNeverShown(t *testing.T) {
	messages, _ := i18n.LoadBundle("../../apps/ghsync/locales", "en")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad credentials: "+r.Header.Get("Authorization"), http.StatusTeapot)
	}))
	defer server.Close()
	e := newEnv(t, "")
	must(t, os.WriteFile(e.paths.Token(), []byte(leakyToken+"\n"), 0o600))
	options := e.options(t, newFakeGitHub(t, nil))
	options.APIBase = server.URL
	_, err := Run(context.Background(), withToken(options, e))
	if text := everyTextOfARun(t, e, err, messages); strings.Contains(text, leakyToken) {
		t.Fatalf("leaked:\n%s", text)
	}
}

// withToken makes the run open its own data/log.txt, the way the app does, so the test covers
// the redaction Run sets up itself.
func withToken(options Options, e env) Options {
	options.Log = nil
	return options
}
