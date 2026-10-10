package ghsync

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"koboapps/internal/i18n"
)

// CheckAccess proves the token can read the repository, then tells apart the three things GitHub
// answers with the same 404: the repository (or the token's access to it), the branch, and the
// folder. A repository without any commit is accessible and has nothing to list, so it passes: the
// first sync then simply finds 0 books.
func (client *Client) CheckAccess(ctx context.Context, config Config) error {
	defaultBranch, err := client.DefaultBranch(ctx, config)
	if err != nil {
		return err
	}
	hasCommits, err := client.hasBranches(ctx, config)
	if err != nil || !hasCommits {
		return err
	}
	branch := config.Branch
	if branch == "" {
		branch = defaultBranch
	} else if err := client.checkBranch(ctx, config, branch); err != nil {
		return err
	}
	if config.Path == "" {
		return nil
	}
	return client.checkFolder(ctx, config, branch)
}

func (client *Client) hasBranches(ctx context.Context, config Config) (bool, error) {
	var branches []json.RawMessage
	if err := client.getJSON(ctx, repoEndpoint(config)+"/branches?per_page=1", &branches); err != nil {
		return false, err
	}
	return len(branches) > 0, nil
}

func (client *Client) checkBranch(ctx context.Context, config Config, branch string) error {
	var details json.RawMessage
	err := client.getJSON(ctx, repoEndpoint(config)+"/branches/"+escapePath(branch), &details)
	if failureStatus(err) == http.StatusNotFound {
		return i18n.NewError("msg.err_branch_not_found", "branch", branch, "repo", config.Repo)
	}
	return err
}

func (client *Client) checkFolder(ctx context.Context, config Config, branch string) error {
	endpoint := repoEndpoint(config) + "/contents/" + escapePath(config.Path) + "?ref=" + queryEscape(branch)
	var listing json.RawMessage
	err := client.getJSON(ctx, endpoint, &listing)
	if failureStatus(err) == http.StatusNotFound || (err == nil && !isJSONArray(listing)) {
		return i18n.NewError("msg.err_folder_not_found", "folder", config.Path, "branch", branch)
	}
	return err
}

// isJSONArray tells a folder listing (an array) from a single file (an object).
func isJSONArray(raw json.RawMessage) bool {
	return bytes.HasPrefix(bytes.TrimSpace(raw), []byte("["))
}
