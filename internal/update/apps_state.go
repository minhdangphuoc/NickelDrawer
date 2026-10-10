package update

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"koboapps/internal/i18n"
	"koboapps/internal/manifest"
	"koboapps/internal/release"
)

// Kinds of app job, stored in apps-job.json.
const (
	AppsJobCheck    = "check"
	AppsJobUpdate   = "update"
	AppsJobInstall  = "install"
	AppsJobRollback = "rollback"
)

// Phases of one app inside a job; each has a localized label (appupdate.phase_<phase>).
const (
	PhaseStop     = "stop"
	PhaseDownload = "download"
	PhaseInstall  = "install"
)

// AppsJob is what the app updater is doing or did last, for the App updates screen. It lives in
// data/update/apps-job.json.
type AppsJob struct {
	State  string `json:"state"` // StateRunning, StateDone or StateFailed
	Job    string `json:"job"`
	PID    int    `json:"pid,omitempty"`
	Done   int    `json:"done,omitempty"`  // 1-based position of the app being handled
	Total  int    `json:"total,omitempty"` // apps in this job
	AppID  string `json:"app,omitempty"`
	Phase  string `json:"phase,omitempty"`
	Time   string `json:"time"`
	Report Report `json:"report"`
}

// ReadAppsJob returns the stored job; ok is false when none ran. A running job whose process
// is gone is reported as failed (interrupted).
func ReadAppsJob(layout Layout) (job AppsJob, ok bool) {
	data, err := os.ReadFile(layout.appsJobFile())
	if err != nil || json.Unmarshal(data, &job) != nil || job.State == "" {
		return AppsJob{}, false
	}
	if job.State == StateRunning && !isProcessAlive(job.PID) {
		job = AppsJob{State: StateFailed, Job: job.Job, AppID: job.AppID, Time: job.Time,
			Report: report("appupdate.error.interrupted")}
	}
	return job, true
}

func writeAppsJob(layout Layout, job AppsJob) error {
	job.Time = time.Now().UTC().Format(time.RFC3339)
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}
	return WriteFileAtomically(layout.appsJobFile(), append(data, '\n'), 0o644)
}

// IsAppsJobRunning is true while another process runs an app job.
func IsAppsJobRunning(layout Layout) bool {
	job, found := ReadAppsJob(layout)
	return found && job.State == StateRunning
}

// appsSequence is the newest catalog sequence this device accepted (anti-rollback).
type appsSequence struct {
	Sequence int64 `json:"sequence"`
}

func readAppsSequence(layout Layout) int64 {
	data, err := os.ReadFile(layout.appsSequenceFile())
	var stored appsSequence
	if err != nil || json.Unmarshal(data, &stored) != nil {
		return 0
	}
	return stored.Sequence
}

func writeAppsSequence(layout Layout, sequence int64) error {
	data, err := json.Marshal(appsSequence{Sequence: sequence})
	if err != nil {
		return err
	}
	return WriteFileAtomically(layout.appsSequenceFile(), append(data, '\n'), 0o644)
}

// ReadAppsCatalog returns the catalog the last successful check verified. It is a cache for the
// screen: every install fetches and verifies the catalog again. A damaged cache is no cache.
func ReadAppsCatalog(layout Layout) (*release.Catalog, bool) {
	data, err := os.ReadFile(layout.appsCatalogFile())
	if err != nil {
		return nil, false
	}
	catalog, err := release.ParseCatalog(data)
	return catalog, err == nil
}

// AppsCheckRecord is the outcome of the last catalog check.
type AppsCheckRecord struct {
	Time   string  `json:"time"`
	Result string  `json:"result"` // CheckUpToDate for "checked", or CheckFailed
	Report *Report `json:"report,omitempty"`
}

// ReadAppsCheck returns the last catalog check; ok is false when none ran.
func ReadAppsCheck(layout Layout) (record AppsCheckRecord, ok bool) {
	data, err := os.ReadFile(layout.appsCheckFile())
	if err != nil || json.Unmarshal(data, &record) != nil || record.Result == "" {
		return AppsCheckRecord{}, false
	}
	return record, true
}

func writeAppsCheck(layout Layout, record AppsCheckRecord) error {
	record.Time = time.Now().UTC().Format(time.RFC3339)
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return WriteFileAtomically(layout.appsCheckFile(), append(data, '\n'), 0o644)
}

// AppState says what a catalog app needs on this device.
type AppState string

// States of a catalog app.
const (
	AppStateUpdate       AppState = "update"
	AppStateInstalled    AppState = "installed"
	AppStateNotInstalled AppState = "not_installed"
)

// AppEntry is one catalog app compared with what is installed.
type AppEntry struct {
	ID        string
	Name      string
	Installed string // version on the device, "" when not installed
	Latest    string // version in the catalog
	State     AppState
	IsHidden  bool
}

// brokenAppVersion stands for an installed folder that does not load: any catalog version
// is newer, so the update repairs it.
const brokenAppVersion = "0.0.0"

// InstalledAppVersions maps each app folder to its version. A folder that fails to load is
// there with brokenAppVersion, so it is offered a repairing update instead of "not installed".
func InstalledAppVersions(layout Layout) map[string]string {
	versions := map[string]string{}
	scan, err := manifest.LoadAll(layout.appsDir())
	if err != nil {
		return versions
	}
	for _, app := range scan.Apps {
		versions[app.ID] = app.Version
	}
	for _, failure := range scan.Failures {
		versions[failure.AppID] = brokenAppVersion
	}
	return versions
}

// AppEntries compares every catalog app with the installed versions. The hidden filter of the
// "not installed" list is the caller's, because it depends on the user's settings. A catalog
// version below the installed one counts as installed: it is never offered.
func AppEntries(catalog *release.Catalog, installed map[string]string, language string) []AppEntry {
	entries := make([]AppEntry, 0, len(catalog.Apps))
	for _, app := range catalog.Apps {
		entry := AppEntry{ID: app.ID, Name: app.DisplayName(language), Latest: app.Version, IsHidden: app.Hidden}
		version, isInstalled := installed[app.ID]
		switch {
		case !isInstalled:
			entry.State = AppStateNotInstalled
		default:
			entry.Installed, entry.State = version, AppStateInstalled
			if order, err := release.CompareVersions(app.Version, version); err == nil && order > 0 {
				entry.State = AppStateUpdate
			}
		}
		entries = append(entries, entry)
	}
	return entries
}

// appFailure is the report of a failed app job; unknown errors become appupdate.error.other.
func appFailure(err error) Report {
	var messageErr *i18n.MessageError
	if errors.As(err, &messageErr) {
		return reportOfError(err)
	}
	return report("appupdate.error.other", "cause", err.Error())
}

func (layout Layout) requestRescan() {
	if err := WriteFileAtomically(layout.appsRescanMarker(), nil, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "warning: cannot ask for a drawer rescan:", err)
	}
}

// ConsumeRescanRequest reports whether a finished job asked for a rescan, and clears the ask,
// so the mod is told once.
func ConsumeRescanRequest(layout Layout) bool {
	if err := os.Remove(layout.appsRescanMarker()); err != nil {
		return false
	}
	return true
}

// WithName adds the display name of the report's app as {name}, for messages that mention it.
func (r Report) WithName(name func(id string) string) Report {
	id := r.Params["app"]
	if id == "" {
		return r
	}
	params := map[string]string{"name": name(id)}
	for key, value := range r.Params {
		params[key] = value
	}
	return Report{Key: r.Key, Params: params}
}

// DescribeAppError is the localized text of an app updater error, naming the app.
func (updater *Updater) DescribeAppError(bundle *i18n.Bundle, err error) string {
	return appFailure(err).WithName(updater.AppName).Render(bundle)
}

// RenderAppReport is the localized text of an app updater result, naming the app.
func (updater *Updater) RenderAppReport(bundle *i18n.Bundle, result Report) string {
	return result.WithName(updater.AppName).Render(bundle)
}
