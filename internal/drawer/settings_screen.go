package drawer

import (
	"koboapps/internal/manifest"
	"koboapps/internal/uikit"
)

const (
	// hiddenAppTemplateBind marks the toggle in the settings ui.json that stands for "one
	// toggle per hidden app"; the caption text right after it is the template of the hint line.
	hiddenAppTemplateBind = "hidden_app"
	showHiddenActionStem  = "show_hidden_"
)

// hiddenAppRow is one toggle of the hidden-apps section.
type hiddenAppRow struct {
	appID   string
	name    string
	isShown bool
	hint    string // "Hidden · for developers" or "Shown in the drawer"
}

// hiddenAppRows lists the installed hidden apps in the device language, for the settings screen.
func (c Config) hiddenAppRows() []hiddenAppRow {
	scan, err := manifest.LoadAll(c.AppsDir)
	if err != nil {
		c.debugf("settings: cannot list apps: %v", err)
		return nil
	}
	settings := c.loadHostSettings()
	var rows []hiddenAppRow
	for _, app := range hiddenApps(scan.Apps) {
		localized := app
		if bundle, _ := app.LoadLocales(c.Locale); bundle != nil {
			localized = app.Localize(bundle)
		}
		isShown := settings.showsHiddenApp(app.ID)
		rows = append(rows, hiddenAppRow{appID: app.ID, name: localized.Name, isShown: isShown, hint: c.hiddenAppHint(isShown)})
	}
	return rows
}

func (c Config) hiddenAppHint(isShown bool) string {
	if isShown {
		return c.message("drawer.settings_hint_shown", nil)
	}
	return c.message("drawer.settings_hint_hidden", nil)
}

func (c Config) addHiddenAppsState(state map[string]any) {
	state["hidden_none"] = ""
	if len(c.hiddenAppRows()) == 0 {
		state["hidden_none"] = c.message("drawer.settings_hidden_none", nil)
	}
}

func isHiddenAppTemplate(component uikit.Component) bool {
	return component.Kind == "toggle" && component.Props["bind"] == hiddenAppTemplateBind
}

// hiddenAppComponents expands the template toggle: a toggle and a hint line per hidden app.
func (s screenContext) hiddenAppComponents() []ScreenComponent {
	resolved := make([]ScreenComponent, 0, 2*len(s.hiddenApps))
	for _, row := range s.hiddenApps {
		action := showHiddenActionStem + row.appID
		isEnabled := true
		resolved = append(resolved,
			ScreenComponent{Type: "toggle", Text: row.name, Bind: action, Value: row.isShown, Enabled: &isEnabled, Action: action},
			ScreenComponent{Type: "text", Text: row.hint, Style: "caption"})
	}
	return resolved
}
