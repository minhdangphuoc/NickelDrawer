package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"koboapps/internal/i18n"
)

const (
	maxMetadataBytes = 1 << 20
	maxReleaseJSON   = 1 << 20
	maxSignature     = 1 << 10
	maxZipBytes      = 256 << 20
	downloadTimeout  = 15 * time.Minute
)

type assetInfo struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

type releaseInfo struct {
	Tag        string      `json:"tag_name"`
	Draft      bool        `json:"draft"`
	Prerelease bool        `json:"prerelease"`
	Assets     []assetInfo `json:"assets"`
}

func (info releaseInfo) asset(name string) (assetInfo, error) {
	for _, asset := range info.Assets {
		if asset.Name == name {
			return asset, nil
		}
	}
	return assetInfo{}, i18n.NewError("update.missing_asset", "name", name, "tag", info.Tag)
}

type githubClient struct {
	http    *http.Client
	apiBase string
}

func newGithubClient(httpClient *http.Client, apiBase string) *githubClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: downloadTimeout}
	}
	return &githubClient{http: httpClient, apiBase: strings.TrimRight(apiBase, "/")}
}

// findRelease returns the newest release the channel allows. GitHub's "latest" endpoint
// already skips drafts and prereleases; the beta channel reads the list instead.
func (client *githubClient) findRelease(ctx context.Context, config Config) (releaseInfo, error) {
	if config.Channel == ChannelBeta {
		return client.newestOfList(ctx, config.Repo)
	}
	var info releaseInfo
	if err := client.getJSON(ctx, "/repos/"+config.Repo+"/releases/latest", &info); err != nil {
		return releaseInfo{}, err
	}
	if info.Draft || info.Prerelease {
		return releaseInfo{}, i18n.NewError("update.no_release", "repo", config.Repo)
	}
	return info, nil
}

func (client *githubClient) newestOfList(ctx context.Context, repo string) (releaseInfo, error) {
	var releases []releaseInfo
	if err := client.getJSON(ctx, "/repos/"+repo+"/releases?per_page=10", &releases); err != nil {
		return releaseInfo{}, err
	}
	for _, info := range releases {
		if !info.Draft {
			return info, nil
		}
	}
	return releaseInfo{}, i18n.NewError("update.no_release", "repo", repo)
}

func (client *githubClient) getJSON(ctx context.Context, endpoint string, target any) error {
	body, status, err := client.get(ctx, client.apiBase+endpoint, maxMetadataBytes)
	if err != nil {
		return err
	}
	if status == http.StatusNotFound {
		return i18n.NewError("update.no_release", "repo", repoOf(endpoint))
	}
	if status != http.StatusOK {
		return i18n.NewError("update.network").WithCause(fmt.Errorf("GitHub answered HTTP %d", status))
	}
	if err := json.Unmarshal(body, target); err != nil {
		return i18n.NewError("update.network").WithCause(fmt.Errorf("unreadable GitHub answer: %w", err))
	}
	return nil
}

func repoOf(endpoint string) string {
	parts := strings.Split(strings.TrimPrefix(endpoint, "/repos/"), "/")
	if len(parts) >= 2 {
		return parts[0] + "/" + parts[1]
	}
	return endpoint
}

// download fetches an asset into memory (small files) with a size cap.
func (client *githubClient) download(ctx context.Context, url string, limit int64) ([]byte, error) {
	body, status, err := client.get(ctx, url, limit)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, i18n.NewError("update.network").WithCause(fmt.Errorf("download answered HTTP %d", status))
	}
	return body, nil
}

func (client *githubClient) get(ctx context.Context, url string, limit int64) ([]byte, int, error) {
	response, err := client.open(ctx, url)
	if err != nil {
		return nil, 0, err
	}
	defer response.Body.Close()
	if isRateLimited(response) {
		return nil, 0, i18n.NewError("update.rate_limited")
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return nil, 0, i18n.NewError("update.network").WithCause(err)
	}
	if int64(len(body)) > limit {
		return nil, 0, i18n.NewError("update.network").WithCause(fmt.Errorf("%s is larger than %d bytes", url, limit))
	}
	return body, response.StatusCode, nil
}

// downloadToFile streams a large asset to disk, so memory stays small on the Kobo.
func (client *githubClient) downloadToFile(ctx context.Context, url, destination string, limit int64) (err error) {
	response, err := client.open(ctx, url)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return i18n.NewError("update.network").WithCause(fmt.Errorf("download answered HTTP %d", response.StatusCode))
	}
	output, err := os.Create(destination)
	if err != nil {
		return fmt.Errorf("cannot create %s: %w", destination, err)
	}
	defer func() {
		if closeErr := output.Close(); err == nil {
			err = closeErr
		}
	}()
	written, err := io.Copy(output, io.LimitReader(response.Body, limit+1))
	if err != nil {
		return i18n.NewError("update.network").WithCause(err)
	}
	if written > limit {
		return i18n.NewError("update.network").WithCause(fmt.Errorf("download is larger than %d bytes", limit))
	}
	return nil
}

func (client *githubClient) open(ctx context.Context, url string) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, i18n.NewError("update.network").WithCause(err)
	}
	request.Header.Set("Accept", "application/vnd.github+json, application/octet-stream")
	request.Header.Set("User-Agent", "nickeldrawer-update")
	response, err := client.http.Do(request)
	if err != nil {
		return nil, transportError(err)
	}
	return response, nil
}

// transportError tells "no connection at all" (Wi-Fi off, DNS gone) from a server that does
// not answer, so the settings screen can say which one to fix.
func transportError(err error) error {
	var dnsErr *net.DNSError
	var opErr *net.OpError
	isOffline := errors.As(err, &dnsErr) || (errors.As(err, &opErr) && opErr.Op == "dial" && !opErr.Timeout())
	if isOffline {
		return i18n.NewError("update.no_network").WithCause(err)
	}
	return i18n.NewError("update.network").WithCause(err)
}

// isRateLimited recognises GitHub's two ways to say "too many requests".
func isRateLimited(response *http.Response) bool {
	if response.StatusCode == http.StatusTooManyRequests {
		return true
	}
	return response.StatusCode == http.StatusForbidden && response.Header.Get("X-RateLimit-Remaining") == "0"
}
