package library

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// hostRequest is a service the app asks the drawer's host for, in the protocol's shape
// (docs/spec/drawer-protocol.md): {"request":"rescan","mode":"quick"}.
type hostRequest struct {
	Request string `json:"request"`
	Mode    string `json:"mode,omitempty"`
}

var quickRescan = []hostRequest{{Request: "rescan", Mode: "quick"}}

// queueRescan remembers that books changed on disk, so Nickel should look at them again. A
// background job cannot answer the drawer itself, so the next state read hands the request
// over (takeRequests).
func (service *Service) queueRescan() {
	content, err := json.Marshal(quickRescan)
	if err == nil {
		err = writeFileAtomic(service.Paths.Requests(), content)
	}
	if err != nil {
		service.Logger.Printf("cannot queue the library rescan: %v", err)
	}
}

// takeRequests returns the queued requests once and forgets them. The file is renamed first
// so two readers never both get the same request.
func (service *Service) takeRequests() []hostRequest {
	taken := service.Paths.Requests() + ".taken"
	if err := os.Rename(service.Paths.Requests(), taken); err != nil {
		return nil
	}
	defer os.Remove(filepath.Clean(taken))
	content, err := os.ReadFile(taken)
	if err != nil {
		return nil
	}
	var requests []hostRequest
	if json.Unmarshal(content, &requests) != nil {
		return nil
	}
	return requests
}
