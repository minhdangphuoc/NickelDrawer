package drawer

import (
	"context"
	"path/filepath"
	"strconv"
	"strings"

	"koboapps/internal/manifest"
)

// RunResult is the response of `drawer run`.
type RunResult struct {
	V         int    `json:"v"`
	OK        bool   `json:"ok"`
	Text      string `json:"text"`
	Rescan    string `json:"rescan"`
	Wifi      bool   `json:"wifi"`
	NeedsWifi bool   `json:"needs_wifi,omitempty"`
	// Set by drawer event when the script opened a toast or dialog declared on the screen.
	Toast  string        `json:"toast,omitempty"`
	Dialog *ScreenDialog `json:"dialog,omitempty"`
	// Services the script asked the mod for and its app.json allows (rescan, wifi).
	Requests []Request `json:"requests,omitempty"`
}

// Run executes an action declared in the app's manifest; nothing else can run. Without a
// screen there is nowhere to show a toast or dialog, so a signal the script prints is dropped.
func (c Config) Run(appID, actionID string) (RunResult, error) {
	result, err := c.run(appID, actionID)
	return withoutSignal(result), err
}

// run is Run before unresolved toast and dialog signals are dropped.
func (c Config) run(appID, actionID string) (RunResult, error) {
	app, err := c.loadApp(appID)
	if err != nil {
		return RunResult{}, err
	}
	for _, action := range app.Actions {
		if action.ID == actionID {
			result, err := c.runAction(app.Dir, action, true)
			return c.withRequests(app, result), err
		}
	}
	return RunResult{}, newError(CodeUnknownAction, "app", appID, "action", actionID)
}

// withRequests moves the requests a script printed into the reply, after the app's own
// allowlist has filtered them.
func (c Config) withRequests(app manifest.App, result RunResult) RunResult {
	if signal, isSignal := parseScreenSignal(result.Text); isSignal {
		result.Requests = c.acceptedRequests(app, signal.Requests)
	}
	return result
}

// RunAction runs one validated action whose script path is relative to baseDir. It is also
// used for host scripts, which the manifest scan never sees.
func (c Config) RunAction(baseDir string, action manifest.Action) (RunResult, error) {
	return c.runAction(baseDir, action, false)
}

// runAction is RunAction; fromApp marks a script written by an app, whose prose never
// reaches the screen (see appText).
func (c Config) runAction(baseDir string, action manifest.Action, fromApp bool) (RunResult, error) {
	result := RunResult{V: ProtocolVersion, Rescan: manifest.RescanNone, Wifi: action.Wifi}
	if action.Wifi && !c.hasIP() {
		result.NeedsWifi = true
		return result, nil
	}
	script := filepath.Join(baseDir, filepath.FromSlash(action.Run))
	if action.Output == manifest.OutputNone {
		return c.startBackground(result, script, action)
	}
	return c.runForeground(result, script, action, fromApp)
}

func (c Config) startBackground(result RunResult, script string, action manifest.Action) (RunResult, error) {
	if err := c.startScript()(script); err != nil {
		return RunResult{}, newError(CodeExecFailed, "app", action.ID, "detail", err.Error())
	}
	result.OK = true
	result.Rescan = action.RescanMode()
	return result, nil
}

func (c Config) runForeground(result RunResult, script string, action manifest.Action, fromApp bool) (RunResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.showTimeout())
	defer cancel()
	output := c.runScript()(ctx, script)
	switch {
	case output.Err != nil:
		return RunResult{}, newError(CodeExecFailed, "app", action.ID, "detail", output.Err.Error())
	case output.TimedOut:
		result.Text = c.message("drawer.timeout", map[string]string{
			"label": action.Label, "seconds": strconv.Itoa(int(c.showTimeout().Seconds()))})
		return result, nil
	}
	result.Text = scriptText(output)
	result.OK = output.ExitCode == 0
	if fromApp {
		result.Text = c.appText(action, result.Text, result.OK)
	}
	if result.OK {
		result.Rescan = action.RescanMode() // a failed action must not trigger a rescan
	}
	return result, nil
}

// scriptText prefers stdout, which is what NickelMenu would show; stderr only helps when
// the script printed nothing else.
func scriptText(output ScriptOutput) string {
	text := strings.TrimRight(output.Stdout, "\r\n")
	if text == "" && output.ExitCode != 0 {
		text = strings.TrimRight(output.Stderr, "\r\n")
	}
	return text
}

// appText keeps only what an app may send to the screen: a toast or dialog signal. Any other
// output is prose written for a text dialog; apps talk through the NickelDrawer API, so it
// goes to the debug log and is dropped. A failed action is reported by a short localized
// message instead of the script's words.
func (c Config) appText(action manifest.Action, text string, isOK bool) string {
	if _, isSignal := parseScreenSignal(text); isSignal {
		return text
	}
	if text != "" {
		c.debugf("action %q printed text outside the drawer protocol, dropped: %s", action.ID, text)
	}
	if !isOK {
		return c.message("drawer.script_failed", map[string]string{"label": action.Label})
	}
	return ""
}

func withoutSignal(result RunResult) RunResult {
	if _, isSignal := parseScreenSignal(result.Text); isSignal {
		result.Text = ""
	}
	return result
}
