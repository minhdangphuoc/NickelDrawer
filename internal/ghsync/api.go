package ghsync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"koboapps/internal/i18n"
)

const (
	requestTimeout = 60 * time.Second
	maxAttempts    = 3
	userAgent      = "koboapps-ghsync"
)

// fatalError stops the whole run (bad token, rate limit, timeout) instead of
// counting against a single file.
type fatalError struct{ error }

func (err fatalError) Unwrap() error { return err.error }

func fatal(key string, pairs ...string) fatalError {
	return fatalError{i18n.NewError(key, pairs...)}
}

// httpFailure keeps the HTTP status of a refused request next to its message, so callers that know
// more than the client can say what a 404 or 409 means in their context.
type httpFailure struct {
	status int
	err    error
}

func (failure httpFailure) Error() string { return failure.err.Error() }
func (failure httpFailure) Unwrap() error { return failure.err }

// failureStatus is the HTTP status behind err, or 0 when err is not an HTTP refusal.
func failureStatus(err error) int {
	var failure httpFailure
	if errors.As(err, &failure) {
		return failure.status
	}
	return 0
}

// transientError marks failures worth retrying.
type transientError struct{ cause error }

func (err transientError) Error() string { return err.cause.Error() }
func (err transientError) Unwrap() error { return err.cause }

// Client talks to the GitHub REST API.
type Client struct {
	apiBase     string
	token       Token
	http        *http.Client
	backoff     time.Duration
	rateLimited bool
}

// NewClient builds a client; backoff is the first retry delay (doubles).
func NewClient(apiBase string, token Token, backoff time.Duration) *Client {
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 20 * time.Second}).DialContext,
		TLSHandshakeTimeout:   20 * time.Second,
		ResponseHeaderTimeout: requestTimeout,
		MaxIdleConns:          1,
		IdleConnTimeout:       30 * time.Second,
	}
	return &Client{
		apiBase: strings.TrimRight(apiBase, "/"),
		token:   token,
		http:    &http.Client{Transport: transport},
		backoff: backoff,
	}
}

// redact removes the token from text that may be logged or shown.
func (client *Client) redact(text string) string {
	if client.token == "" {
		return text
	}
	return strings.ReplaceAll(text, string(client.token), "***")
}

// withRetry runs attempt up to maxAttempts times while it fails transiently.
func (client *Client) withRetry(ctx context.Context, attempt func() error) error {
	delay := client.backoff
	var err error
	for try := 1; try <= maxAttempts; try++ {
		if client.rateLimited {
			return fatal("msg.err_rate_limited")
		}
		err = attempt()
		var transient transientError
		if err == nil || !errors.As(err, &transient) || try == maxAttempts {
			break
		}
		select {
		case <-ctx.Done():
			return fatal("msg.err_timeout")
		case <-time.After(delay):
		}
		delay *= 2
	}
	var transient transientError
	if errors.As(err, &transient) {
		return i18n.NewError("msg.err_network", "cause", client.redact(err.Error()))
	}
	return err
}

// fetchSpec describes one GET. A non-empty ifNoneMatch asks GitHub to answer 304 when nothing
// changed; such an answer does not count against the rate limit.
type fetchSpec struct {
	endpoint    string
	accept      string
	ifNoneMatch string
}

// fetchResult is what a successful GET reports besides the body.
type fetchResult struct {
	etag        string
	notModified bool
}

// get performs one authenticated GET and hands a successful body to consume.
func (client *Client) get(ctx context.Context, spec fetchSpec, consume func(io.Reader) error) (fetchResult, error) {
	var result fetchResult
	err := client.withRetry(ctx, func() error {
		var attemptErr error
		result, attemptErr = client.getOnce(ctx, spec, consume)
		return attemptErr
	})
	return result, err
}

func (client *Client) getOnce(ctx context.Context, spec fetchSpec, consume func(io.Reader) error) (fetchResult, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, client.apiBase+spec.endpoint, nil)
	if err != nil {
		return fetchResult{}, err
	}
	request.Header.Set("Authorization", "Bearer "+string(client.token))
	request.Header.Set("Accept", spec.accept)
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	request.Header.Set("User-Agent", userAgent)
	if spec.ifNoneMatch != "" {
		request.Header.Set("If-None-Match", spec.ifNoneMatch)
	}

	response, err := client.http.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return fetchResult{}, fatal("msg.err_overall_timeout")
		}
		return fetchResult{}, transientError{errors.New(client.redact(err.Error()))}
	}
	defer response.Body.Close()
	if response.Header.Get("X-RateLimit-Remaining") == "0" {
		client.rateLimited = true
	}
	if response.StatusCode == http.StatusNotModified && spec.ifNoneMatch != "" {
		return fetchResult{etag: spec.ifNoneMatch, notModified: true}, nil
	}
	if response.StatusCode != http.StatusOK {
		return fetchResult{}, client.statusError(response)
	}
	if err := consume(response.Body); err != nil {
		if ctx.Err() != nil {
			return fetchResult{}, fatal("msg.err_overall_timeout")
		}
		return fetchResult{}, err
	}
	return fetchResult{etag: response.Header.Get("ETag")}, nil
}

func (client *Client) statusError(response *http.Response) error {
	status := response.StatusCode
	if status >= 500 {
		return transientError{fmt.Errorf("HTTP %d", status)}
	}
	return httpFailure{status: status, err: client.statusMessage(status)}
}

func (client *Client) statusMessage(status int) error {
	switch {
	case client.rateLimited:
		return fatal("msg.err_rate_limited")
	case status == http.StatusUnauthorized:
		return fatal("msg.err_http_401")
	case status == http.StatusForbidden:
		return fatal("msg.err_http_403")
	case status == http.StatusNotFound:
		return fatal("msg.err_http_404")
	default:
		return i18n.NewError("msg.err_http_status", "status", strconv.Itoa(status))
	}
}

// DefaultBranch returns the repository's default branch. A 404 means the repository does not
// exist or the token cannot see it: GitHub answers the same for both, on purpose.
func (client *Client) DefaultBranch(ctx context.Context, config Config) (string, error) {
	var repo struct {
		DefaultBranch string `json:"default_branch"`
	}
	err := client.getJSON(ctx, repoEndpoint(config), &repo)
	if failureStatus(err) == http.StatusNotFound {
		return "", i18n.NewError("msg.err_repo_not_found", "repo", config.Repo)
	}
	if err == nil && repo.DefaultBranch == "" {
		err = i18n.NewError("msg.err_no_default_branch")
	}
	return repo.DefaultBranch, err
}

// TreeEntry is one item of a git tree.
type TreeEntry struct {
	Path string `json:"path"`
	Type string `json:"type"`
	SHA  string `json:"sha"`
	Size int64  `json:"size"`
}

// Tree is the recursive listing of a branch.
type Tree struct {
	Entries   []TreeEntry `json:"tree"`
	Truncated bool        `json:"truncated"`
}

// TreeListing is a Tree plus the validator GitHub sent with it. NotModified is true when the
// etag given to ListTree is still current; Tree is then empty.
type TreeListing struct {
	Tree
	ETag        string
	NotModified bool
}

// ListTree fetches the full recursive tree of the branch. A repository without commits has no
// tree (GitHub answers 409): that is an empty listing, not a failure. A branch that does not
// exist (404) is reported as such. A non-empty etag makes the request conditional.
func (client *Client) ListTree(ctx context.Context, config Config, branch, etag string) (TreeListing, error) {
	var listing TreeListing
	endpoint := repoEndpoint(config) + "/git/trees/" + url.PathEscape(branch) + "?recursive=1"
	result, err := client.getJSONConditional(ctx, endpoint, etag, &listing.Tree)
	switch failureStatus(err) {
	case http.StatusConflict:
		return TreeListing{}, nil
	case http.StatusNotFound:
		return TreeListing{}, i18n.NewError("msg.err_branch_not_found", "branch", branch, "repo", config.Repo)
	}
	listing.ETag, listing.NotModified = result.etag, result.notModified
	return listing, err
}

// Download streams the raw file content to a sink; sink is called once per
// attempt and must reset its output because retries restart the download.
func (client *Client) Download(ctx context.Context, config Config, branch, repoPath string, sink func(io.Reader) error) error {
	spec := fetchSpec{
		endpoint: repoEndpoint(config) + "/contents/" + escapePath(repoPath) + "?ref=" + url.QueryEscape(branch),
		accept:   "application/vnd.github.raw",
	}
	_, err := client.get(ctx, spec, func(body io.Reader) error {
		if err := sink(body); err != nil {
			return transientError{err}
		}
		return nil
	})
	return err
}

func (client *Client) getJSON(ctx context.Context, endpoint string, target any) error {
	_, err := client.getJSONConditional(ctx, endpoint, "", target)
	return err
}

func (client *Client) getJSONConditional(ctx context.Context, endpoint, etag string, target any) (fetchResult, error) {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	spec := fetchSpec{endpoint: endpoint, accept: "application/vnd.github+json", ifNoneMatch: etag}
	return client.get(ctx, spec, func(body io.Reader) error {
		return json.NewDecoder(body).Decode(target)
	})
}

func repoEndpoint(config Config) string {
	return "/repos/" + url.PathEscape(config.Owner()) + "/" + url.PathEscape(config.Name())
}

func escapePath(repoPath string) string {
	segments := strings.Split(repoPath, "/")
	for index, segment := range segments {
		segments[index] = url.PathEscape(segment)
	}
	return strings.Join(segments, "/")
}
