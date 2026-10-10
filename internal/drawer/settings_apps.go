package drawer

import (
	"strconv"
	"strings"
	"time"

	"koboapps/internal/manifest"
	"koboapps/internal/uikit"
	"koboapps/internal/update"
)

const (
	appsCheckActionID     = "apps-check"
	appsUpdateAllActionID = "apps-update-all"
	updateAppActionStem   = "app_update_"
	installAppActionStem  = "app_install_"
	updateAppScript       = "../../bin/update-app.sh"
	updateAppsCheckScript = "../../bin/update-apps-check.sh"
	updateAppsAllScript   = "../../bin/update-apps-all.sh"
	arrowSeparator        = " → "
	captionSeparator      = " · "
	// rescanStateKey puts one rescan request in the state of the screen after an app job.
	rescanStateKey = "requests"
)

// appsView is the catalog compared with the device, as the App updates screen shows it.
type appsView struct {
	hasCatalog bool
	entries    []update.AppEntry
	updates    []update.AppEntry
	installed  []update.AppEntry
	available  []update.AppEntry // not installed, and allowed to show
}

// appsView reads the cached catalog and the installed apps. Hidden catalog apps (developer
// tools such as the UI kit) are offered for install only when the user switched them on in the
// hidden-apps settings; installed ones are listed as usual.
func (c Config) appsView() appsView {
	catalog, hasCatalog := update.ReadAppsCatalog(c.updateLayout())
	view := appsView{hasCatalog: hasCatalog}
	if !hasCatalog {
		return view
	}
	settings := c.loadHostSettings()
	view.entries = update.AppEntries(catalog, c.installedAppVersions(), c.Locale)
	for _, entry := range view.entries {
		switch {
		case entry.State == update.AppStateUpdate:
			view.updates = append(view.updates, entry)
		case entry.State == update.AppStateInstalled:
			view.installed = append(view.installed, entry)
		case !entry.IsHidden || settings.showsHiddenApp(entry.ID):
			view.available = append(view.available, entry)
		}
	}
	return view
}

// installedAppVersions reads the installed apps from this config's apps folder.
func (c Config) installedAppVersions() map[string]string {
	return update.InstalledAppVersions(c.updateLayout())
}

// appUpdateCount is the number of cached app updates, for the dot on the gear.
func (c Config) appUpdateCount() int {
	return len(c.appsView().updates)
}

func (c Config) countMessage(key string, count int) string {
	if count == 1 {
		return c.message(key+"_one", nil)
	}
	return c.message(key, map[string]string{"count": strconv.Itoa(count)})
}

// addAppsState fills the variables of the App updates screen and of its row in the settings.
func (c Config) addAppsState(state map[string]any) {
	view := c.appsView()
	job, hasJob := update.ReadAppsJob(c.updateLayout())
	status, hasStatus, _ := update.ReadStatus(c.updateLayout())
	isBusy := (hasJob && job.State == update.StateRunning) || (hasStatus && status.State == update.StateRunning)
	state["apps_caption"] = c.appsCaption(view)
	state["apps_status"] = ""
	if len(view.updates) > 0 {
		state["apps_status"] = c.countMessage("appupdate.n_updates", len(view.updates))
	}
	state["has_app_updates"] = len(view.updates) > 0
	state["apps_update_all_label"] = c.message("appupdate.update_all", map[string]string{"count": strconv.Itoa(len(view.updates))})
	state["can_apps_update"] = len(view.updates) > 0 && !isBusy
	state["can_apps_check"] = c.updateRepo() != "" && !isBusy
	state["apps_checked"] = c.appsCheckedText()
	state["apps_rows"] = c.appRows(view)
	c.addAppsProgress(state, job, hasJob)
	if state["apps_step_label"] != "" {
		state["apps_checked"] = "" // keeps the screen on one page while a job runs
	}
	if update.ConsumeRescanRequest(c.updateLayout()) {
		state[rescanStateKey] = []any{map[string]any{"request": manifest.RequestRescan, "mode": RescanModeQuick}}
	}
}

func (c Config) appsCaption(view appsView) string {
	if !view.hasCatalog {
		return c.message("appupdate.caption_unchecked", nil)
	}
	var parts []string
	if len(view.updates) > 0 {
		parts = append(parts, c.countMessage("appupdate.n_updates", len(view.updates)))
	} else {
		parts = append(parts, c.message("appupdate.caption_current", nil))
	}
	if len(view.available) > 0 {
		parts = append(parts, c.message("appupdate.n_not_installed", map[string]string{"count": strconv.Itoa(len(view.available))}))
	}
	return strings.Join(parts, captionSeparator)
}

func (c Config) appsCheckedText() string {
	record, hasRecord := update.ReadAppsCheck(c.updateLayout())
	if !hasRecord {
		return c.message("appupdate.never_checked", nil)
	}
	moment, err := time.Parse(time.RFC3339, record.Time)
	if err != nil {
		return ""
	}
	return c.message("appupdate.checked_at", map[string]string{"time": c.Bundle.DateTime(moment)})
}

// addAppsProgress splits the job like the drawer update does: a run in progress is the steps
// block, an outcome is the notice's text, kept until the next run.
func (c Config) addAppsProgress(state map[string]any, job update.AppsJob, hasJob bool) {
	for _, name := range []string{"apps_step_label", "apps_step_done", "apps_step_total", "apps_step_detail", "apps_message"} {
		state[name] = ""
	}
	switch {
	case hasJob && job.State == update.StateRunning:
		c.addRunningJob(state, job)
	case hasJob && !(job.State == update.StateDone && job.Job == update.AppsJobCheck):
		state["apps_message"] = c.renderAppsReport(job.Report)
	}
}

func (c Config) addRunningJob(state map[string]any, job update.AppsJob) {
	state["apps_step_label"] = c.message("appupdate.step_"+job.Job, nil)
	if job.Total > 1 {
		state["apps_step_done"], state["apps_step_total"] = strconv.Itoa(job.Done), strconv.Itoa(job.Total)
	}
	if job.AppID != "" && job.Phase != "" {
		state["apps_step_detail"] = c.message("appupdate.step_detail", map[string]string{
			"name": update.AppDisplayName(c.updateLayout(), job.AppID, c.Locale), "phase": c.message("appupdate.phase_"+job.Phase, nil)})
	}
}

// renderAppsReport is a report of the app updater with the app's name filled in.
func (c Config) renderAppsReport(result update.Report) string {
	named := result.WithName(func(id string) string { return update.AppDisplayName(c.updateLayout(), id, c.Locale) })
	return named.Render(c.Bundle)
}

// appRows lists the sections and rows of the screen, in the order updates, installed, not
// installed; an empty section is left out.
func (c Config) appRows(view appsView) []any {
	rows := []any{}
	if len(view.updates) > 0 {
		rows = append(rows, map[string]any{"kind": "section", "text": c.message("appupdate.sec_updates", nil)})
		for _, entry := range view.updates {
			rows = append(rows, c.tapRow(entry, updateAppActionStem, entry.Installed+arrowSeparator+entry.Latest))
		}
	}
	if len(view.installed) > 0 {
		rows = append(rows, map[string]any{"kind": "section", "text": c.message("appupdate.sec_installed", nil)})
		for _, entry := range view.installed {
			rows = append(rows, map[string]any{"kind": "kv", "key": entry.Name, "value": entry.Installed})
		}
	}
	if len(view.available) > 0 {
		rows = append(rows, map[string]any{"kind": "section", "text": c.message("appupdate.sec_not_installed", nil)})
		for _, entry := range view.available {
			rows = append(rows, c.tapRow(entry, installAppActionStem, entry.Latest))
		}
	}
	return rows
}

// tapRow is a row that asks first, then updates or installs the app. The question names the
// app and the version, so the confirm texts are filled in here, not in ui.json.
func (c Config) tapRow(entry update.AppEntry, actionStem, detail string) map[string]any {
	titleKey, textKey, okKey := "appupdate.confirm_update_title", "appupdate.confirm_update_text", "appupdate.ok_update"
	if actionStem == installAppActionStem {
		titleKey, textKey, okKey = "appupdate.confirm_install_title", "appupdate.confirm_install_text", "appupdate.ok_install"
	}
	params := map[string]string{"name": entry.Name, "version": entry.Latest}
	return map[string]any{"kind": "row", "text": entry.Name, "detail": detail, "action": actionStem + entry.ID,
		"confirm": map[string]any{"title": c.message(titleKey, params), "text": c.message(textKey, params),
			"ok": c.message(okKey, nil), "cancel": c.message("appupdate.cancel", nil)}}
}

// expandAppRows is the screen-side half of appRows (see uikit.ExpandAppRows).
func (s screenContext) expandAppRows(components []uikit.Component) []uikit.Component {
	return uikit.ExpandAppRows(components, s.vars)
}

// runAppsEvent starts the detached job a row of the App updates screen asks for. The row's
// action names the app; it must be one the screen offers right now, so a crafted call cannot
// install anything else. handled is false for events of other screens.
func (c Config) runAppsEvent(event Event) (result RunResult, handled bool, err error) {
	for _, stem := range []string{updateAppActionStem, installAppActionStem} {
		if appID, isApp := strings.CutPrefix(event.ActionID, stem); isApp {
			result, err = c.startAppJob(event, stem, appID)
			return result, true, err
		}
	}
	return RunResult{}, false, nil
}

func (c Config) startAppJob(event Event, stem, appID string) (RunResult, error) {
	if !c.offersApp(stem, appID) {
		return RunResult{}, newError(CodeUnknownApp, "app", appID)
	}
	host, err := c.hostApp(HostSettingsID)
	if err != nil {
		return RunResult{}, err
	}
	action := manifest.Action{ID: event.ActionID, Label: event.ActionID, Run: updateAppScript, Output: manifest.OutputShow, Wifi: true}
	result, err := c.withValue(&appID).RunAction(host.Dir, action)
	if err != nil {
		return RunResult{}, err
	}
	return c.withSignal(event, result, nil)
}

// offersApp is true when the App updates screen lists the app for that kind of action now.
func (c Config) offersApp(stem, appID string) bool {
	view := c.appsView()
	entries := view.updates
	if stem == installAppActionStem {
		entries = view.available
	}
	for _, entry := range entries {
		if entry.ID == appID {
			return true
		}
	}
	return false
}
