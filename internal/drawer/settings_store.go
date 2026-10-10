package drawer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"koboapps/internal/manifest"
	"koboapps/internal/update"
)

const settingsFileName = "settings.json"

// hostSettings is data/settings.json: the user's choices about the drawer itself. It lives in
// the host data folder, not in app.json, so reinstalling or updating an app keeps them.
type hostSettings struct {
	// ShownHiddenApps lists apps whose manifest says hidden:true that the user wants in the drawer.
	ShownHiddenApps []string `json:"shown_hidden_apps"`
}

func (c Config) settingsPath() string { return filepath.Join(c.HostDir, "data", settingsFileName) }

// loadHostSettings reads settings.json. A missing file is the defaults; a damaged one is the
// defaults too (and is logged), because the drawer must still open.
func (c Config) loadHostSettings() hostSettings {
	data, err := os.ReadFile(c.settingsPath())
	if err != nil {
		if !os.IsNotExist(err) {
			c.debugf("cannot read %s: %v", c.settingsPath(), err)
		}
		return hostSettings{}
	}
	var settings hostSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		c.debugf("%s is not valid JSON, using the defaults: %v", c.settingsPath(), err)
		return hostSettings{}
	}
	return settings
}

func (c Config) saveHostSettings(settings hostSettings) error {
	if settings.ShownHiddenApps == nil {
		settings.ShownHiddenApps = []string{}
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return update.WriteFileAtomically(c.settingsPath(), append(data, '\n'), 0o644)
}

func (s hostSettings) showsHiddenApp(appID string) bool {
	for _, shown := range s.ShownHiddenApps {
		if shown == appID {
			return true
		}
	}
	return false
}

// withHiddenApp returns the settings with appID added to or removed from the shown list.
func (s hostSettings) withHiddenApp(appID string, isShown bool) hostSettings {
	kept := make([]string, 0, len(s.ShownHiddenApps)+1)
	for _, shown := range s.ShownHiddenApps {
		if shown != appID {
			kept = append(kept, shown)
		}
	}
	if isShown {
		kept = append(kept, appID)
	}
	sort.Strings(kept)
	return hostSettings{ShownHiddenApps: kept}
}

// shownInDrawer keeps the apps the drawer lists: every visible app, plus hidden ones the
// user switched on in the settings.
func shownInDrawer(apps []manifest.App, settings hostSettings) []manifest.App {
	shown := make([]manifest.App, 0, len(apps))
	for _, app := range apps {
		if !app.IsHidden() || settings.showsHiddenApp(app.ID) {
			shown = append(shown, app)
		}
	}
	return shown
}

// hiddenApps returns the installed apps whose manifest says hidden:true.
func hiddenApps(apps []manifest.App) []manifest.App {
	var hidden []manifest.App
	for _, app := range apps {
		if app.IsHidden() {
			hidden = append(hidden, app)
		}
	}
	return hidden
}
