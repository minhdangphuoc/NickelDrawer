package drawer

import (
	"encoding/json"
	"strings"
)

// ValueEnvName carries the value of a toggle, radio option, field or list row to the action
// script of a drawer event.
const ValueEnvName = "NICKELDRAWER_VALUE"

// Event describes one tap on an app screen.
type Event struct {
	AppID    string
	ActionID string
	Value    *string // nil: the action runs without NICKELDRAWER_VALUE
	ScreenID string  // the screen whose declared toasts and dialogs the script may open
}

// screenSignal is the one-line JSON a script may print to open a declared toast or dialog.
type screenSignal struct {
	Toast  string `json:"toast"`
	Dialog string `json:"dialog"`
}

// RunEvent runs an app action for a screen tap, like Run, with the value in the environment.
// When the script prints {"toast":"id"} or {"dialog":"id"} naming one declared on the screen,
// the reply carries that resolved toast or dialog instead of the raw text.
func (c Config) RunEvent(event Event) (RunResult, error) {
	if event.Value != nil {
		c.ScriptEnv = append(append([]string{}, c.ScriptEnv...), ValueEnvName+"="+*event.Value)
	}
	result, err := c.Run(event.AppID, event.ActionID)
	if err != nil || event.ScreenID == "" || result.Text == "" {
		return result, err
	}
	signal, isSignal := parseScreenSignal(result.Text)
	if !isSignal {
		return result, nil
	}
	screen, err := c.Screen(event.AppID, event.ScreenID)
	if err != nil {
		return result, nil // the action already ran; show its text rather than fail
	}
	return applyScreenSignal(result, signal, screen), nil
}

func parseScreenSignal(text string) (screenSignal, bool) {
	var signal screenSignal
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "{") || json.Unmarshal([]byte(trimmed), &signal) != nil {
		return screenSignal{}, false
	}
	return signal, signal.Toast != "" || signal.Dialog != ""
}

func applyScreenSignal(result RunResult, signal screenSignal, screen ScreenReply) RunResult {
	if toast, found := screen.Toasts[signal.Toast]; found {
		result.Toast = toast
		result.Text = ""
	}
	if dialog, found := screen.Dialogs[signal.Dialog]; found {
		result.Dialog = &dialog
		result.Text = ""
	}
	return result
}
