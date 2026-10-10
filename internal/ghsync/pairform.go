package ghsync

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"koboapps/internal/i18n"
)

const (
	githubCheckTimeout = 30 * time.Second
	minTokenLength     = 20
	maxTokenLength     = 255
	maxBranchLength    = 200
)

var (
	tokenPattern  = regexp.MustCompile(`^[A-Za-z0-9_]+$`)
	branchPattern = regexp.MustCompile(`^[A-Za-z0-9._/-]+$`)
	tokenPrefixes = []string{"github_pat_", "ghp_"}
)

func constantTimeEqual(want, given string) bool {
	return subtle.ConstantTimeCompare([]byte(want), []byte(given)) == 1
}

// pairInput is the validated form content. The token is a Token so it prints as ***.
type pairInput struct {
	config Config
	token  Token
}

func (session *pairSession) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	header := w.Header()
	header.Set("Cache-Control", "no-store")
	header.Set("Referrer-Policy", "no-referrer")
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; form-action 'self'; base-uri 'none'")
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		session.mutex.Lock()
		defer session.mutex.Unlock()
		session.renderUnlessClosed(w, pageForm{})
	case http.MethodPost:
		session.submit(w, r)
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// renderUnlessClosed shows the form, or the page for a session that already ended.
// The caller holds the mutex.
func (session *pairSession) renderUnlessClosed(w http.ResponseWriter, page pageForm) {
	if session.isClosed(w) {
		return
	}
	session.render(w, http.StatusOK, page)
}

// isClosed answers a request to a finished or expired session; the caller holds the mutex.
func (session *pairSession) isClosed(w http.ResponseWriter) bool {
	if session.isExpired() {
		session.finish(PairExpired)
	}
	switch {
	case session.attempts >= maxCodeAttempts:
		session.render(w, http.StatusTooManyRequests, pageForm{Status: "locked"})
	case session.isExpired():
		session.render(w, http.StatusGone, pageForm{Status: "expired"})
	case session.finished:
		session.render(w, http.StatusGone, pageForm{Status: "done"})
	default:
		return false
	}
	return true
}

func (session *pairSession) submit(w http.ResponseWriter, r *http.Request) {
	session.mutex.Lock()
	defer session.mutex.Unlock()
	if session.isClosed(w) {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxFormBytes)
	if err := r.ParseForm(); err != nil {
		session.render(w, http.StatusBadRequest, pageForm{Error: session.describe(i18n.NewError("msg.err_pair_form"), "")})
		return
	}
	values := pageForm{
		Repo:   strings.TrimSpace(r.PostForm.Get("repo")),
		Branch: strings.TrimSpace(r.PostForm.Get("branch")),
		Folder: strings.TrimSpace(r.PostForm.Get("folder")),
	}
	if !session.codeMatches(strings.TrimSpace(r.PostForm.Get("code"))) {
		session.rejectCode(w, values)
		return
	}
	token := Token(strings.TrimSpace(r.PostForm.Get("token")))
	session.pair(w, r.Context(), values, token)
}

func (session *pairSession) rejectCode(w http.ResponseWriter, values pageForm) {
	session.attempts++
	session.options.Log.Printf("pairing: wrong code (%d of %d)", session.attempts, maxCodeAttempts)
	if session.attempts >= maxCodeAttempts {
		session.finish(PairLocked)
		session.render(w, http.StatusTooManyRequests, pageForm{Status: "locked"})
		return
	}
	left := maxCodeAttempts - session.attempts
	values.Error = session.options.Messages.Message("msg.pair_wrong_code", i18n.Params("left", itoa(left)))
	session.render(w, http.StatusForbidden, values)
}

// pair validates, tests the credentials against GitHub, then saves them.
func (session *pairSession) pair(w http.ResponseWriter, ctx context.Context, values pageForm, token Token) {
	logger := NewLogger(session.options.Log.out, token)
	input, err := buildPairInput(session.options.Paths, values, token)
	if err == nil {
		err = session.checkGitHub(ctx, input)
	}
	if err == nil {
		err = savePairing(session.options.Paths, input)
	}
	if err != nil {
		logger.Printf("pairing failed: %s", err)
		values.Error = session.describe(err, string(token))
		session.render(w, http.StatusUnprocessableEntity, values)
		return
	}
	logger.Printf("paired with %s", input.config.Repo)
	session.finish(PairPaired)
	session.render(w, http.StatusOK, pageForm{Status: "ok", Repo: input.config.Repo})
}

func (session *pairSession) checkGitHub(ctx context.Context, input pairInput) error {
	ctx, cancel := context.WithTimeout(ctx, githubCheckTimeout)
	defer cancel()
	client := NewClient(session.options.APIBase, input.token, session.options.Backoff)
	return client.CheckAccess(ctx, input.config)
}

// describe renders err in the phone's language with the token removed.
func (session *pairSession) describe(err error, secret string) string {
	text := session.options.Messages.Describe(err)
	if secret != "" {
		text = strings.ReplaceAll(text, secret, "***")
	}
	return text
}

// buildPairInput validates the form. An existing config.json keeps its dest and extensions.
func buildPairInput(paths Paths, values pageForm, token Token) (pairInput, error) {
	if err := validateTokenFormat(token); err != nil {
		return pairInput{}, err
	}
	if values.Branch != "" && !isValidBranch(values.Branch) {
		return pairInput{}, i18n.NewError("msg.err_pair_branch_invalid")
	}
	config := Config{Repo: values.Repo, Branch: values.Branch, Path: values.Folder}
	if existing, err := LoadConfig(paths.Config()); err == nil {
		config.Dest, config.Extensions = existing.Dest, existing.Extensions
	}
	validated, err := config.withDefaults().validated()
	if err != nil {
		return pairInput{}, err
	}
	return pairInput{config: validated, token: token}, nil
}

func validateTokenFormat(token Token) error {
	value := string(token)
	hasPrefix := false
	for _, prefix := range tokenPrefixes {
		hasPrefix = hasPrefix || strings.HasPrefix(value, prefix)
	}
	if !hasPrefix || len(value) < minTokenLength || len(value) > maxTokenLength || !tokenPattern.MatchString(value) {
		return i18n.NewError("msg.err_pair_token_format")
	}
	return nil
}

func isValidBranch(branch string) bool {
	return len(branch) <= maxBranchLength && branchPattern.MatchString(branch) &&
		!strings.Contains(branch, "..") && !strings.HasPrefix(branch, "/") && !strings.HasSuffix(branch, "/")
}

// savePairing writes the token first, then the config, each atomically with mode 600.
func savePairing(paths Paths, input pairInput) error {
	if err := os.MkdirAll(filepath.Dir(paths.Token()), 0o755); err != nil {
		return i18n.NewError("msg.err_mkdir").WithCause(err)
	}
	if err := writeFileAtomic(paths.Token(), []byte(string(input.token)+"\n")); err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(input.config, "", "  ")
	if err != nil {
		return i18n.NewError("msg.err_write").WithCause(err)
	}
	return writeFileAtomic(paths.Config(), append(encoded, '\n'))
}

// writeFileAtomic writes a temp file in the same folder and renames it over target.
func writeFileAtomic(target string, content []byte) error {
	temp, err := os.CreateTemp(filepath.Dir(target), ".ghsync-pair-*")
	if err != nil {
		return i18n.NewError("msg.err_tempfile").WithCause(err)
	}
	defer os.Remove(temp.Name())
	// Chmod can fail on vfat, where every file looks the same; CreateTemp already asked for 0600.
	_ = temp.Chmod(0o600)
	_, writeErr := temp.Write(content)
	syncErr := temp.Sync()
	closeErr := temp.Close()
	for _, err := range []error{writeErr, syncErr, closeErr} {
		if err != nil {
			return i18n.NewError("msg.err_write").WithCause(err)
		}
	}
	if err := os.Rename(temp.Name(), target); err != nil {
		return i18n.NewError("msg.err_write").WithCause(err)
	}
	return nil
}
