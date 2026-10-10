package drawer

import (
	"koboapps/internal/manifest"
)

// Rescan modes a script may request.
const (
	RescanModeQuick = "quick"
	RescanModeFull  = "full"
)

// Request asks the mod for a Nickel service the script cannot reach itself:
// {"request":"rescan","mode":"quick"|"full"} or {"request":"wifi"}. An app may only ask for
// what its app.json "requests" list declares.
type Request struct {
	Name string `json:"request"`
	Mode string `json:"mode,omitempty"` // rescan only: quick (default) or full
}

// stateRequestsKey is the key of the state object that carries requests of a finished
// background job (an array of Request objects).
const stateRequestsKey = "requests"

// acceptedRequests keeps the requests the app declared, normalised: an unknown name, an
// undeclared one or a bad mode is dropped and logged, never passed on. Duplicates collapse.
func (c Config) acceptedRequests(app manifest.App, asked []Request) []Request {
	var accepted []Request
	seen := map[Request]bool{}
	for _, request := range asked {
		normalised, reason := normaliseRequest(app, request)
		if reason != "" {
			c.debugf("app %q request %q dropped: %s", app.ID, request.Name, reason)
			continue
		}
		if !seen[normalised] {
			seen[normalised] = true
			accepted = append(accepted, normalised)
		}
	}
	return accepted
}

func normaliseRequest(app manifest.App, request Request) (Request, string) {
	if request.Name != manifest.RequestRescan && request.Name != manifest.RequestWifi {
		return Request{}, "unknown request"
	}
	if !app.MayRequest(request.Name) {
		return Request{}, "not declared in app.json requests"
	}
	if request.Name == manifest.RequestWifi {
		return Request{Name: manifest.RequestWifi}, ""
	}
	switch request.Mode {
	case "", RescanModeQuick:
		return Request{Name: manifest.RequestRescan, Mode: RescanModeQuick}, ""
	case RescanModeFull:
		return Request{Name: manifest.RequestRescan, Mode: RescanModeFull}, ""
	}
	return Request{}, "unknown rescan mode"
}

// stateRequests reads the "requests" array of a state object.
func stateRequests(state map[string]any) []Request {
	raw, _ := state[stateRequestsKey].([]any)
	asked := make([]Request, 0, len(raw))
	for _, entry := range raw {
		object, isObject := entry.(map[string]any)
		if !isObject {
			continue
		}
		name, _ := object["request"].(string)
		mode, _ := object["mode"].(string)
		asked = append(asked, Request{Name: name, Mode: mode})
	}
	return asked
}
