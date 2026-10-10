package update

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"koboapps/internal/i18n"
	"koboapps/internal/release"
)

// Outcomes of a check, stored in last-check.json.
const (
	CheckUpToDate  = "up_to_date"
	CheckAvailable = "available"
	CheckFailed    = "failed"
)

// Error keys of a failed check. Each has a localized reason (update.reason_<key>); a failure
// that fits none is "other" and carries its own report.
const (
	ErrorNoNetwork     = "no_network"
	ErrorUnreachable   = "unreachable"
	ErrorRateLimited   = "rate_limited"
	ErrorBadSignature  = "bad_signature"
	ErrorNoRelease     = "no_release_on_channel"
	ErrorOther         = "other"
	maxOtherReasonSize = 80
)

// CheckRecord is what the last `update check` ended with, kept in data/update/last-check.json
// so the settings screen can say what happened after a check that ran detached. Unlike
// available.json it also records "up to date" and failures.
type CheckRecord struct {
	Time     string  `json:"time"`
	Result   string  `json:"result"`
	Version  string  `json:"version,omitempty"`
	ErrorKey string  `json:"error_key,omitempty"`
	Report   *Report `json:"report,omitempty"` // only for ErrorOther: the failure to describe
}

func (layout Layout) lastCheckFile() string {
	return filepath.Join(layout.workDir(), "last-check.json")
}

// ReadLastCheck returns the stored outcome; ok is false when no check ran or the file is
// unreadable (a damaged record is the same as none).
func ReadLastCheck(layout Layout) (record CheckRecord, ok bool) {
	data, err := os.ReadFile(layout.lastCheckFile())
	if err != nil || json.Unmarshal(data, &record) != nil || record.Result == "" {
		return CheckRecord{}, false
	}
	return record, true
}

func writeLastCheck(layout Layout, record CheckRecord) error {
	record.Time = time.Now().UTC().Format(time.RFC3339)
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return WriteFileAtomically(layout.lastCheckFile(), append(data, '\n'), 0o644)
}

// checkRecordOf turns the end of a check into a record.
func checkRecordOf(result Report, err error, current string) CheckRecord {
	if err != nil {
		return failedCheckRecord(err)
	}
	if result.Key == "update.available" {
		return CheckRecord{Result: CheckAvailable, Version: result.Params["version"]}
	}
	return CheckRecord{Result: CheckUpToDate, Version: current}
}

func failedCheckRecord(err error) CheckRecord {
	failure := reportOfError(err)
	record := CheckRecord{Result: CheckFailed, ErrorKey: errorKeyOf(failure.Key)}
	if record.ErrorKey == ErrorOther {
		record.Report = &failure
	}
	return record
}

func errorKeyOf(messageKey string) string {
	switch messageKey {
	case "update.no_network":
		return ErrorNoNetwork
	case "update.network":
		return ErrorUnreachable
	case "update.rate_limited":
		return ErrorRateLimited
	case "update.bad_signature":
		return ErrorBadSignature
	case "update.no_release":
		return ErrorNoRelease
	}
	return ErrorOther
}

// Describe is the one-line outcome for the settings screen, with the local time of the check.
// It is empty when the outcome is out of date: an "update available" for a version that is
// installed by now.
func (record CheckRecord) Describe(bundle *i18n.Bundle, current string) string {
	when := record.localTime()
	switch record.Result {
	case CheckUpToDate:
		return bundle.Message("update.last_up_to_date", i18n.Params("time", when, "version", record.Version))
	case CheckAvailable:
		if !record.isNewerThan(current) {
			return ""
		}
		return bundle.Message("update.last_available", i18n.Params("time", when, "version", record.Version))
	case CheckFailed:
		return bundle.Message("update.last_failed", i18n.Params("time", when, "reason", record.reason(bundle)))
	}
	return ""
}

func (record CheckRecord) isNewerThan(current string) bool {
	order, err := release.CompareVersions(record.Version, current)
	return err == nil && order > 0
}

// localTime is HH:MM in the device's time zone; the record itself is UTC.
func (record CheckRecord) localTime() string {
	moment, err := time.Parse(time.RFC3339, record.Time)
	if err != nil {
		return "--:--"
	}
	return moment.Local().Format("15:04")
}

func (record CheckRecord) reason(bundle *i18n.Bundle) string {
	if record.ErrorKey != ErrorOther || record.Report == nil {
		return bundle.Message("update.reason_"+record.ErrorKey, nil)
	}
	return shorten(oneLine(record.Report.Render(bundle)), maxOtherReasonSize)
}

func oneLine(text string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(text, `\n`, " ")), " ")
}

func shorten(text string, limit int) string {
	if utf8.RuneCountInString(text) <= limit {
		return text
	}
	return string([]rune(text)[:limit-1]) + "…"
}

// recordCheck stores the outcome of a finished check. A failure to store is only a warning:
// the check itself already ended.
func (updater *Updater) recordCheck(result Report, err error) {
	record := checkRecordOf(result, err, updater.options.CurrentVersion)
	if writeErr := writeLastCheck(updater.layout, record); writeErr != nil {
		fmt.Fprintln(os.Stderr, "warning: cannot write the check outcome:", writeErr)
	}
}
