package drawer

import (
	"os"
	"path/filepath"
	"sort"

	"koboapps/internal/manifest"
	"koboapps/internal/uikit"
)

const (
	SectionInstalled = "installed"
	SectionSystem    = "system"

	KindShow       = "show"
	KindBackground = "background"

	HostRefreshID   = "_refresh"
	HostSettingsID  = "_settings"
	HostRefreshAct  = "refresh"
	HostSettingsAct = "open"

	iconFileName = "icon.png"
	hostIconsDir = "icons"
)

// Model is the whole drawer screen.
type Model struct {
	V        int          `json:"v"`
	Title    string       `json:"title"`
	Hint     string       `json:"hint"`
	Sections []Section    `json:"sections"`
	Errors   []AppProblem `json:"errors"`
	// UpdateAvailable is true when a checked update is waiting (the mod may mark the More row).
	UpdateAvailable bool `json:"update_available"`
}

type Section struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Items []Item `json:"items"`
}

type Item struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Status  string   `json:"status"`
	Icon    string   `json:"icon,omitempty"`
	Badge   bool     `json:"badge"`
	Running *bool    `json:"running,omitempty"` // apps only
	Screen  bool     `json:"screen,omitempty"`  // the app has a ui.json: a tap opens its screen
	Actions []Action `json:"actions"`
}

// Action is a row of the long-press menu; the primary one runs on tap.
type Action struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Kind    string `json:"kind"`
	Primary bool   `json:"primary,omitempty"`
	// Menu is true only for actions the app marked `"menu": true`: the long-press menu is
	// opt-in so that screen-only actions (which need a selection or a value) never appear.
	Menu bool `json:"menu"`
	// Confirm is the localized question the mod asks before running the action; empty runs it at once.
	Confirm string `json:"confirm,omitempty"`
}

type AppProblem struct {
	App     string `json:"app"`
	Message string `json:"message"`
}

// BuildModel scans the apps folder and assembles the drawer screen. It fails only when the
// apps folder cannot be read.
func (c Config) BuildModel() (Model, error) {
	scan, err := manifest.LoadAll(c.AppsDir)
	if err != nil {
		return Model{}, newError(CodeIO, "detail", err.Error())
	}
	shownApps := shownInDrawer(scan.Apps, c.loadHostSettings())
	statuses := c.collectStatuses(shownApps)
	items := make([]Item, 0, len(shownApps))
	for _, app := range shownApps {
		items = append(items, c.appItem(app, statuses[app.ID]))
	}
	sortItems(items)
	problems := make([]AppProblem, 0, len(scan.Failures))
	for _, failure := range scan.Failures {
		problems = append(problems, AppProblem{App: failure.AppID, Message: failure.Reason()})
	}
	return Model{
		V:     ProtocolVersion,
		Title: c.message("drawer.title", nil),
		Hint:  c.message("drawer.hint", nil),
		Sections: []Section{
			{ID: SectionInstalled, Title: c.message("drawer.section.installed", nil), Items: items},
			{ID: SectionSystem, Title: c.message("drawer.section.system", nil), Items: c.hostItems()},
		},
		Errors:          problems,
		UpdateAvailable: c.hasPendingUpdate(),
	}, nil
}

func sortItems(items []Item) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Name != items[j].Name {
			return items[i].Name < items[j].Name
		}
		return items[i].ID < items[j].ID
	})
}

func (c Config) appItem(app manifest.App, status Status) Item {
	localizedApp := app
	if bundle, err := app.LoadLocales(c.Locale); bundle != nil {
		_ = err // a broken locale file only means English text; gen-menu reports it
		localizedApp = app.Localize(bundle)
	}
	running := status.Running
	return Item{
		ID:      app.ID,
		Name:    localizedApp.Name,
		Status:  c.appStatusLine(app, status),
		Icon:    existingFile(filepath.Join(app.Dir, iconFileName)),
		Running: &running,
		Screen:  uikit.HasUI(app.Dir),
		Actions: modelActions(localizedApp.Actions),
	}
}

// appStatusLine says why an app cannot work on this host; otherwise the app's own status line.
func (c Config) appStatusLine(app manifest.App, status Status) string {
	if app.NeedsNewerHost(c.Version) {
		return c.message("drawer.needs_newer", nil)
	}
	return status.Line
}

// existingFile returns path when it is a regular file, else "" (the drawer then draws no icon).
func existingFile(path string) string {
	if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
		return ""
	}
	return path
}

// modelActions marks the first action primary: manifest order is the author's priority, and
// an explicit marker would be a manifest change nobody has needed yet.
func modelActions(actions []manifest.Action) []Action {
	result := make([]Action, 0, len(actions))
	for i, action := range actions {
		kind := KindShow
		if action.Output == manifest.OutputNone {
			kind = KindBackground
		}
		result = append(result, Action{
			ID: action.ID, Label: action.Label, Kind: kind, Primary: i == 0,
			Menu: action.InDrawerMenu(), Confirm: action.Confirm,
		})
	}
	return result
}

func (c Config) hostItems() []Item {
	return []Item{
		c.hostItem(HostRefreshID, "drawer.refresh_name", HostRefreshAct, "drawer.refresh_action", "refresh.png"),
		c.settingsItem(),
	}
}

func (c Config) hostItem(id, nameKey, actionID, labelKey, iconName string) Item {
	return Item{
		ID:      id,
		Name:    c.message(nameKey, nil),
		Icon:    existingFile(filepath.Join(c.HostDir, hostIconsDir, iconName)),
		Actions: []Action{{ID: actionID, Label: c.message(labelKey, nil), Kind: KindShow, Primary: true}},
	}
}

// IsHostID reports whether id names a built-in item rather than an app.
func IsHostID(id string) bool { return len(id) > 0 && id[0] == '_' }

// Info is the long-press Info dialog.
type Info struct {
	V           int         `json:"v"`
	Name        string      `json:"name"`
	Version     string      `json:"version"`
	Description string      `json:"description"`
	Storage     InfoStorage `json:"storage"`
	Network     bool        `json:"network"`
	Path        string      `json:"path"`
}

type InfoStorage struct {
	Reads  []string `json:"reads"`
	Writes []string `json:"writes"`
}

// AppInfo describes one installed app in the device language.
func (c Config) AppInfo(appID string) (Info, error) {
	if IsHostID(appID) {
		return Info{}, newError(CodeHostItem, "app", appID)
	}
	app, err := c.loadApp(appID)
	if err != nil {
		return Info{}, err
	}
	if bundle, _ := app.LoadLocales(c.Locale); bundle != nil {
		app = app.Localize(bundle)
	}
	info := Info{
		V: ProtocolVersion, Name: app.Name, Version: app.Version, Description: app.Description,
		Storage: InfoStorage{Reads: []string{}, Writes: []string{}}, Network: app.Network, Path: app.Dir,
	}
	if app.Storage != nil {
		info.Storage.Reads = append(info.Storage.Reads, app.Storage.Reads...)
		info.Storage.Writes = append(info.Storage.Writes, app.Storage.Writes...)
	}
	return info, nil
}

// loadApp validates the id before it touches the file system, then loads the manifest.
func (c Config) loadApp(appID string) (manifest.App, error) {
	if IsHostID(appID) {
		return c.hostApp(appID)
	}
	if !isPlainName(appID) {
		return manifest.App{}, newError(CodeUnknownApp, "app", appID)
	}
	dir := filepath.Join(c.AppsDir, appID)
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return manifest.App{}, newError(CodeUnknownApp, "app", appID)
	}
	app, _, err := manifest.Load(dir)
	if err != nil {
		return manifest.App{}, newError(CodeInvalidApp, "app", appID, "detail", err.Error())
	}
	return app, nil
}

// isPlainName is true for a single folder name: no separators, no dot names.
func isPlainName(name string) bool {
	return name != "" && name != "." && name != ".." && name == filepath.Base(name) && name[0] != '.'
}
