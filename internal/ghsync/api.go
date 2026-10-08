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

// get performs one authenticated GET and hands a successful body to consume.
func (client *Client) get(ctx context.Context, endpoint, accept string, consume func(io.Reader) error) error {
	return client.withRetry(ctx, func() error {
		return client.getOnce(ctx, endpoint, accept, consume)
	})
}

func (client *Client) getOnce(ctx context.Context, endpoint, accept string, consume func(io.Reader) error) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, client.apiBase+endpoint, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+string(client.token))
	request.Header.Set("Accept", accept)
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	request.Header.Set("User-Agent", userAgent)

	response, err := client.http.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return fatal("msg.err_overall_timeout")
		}
		return transientError{errors.New(client.redact(err.Error()))}
	}
	defer response.Body.Close()
	if response.Header.Get("X-RateLimit-Remaining") == "0" {
		client.rateLimited = true
	}
	if response.StatusCode != http.StatusOK {
		return client.statusError(response)
	}
	if err := consume(response.Body); err != nil {
		if ctx.Err() != nil {
			return fatal("msg.err_overall_timeout")
		}
		return err
	}
	return nil
}

func (client *Client) statusError(response *http.Response) error {
	switch status := response.StatusCode; {
	case client.rateLimited:
		return fatal("msg.err_rate_limited")
	case status == http.StatusUnauthorized:
		return fatal("msg.err_http_401")
	case status == http.StatusForbidden:
		return fatal("msg.err_http_403")
	case status == http.StatusNotFound:
		return fatal("msg.err_http_404")
	case status >= 500:
		return transientError{fmt.Errorf("HTTP %d", status)}
	default:
		return i18n.NewError("msg.err_http_status", "status", strconv.Itoa(status))
	}
}

// DefaultBranch returns the repository's default branch.
func (client *Client) DefaultBranch(ctx context.Context, config Config) (string, error) {
	var repo struct {
		DefaultBranch string `json:"default_branch"`
	}
	err := client.getJSON(ctx, repoEndpoint(config), &repo)
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

// ListTree fetches the full recursive tree of the branch.
func (client *Client) ListTree(ctx context.Context, config Config, branch string) (Tree, error) {
	var tree Tree
	endpoint := repoEndpoint(config) + "/git/trees/" + url.PathEscape(branch) + "?recursive=1"
	err := client.getJSON(ctx, endpoint, &tree)
	return tree, err
}

// Download streams the raw file content to a sink; sink is called once per
// attempt and must reset its output because retries restart the download.
func (client *Client) Download(ctx context.Context, config Config, branch, repoPath string, sink func(io.Reader) error) error {
	endpoint := repoEndpoint(config) + "/contents/" + escapePath(repoPath) + "?ref=" + url.QueryEscape(branch)
	return client.get(ctx, endpoint, "application/vnd.github.raw", func(body io.Reader) error {
		if err := sink(body); err != nil {
			return transientError{err}
		}
		return nil
	})
}

func (client *Client) getJSON(ctx context.Context, endpoint string, target any) error {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	return client.get(ctx, endpoint, "application/vnd.github+json", func(body io.Reader) error {
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
