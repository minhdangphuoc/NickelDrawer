package ghsync

import (
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// PairInfo is what an open pairing session leaves for the app screen: the address and the
// one-time code the Kobo shows. It lives in /tmp (mode 600) and only while pairing is open.
type PairInfo struct {
	URL     string    `json:"url"`
	Code    string    `json:"code"`
	Expires time.Time `json:"expires"`
}

// writePairInfo publishes the session to the screen; an empty path disables it.
func writePairInfo(path string, info PairInfo) error {
	if path == "" {
		return nil
	}
	encoded, err := json.Marshal(info)
	if err != nil {
		return err
	}
	return os.WriteFile(path, encoded, 0o600)
}

// ScreenFiles are the /tmp files the scripts and the pair command leave behind.
type ScreenFiles struct {
	SyncPidFile  string // exists while a background sync runs
	PairPidFile  string // exists while pairing is open
	PairInfoFile string
	// PairResultFile holds how the last pairing session ended.
	PairResultFile string
	// ProgressFile holds the download count of a running sync.
	ProgressFile string
	// RescanFlag exists after a sync added books and is consumed by TakeRescanRequest.
	RescanFlag string
}

// ScreenState is the raw (not localized) data of the GitHub screen. It never holds the token:
// the token file is only checked for existence.
type ScreenState struct {
	IsConfigured bool
	Repo         string
	Branch       string
	Folder       string
	LastRun      string
	LastNew      int
	LastExisting int
	LastErrors   int
	LastError    string // already localized and without the token
	IsRunning    bool
	IsPairing    bool
	Pair         PairInfo    // only filled while IsPairing
	PairOutcome  PairOutcome // how the last session ended; "" while open or never run
	Progress     Progress    // only filled while IsRunning and a file is in progress
	HasProgress  bool
}

// LoadScreenState reads local files only. A broken config or state file is "not configured" or
// "never synced" here; `ghsync status` is the place that reports why.
func LoadScreenState(paths Paths, files ScreenFiles, now time.Time) ScreenState {
	var screen ScreenState
	if config, err := LoadConfig(paths.Config()); err == nil && hasTokenFile(paths) {
		screen.IsConfigured = true
		screen.Repo, screen.Branch, screen.Folder = config.Repo, config.Branch, config.Path
	}
	if state, err := LoadState(paths.State()); err == nil {
		screen.LastRun, screen.LastNew = state.LastRun, state.LastNew
		screen.LastExisting, screen.LastErrors = state.LastExisting, state.LastErrors
		screen.LastError = withoutToken(state.LastError, paths)
	}
	screen.IsRunning = isProcessAlive(files.SyncPidFile)
	if screen.IsRunning {
		screen.Progress, screen.HasProgress = readProgress(files.ProgressFile)
	}
	if isProcessAlive(files.PairPidFile) {
		if info, ok := readPairInfo(files.PairInfoFile, now); ok {
			screen.IsPairing, screen.Pair = true, info
		}
	}
	if !screen.IsPairing && files.PairResultFile != "" {
		screen.PairOutcome = ReadPairOutcome(files.PairResultFile)
	}
	return screen
}

func readProgress(path string) (Progress, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Progress{}, false
	}
	var progress Progress
	if json.Unmarshal(raw, &progress) != nil || progress.Total <= 0 {
		return Progress{}, false
	}
	return progress, true
}

// TakeRescanRequest reports whether a finished sync asked for a library rescan, and clears the
// marker: the host performs each request it receives once and does not remember it.
func TakeRescanRequest(flagPath string) bool {
	if flagPath == "" {
		return false
	}
	return os.Remove(flagPath) == nil
}

// withoutToken is defense in depth: state.json is written redacted, but the screen shows its
// text, so a token that got in by hand or by an older version is removed here too.
func withoutToken(text string, paths Paths) string {
	raw, err := os.ReadFile(paths.Token())
	token := strings.TrimSpace(string(raw))
	if err != nil || token == "" {
		return text
	}
	return strings.ReplaceAll(text, token, "***")
}

func hasTokenFile(paths Paths) bool {
	info, err := os.Stat(paths.Token())
	return err == nil && info.Size() > 0
}

func readPairInfo(path string, now time.Time) (PairInfo, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return PairInfo{}, false
	}
	var info PairInfo
	if json.Unmarshal(raw, &info) != nil || info.URL == "" || info.Code == "" || !now.Before(info.Expires) {
		return PairInfo{}, false
	}
	return info, true
}

func isProcessAlive(pidFile string) bool {
	raw, err := os.ReadFile(pidFile)
	if err != nil {
		return false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil || pid <= 0 {
		return false
	}
	err = syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}
