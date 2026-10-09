package main

import (
	"encoding/json"
	"flag"
	"io"
	"strconv"
	"time"

	"koboapps/internal/ghsync"
	"koboapps/internal/i18n"
)

const (
	defaultSyncPidFile = "/tmp/ghsync.pid"
	lastSyncLayout     = "2006-01-02 15:04"
	expiresLayout      = "15:04"
)

// screenVariables is the whole output of `ghsync state`: an explicit allowlist of the GitHub
// screen's variables. The token is not in this type, so it cannot reach the output.
type screenVariables struct {
	Status      string `json:"status"`
	Configured  bool   `json:"configured"`
	Repo        string `json:"repo"`
	Branch      string `json:"branch"`
	Folder      string `json:"folder"`
	LastSync    string `json:"last_sync"`
	Summary     string `json:"summary"`
	Running     bool   `json:"running"`
	CanSync     bool   `json:"can_sync"`
	Pairing     bool   `json:"pairing"`
	PairURL     string `json:"pair_url"`
	PairCode    string `json:"pair_code"`
	PairExpires string `json:"pair_expires"`
	PairHint    string `json:"pair_hint"`
}

func stateCommand(args []string, out io.Writer) int {
	flags := flag.NewFlagSet("state", flag.ContinueOnError)
	flags.SetOutput(out)
	var common commonFlags
	common.register(flags)
	files := ghsync.ScreenFiles{}
	flags.StringVar(&files.SyncPidFile, "sync-pid-file", defaultSyncPidFile, "pid file of a running background sync")
	flags.StringVar(&files.PairPidFile, "pair-pid-file", defaultPairPidFile, "pid file of an open pairing session")
	flags.StringVar(&files.PairInfoFile, "pair-info-file", defaultPairInfoFile, "address and code of an open pairing session")
	if err := flags.Parse(args); err != nil {
		return exitUsage
	}
	screen := ghsync.LoadScreenState(ghsync.Paths{AppDir: common.appDir}, files, time.Now())
	if err := json.NewEncoder(out).Encode(variablesOf(screen, common.messages())); err != nil {
		return 1
	}
	return 0
}

func variablesOf(screen ghsync.ScreenState, messages *i18n.Bundle) screenVariables {
	variables := screenVariables{
		Status:     headerStatus(screen, messages),
		Configured: screen.IsConfigured,
		Repo:       screen.Repo,
		Branch:     screen.Branch,
		Folder:     screen.Folder,
		LastSync:   formatTime(screen.LastRun, lastSyncLayout),
		Summary:    syncSummary(screen, messages),
		Running:    screen.IsRunning,
		CanSync:    screen.IsConfigured && !screen.IsRunning,
		Pairing:    screen.IsPairing,
		PairHint:   messages.Message("msg.state_pair_idle", nil),
	}
	if screen.IsConfigured {
		variables.Branch = defaulted(screen.Branch, messages.Message("msg.state_default_branch", nil))
		variables.Folder = defaulted(screen.Folder, messages.Message("msg.state_whole_repo", nil))
	}
	if screen.IsPairing {
		variables.PairURL, variables.PairCode = screen.Pair.URL, screen.Pair.Code
		variables.PairExpires = messages.Message("msg.state_pair_until", i18n.Params("time", screen.Pair.Expires.Local().Format(expiresLayout)))
		variables.PairHint = messages.Message("msg.state_pair_open", nil)
	}
	return variables
}

func headerStatus(screen ghsync.ScreenState, messages *i18n.Bundle) string {
	switch {
	case screen.IsRunning:
		return messages.Message("msg.state_status_running", nil)
	case screen.IsPairing:
		return messages.Message("msg.state_status_pairing", nil)
	case !screen.IsConfigured:
		return messages.Message("msg.state_status_unpaired", nil)
	}
	return messages.Message("msg.state_status_ready", nil)
}

func syncSummary(screen ghsync.ScreenState, messages *i18n.Bundle) string {
	switch {
	case screen.LastRun == "":
		return messages.Message("msg.state_never", nil)
	case screen.HasLastError:
		return messages.Message("msg.state_failed", nil)
	}
	return messages.Message("msg.state_summary", i18n.Params(
		"new", strconv.Itoa(screen.LastNew), "existing", strconv.Itoa(screen.LastExisting),
		"errors", strconv.Itoa(screen.LastErrors)))
}

func defaulted(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

// formatTime shows an RFC 3339 time in local time; text that does not parse is shown as it is.
func formatTime(text, layout string) string {
	parsed, err := time.Parse(time.RFC3339, text)
	if err != nil {
		return text
	}
	return parsed.Local().Format(layout)
}
