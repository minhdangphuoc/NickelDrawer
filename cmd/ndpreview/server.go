package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"koboapps/internal/i18n"
	"koboapps/internal/manifest"
	"koboapps/internal/uikit"
)

const (
	loopbackHost  = "127.0.0.1"
	defaultPort   = 8790
	defaultScale  = 50
	maxBodyBytes  = 64 << 10
	greyLevelsOff = 0
	greyLevelsMax = 16
)

type serverConfig struct {
	AppDir      string
	Locale      string
	StatePath   string
	ExecEnabled bool
}

// server serves the preview page. It re-reads the app on every page load, so editing
// ui.json and refreshing the browser is the whole workflow.
type server struct {
	config    serverConfig
	mu        sync.Mutex
	overrides map[string]any
	runner    *actionRunner
}

func newServer(config serverConfig) (*server, error) {
	overrides, err := readStateFile(config.StatePath)
	if err != nil {
		return nil, err
	}
	instance := &server{config: config, overrides: overrides}
	if config.ExecEnabled {
		runner, runnerErr := newActionRunner()
		if runnerErr != nil {
			return nil, runnerErr
		}
		instance.runner = runner
	}
	return instance, nil
}

func (s *server) close() {
	if s.runner != nil {
		s.runner.close()
	}
}

func readStateFile(path string) (map[string]any, error) {
	state := map[string]any{}
	if path == "" {
		return state, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read the state file %s: %w", path, err)
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("the state file %s must be a JSON object of variable values: %w", path, err)
	}
	return state, nil
}

// listenLoopback opens the only listener ndpreview ever uses: loopback, never a public address.
func listenLoopback(port int) (net.Listener, error) {
	listener, err := net.Listen("tcp", net.JoinHostPort(loopbackHost, strconv.Itoa(port)))
	if err != nil {
		return nil, fmt.Errorf("cannot listen on %s:%d (is another ndpreview running? try --port): %w", loopbackHost, port, err)
	}
	return listener, nil
}

func (s *server) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.handlePage)
	mux.HandleFunc("PUT /api/state", s.handleSetState)
	mux.HandleFunc("POST /api/event", s.handleEvent)
	return guardHost(mux)
}

// guardHost refuses requests whose Host header is not loopback, so a web page on another
// site cannot reach the preview (or --exec) through DNS rebinding.
func guardHost(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			host = r.Host
		}
		if host != loopbackHost && host != "localhost" {
			http.Error(w, "ndpreview answers only on 127.0.0.1", http.StatusForbidden)
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" && !strings.Contains(origin, "//"+r.Host) {
			http.Error(w, "cross-origin requests are not allowed", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// loadedApp is the app on disk at the moment of a request.
type loadedApp struct {
	app      manifest.App
	doc      *uikit.Document
	warnings []string
	locales  []string
}

// loadApp reads app.json and the ui.json; problems come back as one readable error.
func (s *server) loadApp() (loadedApp, error) { return loadApp(s.config.AppDir, s.config.Locale) }

func loadApp(appDir, locale string) (loadedApp, error) {
	app, _, err := manifest.Load(appDir)
	if err != nil {
		return loadedApp{}, fmt.Errorf("app.json is not valid: %w", err)
	}
	ids := make([]string, len(app.Actions))
	for i, action := range app.Actions {
		ids[i] = action.ID
	}
	if !uikit.HasUI(app.Dir) {
		return loadedApp{}, fmt.Errorf("%s has no ui.json; create one (see docs/spec/ui-json.md)", app.Dir)
	}
	doc, report, err := uikit.LoadDir(app.Dir, ids)
	if err != nil {
		return loadedApp{}, err
	}
	if !report.OK() {
		return loadedApp{}, fmt.Errorf("ui.json is invalid:\n%s", joinIssueLines(report.Errors))
	}
	loaded := loadedApp{app: app, doc: doc, locales: localeNames(app.Dir, locale)}
	for _, warning := range report.Warnings {
		loaded.warnings = append(loaded.warnings, warning.String())
	}
	return loaded, nil
}

func joinIssueLines(issues []uikit.Issue) string {
	lines := make([]string, len(issues))
	for i, issue := range issues {
		lines[i] = issue.String()
	}
	return strings.Join(lines, "\n")
}

func localeNames(appDir, requested string) []string {
	names := map[string]bool{i18n.SourceLocale: true, requested: true}
	entries, _ := os.ReadDir(filepath.Join(appDir, "locales"))
	for _, entry := range entries {
		if name, isJSON := strings.CutSuffix(entry.Name(), ".json"); isJSON {
			names[name] = true
		}
	}
	list := make([]string, 0, len(names))
	for name := range names {
		list = append(list, name)
	}
	sort.Strings(list)
	return list
}

func (s *server) currentVars(doc *uikit.Document) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	vars := map[string]any{}
	for name, value := range doc.Vars {
		vars[name] = value
	}
	for name, value := range s.overrides {
		vars[name] = value
	}
	return vars
}

func (s *server) parseView(r *http.Request, doc *uikit.Document) viewOptions {
	query := r.URL.Query()
	view := viewOptions{
		Screen:  firstNonEmpty(query.Get("screen"), doc.Start),
		Locale:  firstNonEmpty(query.Get("locale"), s.config.Locale),
		Scale:   defaultScale,
		Toast:   query.Get("toast"),
		Dialog:  query.Get("dialog"),
		Confirm: query.Get("confirm"),
	}
	view.Page, _ = strconv.Atoi(query.Get("page"))
	if query.Get("scale") == "100" {
		view.Scale = 100
	}
	if query.Get("grey") == strconv.Itoa(greyLevelsMax) {
		view.Grey = greyLevelsMax
	}
	return view
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func (s *server) handlePage(w http.ResponseWriter, r *http.Request) {
	loaded, err := s.loadApp()
	if err != nil {
		writeErrorPage(w, err)
		return
	}
	view := s.parseView(r, loaded.doc)
	vars := s.currentVars(loaded.doc)
	rc := renderContext{doc: loaded.doc, locale: view.Locale, vars: vars}
	screen, found := loaded.doc.Screens[view.Screen]
	if !found {
		writeErrorPage(w, fmt.Errorf("screen %q does not exist; screens: %s", view.Screen, strings.Join(loaded.doc.ScreenIDs(), ", ")))
		return
	}
	page, err := layoutScreen(rc, view.Screen, view.Page, rc.overlayFor(screen, view))
	if err != nil {
		writeErrorPage(w, err)
		return
	}
	view.Page = page.Index
	stateJSON, _ := json.MarshalIndent(vars, "", "  ")
	side := sidebarData{
		AppName: loaded.app.Name, ScreenIDs: loaded.doc.ScreenIDs(), Locales: loaded.locales, StateJSON: string(stateJSON),
		Warnings: loaded.warnings, ToastIDs: sortedMapKeys(screen.Toasts), DialogIDs: sortedMapKeys(screen.Dialogs), ExecActive: s.runner != nil,
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	io.WriteString(w, renderPage(view, page, side))
}

func sortedMapKeys[V any](items map[string]V) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// overlayFor resolves the toast or dialog the query asks for.
func (rc renderContext) overlayFor(screen *uikit.Screen, view viewOptions) Overlay {
	var overlay Overlay
	if toast, has := screen.Toasts[view.Toast]; has {
		overlay.Toast = rc.text(toast)
	}
	if dialog, has := screen.Dialogs[view.Dialog]; has {
		overlay.Dialog = rc.dialogSpec(dialog["title"], dialog["text"], dialog["ok"], dialog["cancel"], dialog["action"])
	}
	if view.Confirm != "" {
		overlay.Dialog = rc.confirmDialog(screen, view.Confirm)
	}
	return overlay
}

func (rc renderContext) dialogSpec(title, body, ok, cancel, action any) *DialogSpec {
	spec := &DialogSpec{Title: rc.text(title), Text: rc.text(body), Ok: rc.text(ok), Cancel: rc.text(cancel)}
	if spec.Cancel == "" {
		spec.Cancel = rc.host("cancel")
	}
	if name, isString := action.(string); isString {
		spec.OkEvent = actionEvent(name)
	}
	return spec
}

// confirmDialog builds the dialog of a button with a "confirm" prop, found by its path.
func (rc renderContext) confirmDialog(screen *uikit.Screen, path string) *DialogSpec {
	for _, component := range append(append([]uikit.Component{}, screen.Body...), screen.Footer...) {
		confirm, isObject := component.Prop("confirm").(map[string]any)
		if component.Path != path || !isObject {
			continue
		}
		title := firstPresentValue(confirm["title"], component.Main())
		return rc.dialogSpec(title, confirm["text"], confirm["ok"], confirm["cancel"], component.Prop("action"))
	}
	return nil
}

func firstPresentValue(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func writeErrorPage(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusUnprocessableEntity)
	fmt.Fprintf(w, "<!doctype html><meta charset=utf-8><title>ndpreview</title><body style=\"font:14px monospace;padding:24px\"><h1>Cannot preview</h1><pre>%s</pre><p>Fix the file and reload.</p>", html.EscapeString(err.Error()))
}

func (s *server) handleSetState(w http.ResponseWriter, r *http.Request) {
	var state map[string]any
	if err := json.NewDecoder(io.LimitReader(r.Body, maxBodyBytes)).Decode(&state); err != nil {
		http.Error(w, "the state must be a JSON object: "+err.Error(), http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	s.overrides = state
	s.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

// eventRequest is what the page posts when an element is tapped.
type eventRequest struct {
	Line   string `json:"line"`
	Action string `json:"action"`
}

func (s *server) handleEvent(w http.ResponseWriter, r *http.Request) {
	var request eventRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, maxBodyBytes)).Decode(&request); err != nil {
		http.Error(w, "bad event request: "+err.Error(), http.StatusBadRequest)
		return
	}
	var event map[string]any
	if err := json.Unmarshal([]byte(request.Line), &event); err != nil {
		http.Error(w, "the event is not a JSON line: "+err.Error(), http.StatusBadRequest)
		return
	}
	loaded, err := s.loadApp()
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	response, status := s.applyEvent(loaded, event, request)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)
}

func (s *server) applyEvent(loaded loadedApp, event map[string]any, request eventRequest) (map[string]any, int) {
	response := map[string]any{"send": event}
	actionID := request.Action
	switch event["event"] {
	case "action":
		actionID, _ = event["name"].(string)
	case "toggle", "input":
		bind, _ := event["bind"].(string)
		s.mu.Lock()
		s.overrides[bind] = event["value"]
		s.mu.Unlock()
		response["reload"] = true
	}
	if actionID == "" {
		return response, http.StatusOK
	}
	return s.describeAction(loaded.app, actionID, request.Line, response)
}

func (s *server) describeAction(app manifest.App, actionID, eventLine string, response map[string]any) (map[string]any, int) {
	for _, action := range app.Actions {
		if action.ID != actionID {
			continue
		}
		response["action"] = map[string]any{"id": action.ID, "label": action.Label, "run": action.Run}
		if s.runner != nil {
			response["exec"] = s.runner.run(app, action, eventLine)
		}
		return response, http.StatusOK
	}
	response["error"] = fmt.Sprintf("action %q is not in app.json", actionID)
	return response, http.StatusBadRequest
}

var errNotLoopback = errors.New("not a loopback address")

// isLoopbackAddress reports whether the listener address is on the loopback interface.
func isLoopbackAddress(addr net.Addr) error {
	tcp, ok := addr.(*net.TCPAddr)
	if !ok || !tcp.IP.IsLoopback() {
		return errNotLoopback
	}
	return nil
}
