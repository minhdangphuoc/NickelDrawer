package update

import (
	"encoding/json"
	"errors"
	"os"
	"syscall"
	"time"

	"koboapps/internal/i18n"
)

// Run states stored in status.json.
const (
	StateRunning = "running"
	StateDone    = "done"
	StateFailed  = "failed"
)

// JobCheck tags a status written by `update check`. Statuses of apply and rollback carry no
// job, so files written before the tag existed still mean "an install".
const JobCheck = "check"

// Report is a localized result: a message key and its parameters.
type Report struct {
	Key    string            `json:"key"`
	Params map[string]string `json:"params,omitempty"`
}

func report(key string, pairs ...string) Report {
	return Report{Key: key, Params: i18n.Params(pairs...)}
}

// Render prints the report in the bundle's language.
func (r Report) Render(bundle *i18n.Bundle) string {
	return bundle.Message(r.Key, r.Params)
}

// Status is what `appsctl update status` shows: the last apply or rollback.
type Status struct {
	State  string `json:"state"`
	Job    string `json:"job,omitempty"`
	PID    int    `json:"pid,omitempty"`
	Time   string `json:"time"`
	Report Report `json:"report"`
}

// ReadStatus returns the stored status; ok is false when no update ever ran.
// A "running" status whose process has died is reported as failed (interrupted).
func ReadStatus(layout Layout) (status Status, ok bool, err error) {
	data, err := os.ReadFile(layout.statusFile())
	if os.IsNotExist(err) {
		return Status{}, false, nil
	}
	if err != nil {
		return Status{}, false, err
	}
	if err := json.Unmarshal(data, &status); err != nil {
		return Status{}, false, err
	}
	if status.State == StateRunning && !isProcessAlive(status.PID) {
		status = Status{State: StateFailed, Job: status.Job, Time: status.Time, Report: report("update.interrupted")}
	}
	return status, true, nil
}

func writeStatus(layout Layout, status Status) error {
	status.Time = time.Now().UTC().Format(time.RFC3339)
	data, err := json.Marshal(status)
	if err != nil {
		return err
	}
	return WriteFileAtomically(layout.statusFile(), append(data, '\n'), 0o644)
}

func isProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

// reportOfError converts any error to a report; unknown errors become update.failed.
func reportOfError(err error) Report {
	var messageErr *i18n.MessageError
	if !errors.As(err, &messageErr) {
		return report("update.failed", "cause", err.Error())
	}
	params := map[string]string{}
	for name, value := range messageErr.Params {
		params[name] = value
	}
	if messageErr.Cause != nil {
		params["cause"] = messageErr.Cause.Error()
	}
	return Report{Key: messageErr.Key, Params: params}
}
