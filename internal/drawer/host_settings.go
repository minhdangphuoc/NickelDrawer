package drawer

import (
	"path/filepath"

	"koboapps/internal/manifest"
	"koboapps/internal/uikit"
	"koboapps/internal/update"
)

const (
	// hostScreenDir holds ui.json and locales/ of the built-in settings screen, laid out like
	// an app folder so the screen goes through the same loader, validator and localizer.
	hostScreenDir = "screens/settings"
	// The scripts live in the host bin/; their path is relative to hostScreenDir.
	updateCheckScript = "../../bin/update-check.sh"
	updateApplyScript = "../../bin/update-apply.sh"

	updateActionCheck = "check"
	updateActionApply = "apply"
	debugActionID     = "debug"
)

// hostApp describes a built-in item that has a screen as a pseudo app, so `drawer screen`,
// `drawer event` and `drawer run` treat it like any other app. Only _settings has one.
func (c Config) hostApp(id string) (manifest.App, error) {
	if id != HostSettingsID {
		return manifest.App{}, newError(CodeHostItem, "app", id)
	}
	return manifest.App{
		ID: id, Name: c.message("drawer.settings_name", nil), Dir: filepath.Join(c.HostDir, hostScreenDir),
		Actions: []manifest.Action{
			{ID: stateActionID, Label: stateActionID, Run: stateActionID, Output: manifest.OutputShow},
			{ID: updateActionCheck, Label: c.message("drawer.settings_check", nil), Run: updateCheckScript,
				Output: manifest.OutputShow, Wifi: true},
			{ID: updateActionApply, Label: c.message("drawer.settings_apply", nil), Run: updateApplyScript,
				Output: manifest.OutputShow, Wifi: true},
			{ID: appsCheckActionID, Label: c.message("drawer.settings_check", nil), Run: updateAppsCheckScript,
				Output: manifest.OutputShow, Wifi: true},
			{ID: appsUpdateAllActionID, Label: c.message("appupdate.update_all", map[string]string{"count": "0"}), Run: updateAppsAllScript,
				Output: manifest.OutputShow, Wifi: true},
			// Handled in Go (settings_events.go); the script path is never run.
			{ID: debugActionID, Label: c.message("drawer.settings_debug", nil), Run: debugActionID,
				Output: manifest.OutputShow},
			// The marker row of the App updates screen; expanded by uikit.ExpandAppRows, never run.
			{ID: uikit.AppRowsMarkerAction, Label: uikit.AppRowsMarkerAction, Run: uikit.AppRowsMarkerAction,
				Output: manifest.OutputShow},
		},
		Requests: []string{manifest.RequestRescan},
	}, nil
}

// updateLayout derives the onboard root from the host folder (<root>/.adds/koboapps).
func (c Config) updateLayout() update.Layout {
	return update.Layout{OnboardRoot: c.onboardRoot(), AppsDir: c.AppsDir}
}

func (c Config) onboardRoot() string { return filepath.Dir(filepath.Dir(c.HostDir)) }

// pendingUpdate returns the update the last check found, when it is newer than this build.
func (c Config) pendingUpdate() (update.Availability, bool) {
	found, hasCheck := update.ReadAvailability(c.updateLayout())
	return found, hasCheck && found.IsNewerThan(c.Version)
}

// pendingUpdateCount counts the waiting updates: the drawer's and every app's.
func (c Config) pendingUpdateCount() int {
	count := c.appUpdateCount()
	if c.hasDrawerUpdate() {
		count++
	}
	return count
}

func (c Config) hasDrawerUpdate() bool {
	_, hasPending := c.pendingUpdate()
	return hasPending
}

func (c Config) hasPendingUpdate() bool { return c.pendingUpdateCount() > 0 }

// settingsItem is the header gear of the drawer. A waiting update is its badge, which the
// mod draws as the black dot on the gear.
func (c Config) settingsItem() Item {
	item := c.hostItem(HostSettingsID, "drawer.settings_name", HostSettingsAct, "drawer.settings_open", "settings.png")
	item.Screen = true
	if count := c.pendingUpdateCount(); count > 0 {
		item.Badge = true
		item.Status = c.message("drawer.updates_status", nil)
		if count > 1 {
			item.Status = c.countMessage("appupdate.n_updates", count)
		}
	}
	return item
}
