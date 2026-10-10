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

	// noValue fills a row of the status list that has nothing to show yet.
	noValue = "—"

	// maxErrorChars keeps the failure reason within the notice on the e-ink screen.
	maxErrorChars = 160
)

// hostRequest is a service the screen asks the host for (docs/spec/drawer-protocol.md, Requests).
type hostRequest struct {
	Request string `json:"request"`
	Mode    string `json:"mode,omitempty"`
}

// screenVariables is the whole output of `ghsync state`: an explicit allowlist of the GitHub
// screen's variables. The token is not in this type, so it cannot reach the output.
type screenVariables struct {
	Status      string `json:"status"`
	Configured  bool   `json:"configured"`
	Guide       string `json:"guide"`
	Repo        string `json:"repo"`
	Branch      string `json:"branch"`
	Folder      string `json:"folder"`
	LastSync    string `json:"last_sync"`
	Added       string `json:"added"`
	Summary     string `json:"summary"`
	LastError   string `json:"last_error"`
	Running     bool   `json:"running"`
	CanSync     bool   `json:"can_sync"`
	Progress    string `json:"progress"` // the steps headline: what is happening
	StepDone    string `json:"step_done"`
	StepTotal   string `json:"step_total"`
	StepDetail  string `json:"step_detail"`
	Pairing     bool   `json:"pairing"`
	PairRow     string `json:"pair_row"`
	PairURL     string `json:"pair_url"`
	PairCode    string `json:"pair_code"`
	PairExpires string `json:"pair_expires"`
	PairHint    string `json:"pair_hint"`
	PairNote    string `json:"pair_note"`

	Requests []hostRequest `json:"requests,omitempty"`
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
	flags.StringVar(&files.PairResultFile, "pair-result-file", defaultPairResultFile, "how the last pairing session ended")
	flags.StringVar(&files.ProgressFile, "progress-file", defaultProgressFile, "download count of a running sync")
	flags.StringVar(&files.RescanFlag, "rescan-flag", defaultRescanFlag, "marker left by a sync that added books")
	if err := flags.Parse(args); err != nil {
		return exitUsage
	}
	now := time.Now()
	screen := ghsync.LoadScreenState(ghsync.Paths{AppDir: common.appDir}, files, now)
	variables := variablesOf(screen, common.messages(), now)
	if !screen.IsRunning && ghsync.TakeRescanRequest(files.RescanFlag) {
		variables.Requests = []hostRequest{{Request: "rescan", Mode: "quick"}}
	}
	if err := json.NewEncoder(out).Encode(variables); err != nil {
		return 1
	}
	return 0
}

func variablesOf(screen ghsync.ScreenState, messages *i18n.Bundle, now time.Time) screenVariables {
	variables := screenVariables{
		Status:     headerStatus(screen, messages),
		Configured: screen.IsConfigured,
		Repo:       defaulted(screen.Repo, noValue),
		Branch:     noValue,
		Folder:     noValue,
		LastSync:   defaulted(formatTime(screen.LastRun, messages), noValue),
		Added:      noValue,
		Summary:    syncSummary(screen, messages),
		LastError:  shortened(screen.LastError, maxErrorChars),
		Running:    screen.IsRunning,
		CanSync:    screen.IsConfigured && !screen.IsRunning,
		Pairing:    screen.IsPairing,
		PairHint:   messages.Message("msg.state_pair_idle", nil),
		PairNote:   pairNote(screen, messages),
	}
	if screen.LastRun != "" {
		variables.Added = strconv.Itoa(screen.LastNew)
	}
	if screen.IsConfigured {
		variables.Branch = defaulted(screen.Branch, messages.Message("msg.state_default_branch", nil))
		variables.Folder = defaulted(screen.Folder, messages.Message("msg.state_whole_repo", nil))
	} else {
		variables.Guide = messages.Message("msg.state_guide", nil)
	}
	if screen.IsRunning {
		fillProgress(&variables, screen, messages)
	}
	if screen.IsPairing {
		fillPairing(&variables, screen, messages, now)
	}
	return variables
}

func fillProgress(variables *screenVariables, screen ghsync.ScreenState, messages *i18n.Bundle) {
	if !screen.HasProgress {
		variables.Progress = messages.Message("msg.state_step_preparing", nil)
		return
	}
	variables.Progress = messages.Message("msg.state_step_downloading", nil)
	variables.StepDone, variables.StepTotal = strconv.Itoa(screen.Progress.Done), strconv.Itoa(screen.Progress.Total)
	variables.StepDetail = screen.Progress.File
}

func fillPairing(variables *screenVariables, screen ghsync.ScreenState, messages *i18n.Bundle, now time.Time) {
	variables.PairRow = messages.Message("msg.state_pair_row_open", nil)
	variables.PairURL, variables.PairCode = screen.Pair.URL, screen.Pair.Code
	variables.PairHint = messages.Message("msg.state_pair_open", nil)
	variables.PairExpires = messages.Message("msg.state_pair_until", i18n.Params(
		"minutes", strconv.Itoa(minutesLeft(screen.Pair.Expires, now)),
		"time", messages.Clock(screen.Pair.Expires)))
}

// minutesLeft rounds up, so the last partial minute still reads "1 minute".
func minutesLeft(expires, now time.Time) int {
	left := expires.Sub(now)
	if left <= 0 {
		return 0
	}
	return int((left + time.Minute - 1) / time.Minute)
}

// pairNote says how the last pairing session ended; it is empty while one is open.
func pairNote(screen ghsync.ScreenState, messages *i18n.Bundle) string {
	switch screen.PairOutcome {
	case ghsync.PairPaired:
		return messages.Message("msg.pair_done", i18n.Params("repo", screen.Repo))
	case ghsync.PairLocked:
		return messages.Message("msg.pair_result_locked", nil)
	case ghsync.PairExpired:
		return messages.Message("msg.pair_result_expired", nil)
	case ghsync.PairStopped:
		return messages.Message("msg.pair_result_stopped", nil)
	}
	return ""
}

// shortened cuts text to limit characters (not bytes: Vietnamese letters take several).
func shortened(text string, limit int) string {
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit-1]) + "…"
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
	case screen.LastError != "":
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

// formatTime shows an RFC 3339 time as the locale writes a date and time, in local time; text
// that does not parse is shown as it is.
func formatTime(text string, messages *i18n.Bundle) string {
	parsed, err := time.Parse(time.RFC3339, text)
	if err != nil {
		return text
	}
	return messages.DateTime(parsed)
}
