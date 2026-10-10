package drawer

import (
	"encoding/json"
	"strconv"
	"strings"

	"koboapps/internal/uikit"
)

// ValueEnvName carries the value of a toggle, radio option, field or list row to the action
// script of a drawer event.
const ValueEnvName = "NICKELDRAWER_VALUE"

// A list row menu event also tells the script which list and row: the list's bound variable
// and the zero-based row index. NICKELDRAWER_VALUE carries the row's primary text.
const (
	ListEnvName  = "NICKELDRAWER_LIST"
	IndexEnvName = "NICKELDRAWER_INDEX"
)

// Event describes one tap on an app screen.
type Event struct {
	AppID    string
	ActionID string
	Value    *string // nil: the action runs without NICKELDRAWER_VALUE
	ScreenID string  // the screen whose declared toasts and dialogs the script may open
	// List and Index mark a list row menu event: the bound variable of the list and the row.
	List  string
	Index int
}

// screenSignal is what a script prints to talk to the drawer: JSON objects, one per line,
// that open a declared toast or dialog or ask the mod for a service (Request).
type screenSignal struct {
	Toast    string
	Dialog   string
	Requests []Request
}

// signalLine is one JSON object of a script's output.
type signalLine struct {
	Toast   string `json:"toast"`
	Dialog  string `json:"dialog"`
	Request string `json:"request"`
	Mode    string `json:"mode"`
}

// RunEvent runs an app action for a screen tap, like Run, with the value in the environment.
// When the script prints {"toast":"id"} or {"dialog":"id"} naming one declared on the screen,
// the reply carries that resolved toast or dialog; a signal naming nothing declared is dropped.
func (c Config) RunEvent(event Event) (RunResult, error) {
	if event.Value != nil {
		c.ScriptEnv = append(append([]string{}, c.ScriptEnv...), ValueEnvName+"="+*event.Value)
	}
	if event.List != "" {
		if err := c.checkListMenu(event); err != nil {
			return RunResult{}, err
		}
		c.ScriptEnv = append(append([]string{}, c.ScriptEnv...),
			ListEnvName+"="+event.List, IndexEnvName+"="+strconv.Itoa(event.Index))
	}
	result, err := c.run(event.AppID, event.ActionID)
	if err != nil || event.ScreenID == "" || result.Text == "" {
		return withoutSignal(result), err
	}
	signal, isSignal := parseScreenSignal(result.Text)
	if !isSignal {
		return result, nil
	}
	screen, err := c.Screen(event.AppID, event.ScreenID, event.Value)
	if err != nil {
		return withoutSignal(result), nil // the action already ran; there is no screen to answer on
	}
	return withoutSignal(applyScreenSignal(result, signal, screen)), nil
}

// parseScreenSignal reads a script's whole output as a stream of JSON objects. Anything else
// (prose, a half-valid stream) is not a signal. The first toast and the first dialog win.
func parseScreenSignal(text string) (screenSignal, bool) {
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "{") {
		return screenSignal{}, false
	}
	var signal screenSignal
	decoder := json.NewDecoder(strings.NewReader(trimmed))
	for decoder.More() {
		var line signalLine
		if decoder.Decode(&line) != nil {
			return screenSignal{}, false
		}
		signal.add(line)
	}
	return signal, signal.Toast != "" || signal.Dialog != "" || len(signal.Requests) > 0
}

func (signal *screenSignal) add(line signalLine) {
	if signal.Toast == "" {
		signal.Toast = line.Toast
	}
	if signal.Dialog == "" {
		signal.Dialog = line.Dialog
	}
	if line.Request != "" {
		signal.Requests = append(signal.Requests, Request{Name: line.Request, Mode: line.Mode})
	}
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

// checkListMenu lets a menu event run only an action that the screen's list really offers in
// its row menu, so a crafted call cannot reach other actions through the list path.
func (c Config) checkListMenu(event Event) error {
	app, err := c.loadApp(event.AppID)
	if err != nil {
		return err
	}
	doc, err := loadScreenDocument(app)
	if err != nil {
		return err
	}
	screenID := event.ScreenID
	if screenID == "" {
		screenID = doc.Start
	}
	if screen, found := doc.Screens[screenID]; found && offersListMenu(screen, event.List, event.ActionID) {
		return nil
	}
	return newError(CodeUnknownAction, "app", event.AppID, "action", event.ActionID)
}

func offersListMenu(screen *uikit.Screen, bind, actionID string) bool {
	for _, component := range append(append([]uikit.Component{}, screen.Body...), screen.Footer...) {
		reference, _ := component.Main().(string)
		if component.Kind != "list" || strings.TrimPrefix(reference, "$") != bind {
			continue
		}
		entries, _ := component.Prop("menu").([]any)
		for _, entry := range entries {
			if fields, isObject := entry.(map[string]any); isObject && fields["id"] == actionID {
				return true
			}
		}
	}
	return false
}
