package drawer

import (
	"os"
	"path/filepath"
	"strings"

	"koboapps/internal/device"
	"koboapps/internal/project"
	"koboapps/internal/update"
)

const (
	checkMark       = "✓"
	warningMark     = "!"
	versionFileName = "version"
)

// hostState is the state of the built-in settings screen. It reads the files `appsctl update`
// writes and the host data folder, so the screen never touches the network.
func (c Config) hostState(string) map[string]any {
	state := map[string]any{
		"current": c.Version,
		"debug":   c.hasDebugFlag(),
	}
	c.addUpdateState(state)
	c.addProjectState(state)
	c.addHiddenAppsState(state)
	return state
}

func (c Config) addUpdateState(state map[string]any) {
	status, hasStatus, _ := update.ReadStatus(c.updateLayout())
	pending, hasPending := c.pendingUpdate()
	source := c.updateRepo()
	isBusy := hasStatus && status.State == update.StateRunning
	hasSource := source != ""
	state["source"] = source
	if !hasSource {
		state["source"] = c.message("drawer.settings_source_off", nil)
	}
	state["latest"] = ""
	state["latest_label"] = ""
	state["latest_value"] = ""
	state["apply_label"] = c.message("drawer.settings_apply", nil)
	if hasPending {
		state["latest"] = pending.Version
		state["latest_label"] = c.message("drawer.settings_latest", nil)
		state["latest_value"] = pending.Version
		state["apply_label"] = c.message("drawer.settings_apply_to", map[string]string{"version": pending.Version})
	}
	state["notes"] = pending.Notes
	state["has_update"] = hasSource && hasPending
	state["can_update"] = hasSource && hasPending && !isBusy
	state["can_check"] = hasSource && !isBusy
	c.addProgressState(state, status, hasStatus)
}

// addProgressState splits the update line: a run in progress is the label of the screen's steps
// block (no count: a check or an install has none), an outcome is the notice's text.
func (c Config) addProgressState(state map[string]any, status update.Status, hasStatus bool) {
	state["step_label"], state["message"] = "", ""
	if hasStatus && status.State == update.StateRunning {
		state["step_label"] = status.Report.Render(c.Bundle)
		return
	}
	state["message"] = c.updateMessage(status, hasStatus)
}

// updateRepo is the owner/name that data/update.json names; empty switches updates off.
func (c Config) updateRepo() string {
	config, err := update.LoadConfig(c.updateLayout())
	if err != nil {
		return ""
	}
	return config.Repo
}

// updateMessage is the outcome line of the Updates section: the newer of the last install
// result and the last check outcome. Both stay until the next run; a run in progress is shown
// by the steps block instead (addProgressState).
func (c Config) updateMessage(status update.Status, hasStatus bool) string {
	check, hasCheck := update.ReadLastCheck(c.updateLayout())
	isInstallNews := hasStatus && status.Job != update.JobCheck && isInstallResult(status)
	if isInstallNews && (!hasCheck || status.Time > check.Time) {
		return status.Report.Render(c.Bundle)
	}
	if hasCheck {
		return check.Describe(c.Bundle, c.Version)
	}
	return ""
}

// isInstallResult is true for the outcomes worth showing: a failure, an install (which may
// ask for a restart) or a rollback. A plain "up to date" from an install run is not news.
func isInstallResult(status update.Status) bool {
	switch {
	case status.State == update.StateFailed:
		return true
	case status.Report.Key == "update.applied", status.Report.Key == "update.applied_reboot",
		status.Report.Key == "update.rolled_back":
		return true
	}
	return false
}

func (c Config) addProjectState(state map[string]any) {
	info, err := project.Load(project.Path(c.HostDir))
	if err != nil {
		c.debugf("%v", err)
	}
	state["version_text"] = c.message("drawer.about_version", map[string]string{"version": c.Version})
	state["project_name"] = info.Name
	state["maintainer"] = info.Maintainer
	state["license"] = info.License
	state["project_url"] = ""
	if err == nil {
		state["project_url"] = info.URL()
	}
	state["project_repo"] = info.Repo
	state["firmware_range"] = info.FirmwareRange()
	state["firmware"] = c.firmwareText(info)
}

// firmwareText is the installed firmware with a check mark when the mod supports it and "!"
// when it does not (the state is text, never colour alone).
func (c Config) firmwareText(info project.Info) string {
	installed, err := device.Read(filepath.Join(c.onboardRoot(), ".kobo", versionFileName))
	if err != nil {
		return c.message("drawer.settings_firmware_unknown", nil)
	}
	mark := warningMark
	if info.SupportsFirmware(installed.Firmware) {
		mark = checkMark
	}
	return strings.TrimSpace(installed.Firmware + " " + mark)
}

func (c Config) debugFlagPath() string {
	return filepath.Join(c.onboardRoot(), ".adds", "nickeldrawer", "debug")
}

func (c Config) hasDebugFlag() bool {
	_, err := os.Stat(c.debugFlagPath())
	return err == nil
}
