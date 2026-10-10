package drawer

import (
	"path/filepath"

	"koboapps/internal/manifest"
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
			// Handled in Go (settings_events.go); the script path is never run.
			{ID: debugActionID, Label: c.message("drawer.settings_debug", nil), Run: debugActionID,
				Output: manifest.OutputShow},
		},
	}, nil
}

// updateLayout derives the onboard root from the host folder (<root>/.adds/koboapps).
func (c Config) updateLayout() update.Layout {
	return update.Layout{OnboardRoot: c.onboardRoot()}
}

func (c Config) onboardRoot() string { return filepath.Dir(filepath.Dir(c.HostDir)) }

// pendingUpdate returns the update the last check found, when it is newer than this build.
func (c Config) pendingUpdate() (update.Availability, bool) {
	found, hasCheck := update.ReadAvailability(c.updateLayout())
	return found, hasCheck && found.IsNewerThan(c.Version)
}

func (c Config) hasPendingUpdate() bool {
	_, hasPending := c.pendingUpdate()
	return hasPending
}

// settingsItem is the header gear of the drawer. A waiting update is its badge, which the
// mod draws as the black dot on the gear.
func (c Config) settingsItem() Item {
	item := c.hostItem(HostSettingsID, "drawer.settings_name", HostSettingsAct, "drawer.settings_open", "settings.png")
	item.Screen = true
	if c.hasPendingUpdate() {
		item.Badge = true
		item.Status = c.message("drawer.updates_status", nil)
	}
	return item
}
