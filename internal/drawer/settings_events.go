package drawer

import (
	"os"
	"path/filepath"
	"strings"

	"koboapps/internal/manifest"
)

// runSettingsEvent applies a toggle of the settings screen. handled is false for the actions
// that run host scripts (check, apply); those go through the normal action path.
func (c Config) runSettingsEvent(event Event) (result RunResult, handled bool, err error) {
	switch {
	case event.ActionID == debugActionID:
		return c.settingsDone(c.setDebugFlag(event.Value))
	case strings.HasPrefix(event.ActionID, showHiddenActionStem):
		appID := strings.TrimPrefix(event.ActionID, showHiddenActionStem)
		return c.settingsDone(c.setHiddenAppShown(appID, event.Value))
	}
	return RunResult{}, false, nil
}

func (c Config) settingsDone(err error) (RunResult, bool, error) {
	if err != nil {
		return RunResult{}, true, err
	}
	return RunResult{V: ProtocolVersion, OK: true, Rescan: manifest.RescanNone}, true, nil
}

// toggleValue reads the "true"/"false" a toggle event carries.
func toggleValue(actionID string, value *string) (bool, error) {
	if value != nil {
		switch *value {
		case "true":
			return true, nil
		case "false":
			return false, nil
		}
	}
	return false, newError(CodeUsage, "action", actionID)
}

func (c Config) setDebugFlag(value *string) error {
	isOn, err := toggleValue(debugActionID, value)
	if err != nil {
		return err
	}
	flag := c.debugFlagPath()
	if !isOn {
		if err := os.Remove(flag); err != nil && !os.IsNotExist(err) {
			return newError(CodeIO, "detail", err.Error())
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(flag), 0o755); err != nil {
		return newError(CodeIO, "detail", err.Error())
	}
	if err := os.WriteFile(flag, nil, 0o644); err != nil {
		return newError(CodeIO, "detail", err.Error())
	}
	return nil
}

// setHiddenAppShown stores the choice for an installed app whose manifest says hidden; any
// other id is refused, so the file only ever names apps the screen offered.
func (c Config) setHiddenAppShown(appID string, value *string) error {
	isShown, err := toggleValue(showHiddenActionStem+appID, value)
	if err != nil {
		return err
	}
	if !c.isInstalledHiddenApp(appID) {
		return newError(CodeUnknownApp, "app", appID)
	}
	updated := c.loadHostSettings().withHiddenApp(appID, isShown)
	if err := c.saveHostSettings(updated); err != nil {
		return newError(CodeIO, "detail", err.Error())
	}
	return nil
}

func (c Config) isInstalledHiddenApp(appID string) bool {
	for _, row := range c.hiddenAppRows() {
		if row.appID == appID {
			return true
		}
	}
	return false
}
