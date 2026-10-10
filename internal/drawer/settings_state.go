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
	newVersionMark  = "● "
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
		state["latest_value"] = newVersionMark + pending.Version
		state["apply_label"] = c.message("drawer.settings_apply_to", map[string]string{"version": pending.Version})
	}
	state["notes"] = pending.Notes
	state["message"] = c.updateMessage(status, hasStatus)
	state["can_update"] = hasSource && hasPending && !isBusy
	state["can_check"] = hasSource && !isBusy
}

// updateRepo is the owner/name that data/update.json names; empty switches updates off.
func (c Config) updateRepo() string {
	config, err := update.LoadConfig(c.updateLayout())
	if err != nil {
		return ""
	}
	return config.Repo
}

// updateMessage shows the last run when it is news: running, failed, or an install that
// needs a restart. A plain "up to date" needs no line of its own.
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
