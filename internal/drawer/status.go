package drawer

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"koboapps/internal/manifest"
)

const (
	statusActionID = "status"
	cacheFileName  = "koboapps-drawer-status.json"
)

// Status is the drawer's view of one app's optional "status" action.
type Status struct {
	Line    string `json:"line"`
	Running bool   `json:"running"`
}

type cacheEntry struct {
	At int64 `json:"at"` // unix nanoseconds
	Status
}

type cacheFile struct {
	Locale  string                `json:"locale"` // status text is localized by the script
	Entries map[string]cacheEntry `json:"entries"`
}

// parseStatus reads a status script's stdout: a first line {"running":true,"line":"..."} or
// else the first non-empty line as plain text.
func parseStatus(stdout string) Status {
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var structured struct {
			Running bool   `json:"running"`
			Line    string `json:"line"`
		}
		if strings.HasPrefix(line, "{") && json.Unmarshal([]byte(line), &structured) == nil {
			return Status{Line: structured.Line, Running: structured.Running}
		}
		return Status{Line: line}
	}
	return Status{}
}

func hasStatusAction(app manifest.App) (manifest.Action, bool) {
	for _, action := range app.Actions {
		if action.ID == statusActionID {
			return action, true
		}
	}
	return manifest.Action{}, false
}

// collectStatuses returns a status per app id. Fresh cache entries are reused; the rest run in
// parallel, each with its own short budget. A slow or failing script yields an empty status
// (and is cached too, so a hanging script costs one second per 30 s, not per call).
func (c Config) collectStatuses(apps []manifest.App) map[string]Status {
	cached := c.loadCache()
	now := c.now()
	statuses := map[string]Status{}
	var stale []manifest.App
	for _, app := range apps {
		if _, hasStatus := hasStatusAction(app); !hasStatus {
			continue
		}
		entry, found := cached.Entries[app.ID]
		if found && isFresh(entry, now, c.cacheTTL()) {
			statuses[app.ID] = entry.Status
			continue
		}
		stale = append(stale, app)
	}
	if len(stale) == 0 {
		return statuses
	}
	fresh := c.runStatuses(stale)
	entries := map[string]cacheEntry{}
	for id, entry := range cached.Entries {
		if _, kept := statuses[id]; kept {
			entries[id] = entry
		}
	}
	for i, app := range stale {
		statuses[app.ID] = fresh[i]
		entries[app.ID] = cacheEntry{At: now.UnixNano(), Status: fresh[i]}
	}
	c.saveCache(cacheFile{Locale: c.Locale, Entries: entries})
	return statuses
}

func (c Config) runStatuses(apps []manifest.App) []Status {
	results := make([]Status, len(apps))
	var wait sync.WaitGroup
	for i, app := range apps {
		wait.Add(1)
		go func() {
			defer wait.Done()
			results[i] = c.runStatus(app)
		}()
	}
	wait.Wait()
	return results
}

func (c Config) runStatus(app manifest.App) Status {
	action, _ := hasStatusAction(app)
	ctx, cancel := context.WithTimeout(context.Background(), c.statusTimeout())
	defer cancel()
	output := c.runScript()(ctx, filepath.Join(app.Dir, filepath.FromSlash(action.Run)))
	if output.Err != nil || output.TimedOut || output.ExitCode != 0 {
		return Status{}
	}
	return parseStatus(output.Stdout)
}

func (c Config) cachePath() string { return filepath.Join(c.cacheDir(), cacheFileName) }

// loadCache never fails: a missing, broken or other-language cache is an empty one.
func (c Config) loadCache() cacheFile {
	data, err := os.ReadFile(c.cachePath())
	var cache cacheFile
	if err != nil || json.Unmarshal(data, &cache) != nil || cache.Locale != c.Locale || cache.Entries == nil {
		return cacheFile{Entries: map[string]cacheEntry{}}
	}
	return cache
}

// saveCache is best effort: without a cache the next call just runs the scripts again.
func (c Config) saveCache(cache cacheFile) {
	data, err := json.Marshal(cache)
	if err != nil {
		return
	}
	temp, err := os.CreateTemp(c.cacheDir(), cacheFileName+".*")
	if err != nil {
		return
	}
	_, writeErr := temp.Write(data)
	closeErr := temp.Close()
	if writeErr != nil || closeErr != nil || os.Rename(temp.Name(), c.cachePath()) != nil {
		os.Remove(temp.Name())
	}
}

// isFresh also rejects entries from the future, which a clock change on the Kobo can cause.
func isFresh(entry cacheEntry, now time.Time, ttl time.Duration) bool {
	age := now.Sub(time.Unix(0, entry.At))
	return age >= 0 && age < ttl
}
