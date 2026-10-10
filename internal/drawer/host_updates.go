package drawer

import (
	"path/filepath"

	"koboapps/internal/manifest"
	"koboapps/internal/update"
)

const (
	// hostScreenDir holds ui.json and locales/ of the built-in updates screen, laid out like
	// an app folder so the screen goes through the same loader, validator and localizer.
	hostScreenDir = "screens/updates"
	// The scripts live in the host bin/; their path is relative to hostScreenDir.
	updateCheckScript = "../../bin/update-check.sh"
	updateApplyScript = "../../bin/update-apply.sh"

	updateActionCheck = "check"
	updateActionApply = "apply"
)

// hostApp describes a built-in item that has a screen as a pseudo app, so `drawer screen`,
// `drawer event` and `drawer run` treat it like any other app. Only _updates has one.
func (c Config) hostApp(id string) (manifest.App, error) {
	if id != HostUpdatesID {
		return manifest.App{}, newError(CodeHostItem, "app", id)
	}
	return manifest.App{
		ID: id, Name: c.message("drawer.updates_name", nil), Dir: filepath.Join(c.HostDir, hostScreenDir),
		Actions: []manifest.Action{
			{ID: stateActionID, Label: stateActionID, Run: stateActionID, Output: manifest.OutputShow},
			{ID: updateActionCheck, Label: c.message("drawer.updates_action", nil), Run: updateCheckScript,
				Output: manifest.OutputShow, Wifi: true},
			{ID: updateActionApply, Label: c.message("drawer.updates_apply", nil), Run: updateApplyScript,
				Output: manifest.OutputShow, Wifi: true},
		},
	}, nil
}

// updateLayout derives the onboard root from the host folder (<root>/.adds/koboapps).
func (c Config) updateLayout() update.Layout {
	return update.Layout{OnboardRoot: filepath.Dir(filepath.Dir(c.HostDir))}
}

// pendingUpdate returns the update the last check found, when it is newer than this build.
func (c Config) pendingUpdate() (update.Availability, bool) {
	found, hasCheck := update.ReadAvailability(c.updateLayout())
	return found, hasCheck && found.IsNewerThan(c.Version)
}

// hostState is the state of the built-in updates screen. It reads the files `appsctl update`
// writes, so the screen never touches the network.
func (c Config) hostState(string) map[string]any {
	layout := c.updateLayout()
	status, hasStatus, _ := update.ReadStatus(layout)
	pending, hasPending := c.pendingUpdate()
	isBusy := hasStatus && status.State == update.StateRunning
	return map[string]any{
		"current":    c.Version,
		"latest":     c.latestVersionText(pending, hasPending),
		"notes":      pending.Notes,
		"message":    c.updateMessage(status, hasStatus),
		"can_update": hasPending && !isBusy,
		"can_check":  !isBusy,
	}
}

func (c Config) latestVersionText(pending update.Availability, hasPending bool) string {
	switch {
	case hasPending:
		return pending.Version
	case c.hasCheckedUpToDate():
		return c.message("drawer.updates_latest_current", nil)
	}
	return c.message("drawer.updates_latest_none", nil)
}

func (c Config) hasCheckedUpToDate() bool {
	found, hasCheck := update.ReadAvailability(c.updateLayout())
	return hasCheck && !found.IsAvailable
}

// updateMessage shows the last run when it is news: running, failed, or an install that
// needs a restart. A plain "up to date" is already in the version line.
func (c Config) updateMessage(status update.Status, hasStatus bool) string {
	if !hasStatus {
		return ""
	}
	switch {
	case status.State == update.StateRunning, status.State == update.StateFailed:
	case status.Report.Key == "update.applied", status.Report.Key == "update.applied_reboot",
		status.Report.Key == "update.rolled_back":
	default:
		return ""
	}
	return status.Report.Render(c.Bundle)
}

// updatesItem is the "Updates" row of the system section; it shows the pending update as a
// status line and a badge.
func (c Config) updatesItem() Item {
	item := c.hostItem(HostUpdatesID, "drawer.updates_name", HostUpdatesAct, "drawer.updates_action", "updates.png")
	item.Screen = true
	if _, hasPending := c.pendingUpdate(); hasPending {
		item.Badge = true
		item.Status = c.message("drawer.updates_status", nil)
	}
	return item
}
