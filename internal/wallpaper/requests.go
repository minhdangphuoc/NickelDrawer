package wallpaper

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// HostRequest asks the drawer's host for a service, in the protocol's shape
// ({"request":"rescan","mode":"quick"}, docs/spec/drawer-protocol.md).
type HostRequest struct {
	Request string `json:"request"`
	Mode    string `json:"mode,omitempty"`
}

func (paths Paths) Requests() string { return filepath.Join(paths.Data(), "requests.json") }

// QueueRescan remembers that an import finished, so Nickel looks at the library again. The
// background job cannot answer the drawer itself; the next state read hands the request over.
func (service *Service) QueueRescan() {
	content, _ := json.Marshal([]HostRequest{{Request: "rescan", Mode: "quick"}})
	if err := writeFileAtomic(service.Paths.Requests(), "requests-", content); err != nil {
		service.Log.Printf("cannot queue the library rescan: %v", err)
	}
}

// TakeRequests returns the queued requests once and forgets them (renamed first, so two
// readers never both get one).
func (service *Service) TakeRequests() []HostRequest {
	taken := service.Paths.Requests() + ".taken"
	if os.Rename(service.Paths.Requests(), taken) != nil {
		return nil
	}
	defer os.Remove(taken)
	content, err := os.ReadFile(taken)
	var requests []HostRequest
	if err != nil || json.Unmarshal(content, &requests) != nil {
		return nil
	}
	return requests
}
