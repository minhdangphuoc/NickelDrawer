// Package manifest loads and validates app.json files (format v1, see docs/spec/app-format.md).
package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"koboapps/internal/release"
	"koboapps/internal/storage"
)

const (
	SchemaVersion   = 1
	ManifestName    = "app.json"
	OutputShow      = "show"
	OutputNone      = "none"
	RescanNone      = "none"
	RescanQuick     = "quick"
	RescanFull      = "full"
	DefaultTimeout  = 5000
	minTimeout      = 500
	maxTimeout      = 9999 // NickelMenu rejects cmd_output timeouts of 10000 ms or more.
	maxActions      = 12
	maxNameLen      = 24
	maxLabelLen     = 32
	maxDescriptionN = 120

	// Host services a script may request through the drawer protocol (see
	// docs/spec/drawer-protocol.md); an app lists the ones it uses in app.json "requests".
	RequestRescan = "rescan"
	RequestWifi   = "wifi"
)

var (
	appIDPattern    = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,30}$`)
	actionIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,30}$`)
	semverPattern   = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

	knownAppFields    = []string{"schema", "id", "name", "version", "description", "menu", "hidden", "storage", "network", "requests", "min_host", "actions"}
	knownActionFields = []string{"id", "label", "run", "output", "timeout_ms", "wifi", "rescan", "menu", "confirm"}
)

// App is a parsed app.json. Use the methods for fields that have defaults.
type App struct {
	Schema      int            `json:"schema"`
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Version     string         `json:"version"`
	Description string         `json:"description"`
	Menu        *bool          `json:"menu"`
	Hidden      bool           `json:"hidden"`
	Storage     *StorageAccess `json:"storage"`
	Network     bool           `json:"network"`
	Requests    []string       `json:"requests"` // host services scripts may ask for: rescan, wifi
	MinHost     string         `json:"min_host"` // oldest NickelDrawer the app works with, MAJOR.MINOR.PATCH[-prerelease]
	Actions     []Action       `json:"actions"`
	Dir         string         `json:"-"` // absolute folder the manifest was loaded from
}

// StorageAccess declares which logical storage folders (see internal/storage) an app reads and
// writes. It is documentation that appsctl validate checks and reviews hold the code to.
type StorageAccess struct {
	Reads  []string `json:"reads"`
	Writes []string `json:"writes"`
}

// Action is one runnable entry of an app.
type Action struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Run       string `json:"run"`
	Output    string `json:"output"`
	TimeoutMS *int   `json:"timeout_ms"`
	Wifi      bool   `json:"wifi"`
	Rescan    string `json:"rescan"`
	Menu      *bool  `json:"menu"`
	Confirm   string `json:"confirm"` // question asked before the drawer runs the action; empty: none
}

// Warning is a non-fatal remark about a manifest.
type Warning string

// ValidationError lists every rule a manifest breaks, so authors fix them in one pass.
type ValidationError struct {
	Problems []string
}

func (e *ValidationError) Error() string {
	return strings.Join(e.Problems, "; ")
}

// IsHidden reports whether the app is kept out of every list (NickelMenu and the drawer). A
// hidden app stays validated and runnable by id, e.g. while its feature is postponed.
func (a App) IsHidden() bool {
	return a.Hidden
}

// InMenu reports whether the app appears in the menu (default true; never for a hidden app).
func (a App) InMenu() bool {
	return !a.Hidden && (a.Menu == nil || *a.Menu)
}

// InMenu reports whether the action appears in NickelMenu (default true). A hidden action stays
// valid and runnable; the native drawer shows it later.
func (a Action) InMenu() bool {
	return a.Menu == nil || *a.Menu
}

// InDrawerMenu reports whether the action is in the drawer's long-press menu. Unlike
// NickelMenu this is opt-in (`"menu": true`): most actions only make sense on the app's screen.
func (a Action) InDrawerMenu() bool {
	return a.Menu != nil && *a.Menu
}

// Timeout returns the cmd_output timeout in ms; 0 when the action has no dialog.
func (a Action) Timeout() int {
	if a.Output != OutputShow {
		return 0
	}
	if a.TimeoutMS == nil {
		return DefaultTimeout
	}
	return *a.TimeoutMS
}

// RescanMode returns the rescan setting with its default applied.
func (a Action) RescanMode() string {
	if a.Rescan == "" {
		return RescanNone
	}
	return a.Rescan
}

// Load reads and validates <appDir>/app.json.
// The returned error is a *ValidationError when the manifest is readable but breaks rules.
func Load(appDir string) (App, []Warning, error) {
	dir, err := filepath.Abs(appDir)
	if err != nil {
		return App{}, nil, fmt.Errorf("cannot resolve app folder %q: %w", appDir, err)
	}
	data, err := os.ReadFile(filepath.Join(dir, ManifestName))
	if err != nil {
		return App{}, nil, fmt.Errorf("cannot read %s in %s (every app folder needs one): %w", ManifestName, dir, err)
	}
	var app App
	if err := json.Unmarshal(data, &app); err != nil {
		return App{}, nil, &ValidationError{[]string{"app.json is not valid JSON for this format: " + err.Error()}}
	}
	app.Dir = dir
	warnings := unknownFieldWarnings(data)
	if problems := validateApp(app, dir); len(problems) > 0 {
		return App{}, warnings, &ValidationError{problems}
	}
	return app, warnings, nil
}

func validateApp(app App, dir string) []string {
	var problems []string
	add := func(format string, args ...any) { problems = append(problems, fmt.Sprintf(format, args...)) }

	if app.Schema != SchemaVersion {
		add("schema must be %d (got %d); add \"schema\": %d", SchemaVersion, app.Schema, SchemaVersion)
	}
	if !appIDPattern.MatchString(app.ID) {
		add("id %q must match ^[a-z0-9][a-z0-9-]{1,30}$ (lowercase letters, digits, dashes; 2-31 chars)", app.ID)
	}
	if folder := filepath.Base(dir); app.ID != folder {
		add("id %q must equal the folder name %q; rename the folder or change the id", app.ID, folder)
	}
	problems = append(problems, validateText("name", app.Name, 1, maxNameLen)...)
	if !semverPattern.MatchString(app.Version) {
		add("version %q must look like MAJOR.MINOR.PATCH or MAJOR.MINOR.PATCH-prerelease, e.g. 1.0.0 or 1.0.0-beta.8", app.Version)
	}
	if utf8.RuneCountInString(app.Description) > maxDescriptionN {
		add("description is longer than %d characters; shorten it", maxDescriptionN)
	}
	problems = append(problems, validateStorage(app.Storage)...)
	problems = append(problems, validateRequests(app.Requests)...)
	problems = append(problems, validateMinHost(app.MinHost)...)
	problems = append(problems, validateActions(app.Actions, dir)...)
	return problems
}

func validateStorage(access *StorageAccess) []string {
	if access == nil {
		return nil
	}
	var problems []string
	for field, names := range map[string][]string{"reads": access.Reads, "writes": access.Writes} {
		for _, name := range names {
			if !storage.IsKnown(name) {
				problems = append(problems, fmt.Sprintf("storage.%s: unknown folder %q; use one of %s",
					field, name, knownStorageNames()))
			}
		}
	}
	sort.Strings(problems)
	return problems
}

func validateRequests(requests []string) []string {
	var problems []string
	seen := map[string]bool{}
	for _, request := range requests {
		if request != RequestRescan && request != RequestWifi {
			problems = append(problems, fmt.Sprintf("requests: unknown request %q; use %q or %q", request, RequestRescan, RequestWifi))
		}
		if seen[request] {
			problems = append(problems, fmt.Sprintf("requests: %q is listed twice; remove the duplicate", request))
		}
		seen[request] = true
	}
	return problems
}

func validateMinHost(minHost string) []string {
	if minHost == "" || release.ValidVersion(minHost) {
		return nil
	}
	return []string{fmt.Sprintf("min_host %q must look like MAJOR.MINOR.PATCH, e.g. 1.0.0", minHost)}
}

// MayRequest reports whether the app declared the host service in "requests".
func (a App) MayRequest(request string) bool {
	return contains(a.Requests, request)
}

// NeedsNewerHost reports whether the app asks for a newer NickelDrawer than hostVersion. A
// host version that is not a release number (a development build) never blocks an app.
func (a App) NeedsNewerHost(hostVersion string) bool {
	if a.MinHost == "" || !release.ValidVersion(hostVersion) {
		return false
	}
	order, err := release.CompareVersions(hostVersion, a.MinHost)
	return err == nil && order < 0
}

func knownStorageNames() string {
	names := make([]string, 0)
	for _, folder := range storage.Known() {
		names = append(names, string(folder))
	}
	return strings.Join(names, ", ")
}

// validateText checks length in characters and the NickelMenu field separator ':'.
func validateText(field, value string, minLen, maxLen int) []string {
	var problems []string
	if n := utf8.RuneCountInString(value); n < minLen || n > maxLen {
		problems = append(problems, fmt.Sprintf("%s must be %d-%d characters (got %d)", field, minLen, maxLen, n))
	}
	if strings.Contains(value, ":") {
		problems = append(problems, fmt.Sprintf("%s %q must not contain ':' (NickelMenu field separator); remove it", field, value))
	}
	return problems
}

func validateActions(actions []Action, dir string) []string {
	if len(actions) < 1 || len(actions) > maxActions {
		return []string{fmt.Sprintf("actions must have 1-%d items (got %d)", maxActions, len(actions))}
	}
	var problems []string
	seen := map[string]bool{}
	for i, action := range actions {
		where := fmt.Sprintf("actions[%d]", i)
		if seen[action.ID] {
			problems = append(problems, fmt.Sprintf("%s: duplicate action id %q; ids must be unique within the app", where, action.ID))
		}
		seen[action.ID] = true
		for _, problem := range validateAction(action, dir) {
			problems = append(problems, where+": "+problem)
		}
	}
	return problems
}

func validateAction(action Action, dir string) []string {
	var problems []string
	add := func(format string, args ...any) { problems = append(problems, fmt.Sprintf(format, args...)) }

	if !actionIDPattern.MatchString(action.ID) {
		add("id %q must match ^[a-z0-9][a-z0-9-]{0,30}$", action.ID)
	}
	problems = append(problems, validateText("label", action.Label, 1, maxLabelLen)...)
	if problem := validateRunPath(action.Run, dir); problem != "" {
		problems = append(problems, problem)
	}
	if action.Confirm != "" {
		problems = append(problems, validateText("confirm", action.Confirm, 1, maxDescriptionN)...)
	}
	problems = append(problems, validateOutput(action)...)
	switch action.RescanMode() {
	case RescanNone, RescanQuick, RescanFull:
	default:
		add("rescan %q must be one of none, quick, full", action.Rescan)
	}
	return problems
}

func validateOutput(action Action) []string {
	switch action.Output {
	case OutputShow:
		if t := action.TimeoutMS; t != nil && (*t < minTimeout || *t > maxTimeout) {
			return []string{fmt.Sprintf("timeout_ms %d must be %d-%d (NickelMenu limit is below 10000)", *t, minTimeout, maxTimeout)}
		}
	case OutputNone:
		if action.TimeoutMS != nil {
			return []string{"timeout_ms is only allowed with output \"show\"; remove it"}
		}
	default:
		return []string{fmt.Sprintf("output %q must be \"show\" or \"none\"", action.Output)}
	}
	return nil
}

// validateRunPath returns "" when run is a safe, existing, executable file inside dir.
func validateRunPath(run, dir string) string {
	if run == "" {
		return "run is empty; give a path such as bin/start.sh"
	}
	if strings.HasPrefix(run, "/") || strings.Contains(run, ":") || hasParentSegment(run) {
		return fmt.Sprintf("run %q must be relative to the app folder, without leading '/', '..' or ':'", run)
	}
	info, err := os.Stat(filepath.Join(dir, filepath.FromSlash(run)))
	if err != nil {
		return fmt.Sprintf("run file %q does not exist; create it or fix the path", run)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		return fmt.Sprintf("run file %q is not an executable file; run chmod +x on it", run)
	}
	return ""
}

func hasParentSegment(path string) bool {
	for _, segment := range strings.Split(path, "/") {
		if segment == ".." {
			return true
		}
	}
	return false
}

// unknownFieldWarnings finds fields this version does not know, for forward compatibility.
func unknownFieldWarnings(data []byte) []Warning {
	var raw struct {
		Actions []map[string]json.RawMessage `json:"actions"`
	}
	var top map[string]json.RawMessage
	if json.Unmarshal(data, &top) != nil || json.Unmarshal(data, &raw) != nil {
		return nil
	}
	var warnings []Warning
	for _, name := range unknownKeys(top, knownAppFields) {
		warnings = append(warnings, Warning(fmt.Sprintf("unknown field %q ignored", name)))
	}
	for i, action := range raw.Actions {
		for _, name := range unknownKeys(action, knownActionFields) {
			warnings = append(warnings, Warning(fmt.Sprintf("actions[%d]: unknown field %q ignored", i, name)))
		}
	}
	return warnings
}

func unknownKeys(fields map[string]json.RawMessage, known []string) []string {
	var unknown []string
	for name := range fields {
		if !contains(known, name) {
			unknown = append(unknown, name)
		}
	}
	sort.Strings(unknown)
	return unknown
}

func contains(list []string, item string) bool {
	for _, entry := range list {
		if entry == item {
			return true
		}
	}
	return false
}
