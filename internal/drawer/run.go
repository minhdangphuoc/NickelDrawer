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
}

// Run executes an action declared in the app's manifest; nothing else can run.
func (c Config) Run(appID, actionID string) (RunResult, error) {
	app, err := c.loadApp(appID)
	if err != nil {
		return RunResult{}, err
	}
	for _, action := range app.Actions {
		if action.ID == actionID {
			return c.RunAction(app.Dir, action)
		}
	}
	return RunResult{}, newError(CodeUnknownAction, "app", appID, "action", actionID)
}

// RunAction runs one validated action whose script path is relative to baseDir. It is also
// used for host scripts, which the manifest scan never sees.
func (c Config) RunAction(baseDir string, action manifest.Action) (RunResult, error) {
	result := RunResult{V: ProtocolVersion, Rescan: manifest.RescanNone, Wifi: action.Wifi}
	if action.Wifi && !c.hasIP() {
		result.NeedsWifi = true
		return result, nil
	}
	script := filepath.Join(baseDir, filepath.FromSlash(action.Run))
	if action.Output == manifest.OutputNone {
		return c.startBackground(result, script, action)
	}
	return c.runForeground(result, script, action)
}

func (c Config) startBackground(result RunResult, script string, action manifest.Action) (RunResult, error) {
	if err := c.startScript()(script); err != nil {
		return RunResult{}, newError(CodeExecFailed, "app", action.ID, "detail", err.Error())
	}
	result.OK = true
	result.Rescan = action.RescanMode()
	return result, nil
}

func (c Config) runForeground(result RunResult, script string, action manifest.Action) (RunResult, error) {
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
