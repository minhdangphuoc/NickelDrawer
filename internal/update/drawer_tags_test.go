package update

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// repoReleases is the list a repo with drawer, app and catalog releases returns, newest first
// by creation date: the app and catalog releases were created after the drawer's.
func repoReleases() []releaseInfo {
	return []releaseInfo{
		{Tag: "apps"},
		{Tag: "todo-v1.0.2"},
		{Tag: "wallpaper-v0.4.0"},
		{Tag: "v1.0.0-beta.5", Prerelease: true},
		{Tag: "v9.9.9", Draft: true},
		{Tag: "v0.9.0"},
		{Tag: "v1.0.0-beta.4", Prerelease: true},
		{Tag: "vNext"},
		{Tag: "1.2.0"},
	}
}

func TestDrawerSelectionIgnoresAppAndCatalogReleases(t *testing.T) {
	cases := map[string]struct {
		allowPrerelease bool
		want            string
	}{
		"stable skips prereleases, apps and drafts": {false, "v0.9.0"},
		"beta takes the newest drawer prerelease":   {true, "v1.0.0-beta.5"},
	}
	for name, test := range cases {
		got, found := newestDrawerRelease(repoReleases(), test.allowPrerelease)
		if !found || got.Tag != test.want {
			t.Errorf("%s: got %q (found %v), want %q", name, got.Tag, found, test.want)
		}
	}
	onlyApps := []releaseInfo{{Tag: "apps"}, {Tag: "todo-v1.0.2"}}
	for _, allow := range []bool{false, true} {
		if _, found := newestDrawerRelease(onlyApps, allow); found {
			t.Errorf("allowPrerelease=%v: an app release was taken for the drawer", allow)
		}
	}
}

func TestNewestDrawerReleaseIsTheHighestVersionNotTheLatestCreated(t *testing.T) {
	releases := []releaseInfo{{Tag: "v1.0.0"}, {Tag: "v1.2.0"}, {Tag: "v1.1.0"}, {Tag: "v1.2.0-rc.1", Prerelease: true}}
	if got, _ := newestDrawerRelease(releases, true); got.Tag != "v1.2.0" {
		t.Errorf("got %s", got.Tag)
	}
}

func TestFindReleaseOnBothChannelsSkipsAppReleases(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/o/r/releases" {
			http.NotFound(w, r) // the "latest" endpoint would answer with an app release; it is not used
			return
		}
		json.NewEncoder(w).Encode(repoReleases())
	}))
	defer server.Close()
	client := newGithubClient(nil, server.URL)
	for channel, want := range map[string]string{ChannelStable: "v0.9.0", ChannelBeta: "v1.0.0-beta.5"} {
		info, err := client.findRelease(context.Background(), Config{Repo: "o/r", Channel: channel})
		if err != nil || info.Tag != want {
			t.Errorf("%s: tag=%q err=%v, want %s", channel, info.Tag, err, want)
		}
	}
}
