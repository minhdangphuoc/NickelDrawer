// Package project reads host/config/project.json, the one place that names the project: its
// GitHub repo, maintainer, license, supported firmware and supported models. The settings screen, the About
// screen and the default update.json all come from it, so a fork edits a single file. The
// version is not in it: it is the Makefile VERSION compiled into appsctl.
package project

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// FileName is the project config inside the host folder's config/.
const FileName = "project.json"

const gitHubHost = "github.com"

// modelIDLength is how many trailing digits of a Kobo device id name the model.
const modelIDLength = 3

var repoPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*/[A-Za-z0-9._-]+$`)

// modelIDPattern is the last three digits of a Kobo device id (see internal/device).
var modelIDPattern = regexp.MustCompile(`^[0-9]{3}$`)

// Model is one Kobo model the drawer runs on: every one has the tested 1264x1680 screen.
type Model struct {
	ID       string `json:"id"`       // last three digits of the device id, e.g. "388"
	Codename string `json:"codename"` // koboutils codename, e.g. "io"
	Name     string `json:"name"`
}

// Info is the content of project.json.
type Info struct {
	Name              string   `json:"name"`
	Maintainer        string   `json:"maintainer"`
	Repo              string   `json:"repo"` // owner/name on GitHub
	License           string   `json:"license"`
	Channel           string   `json:"channel"`
	SupportedFirmware []string `json:"supported_firmware"`
	SupportedModels   []Model  `json:"supported_models"`
}

// Path is where project.json lives below the host folder (.adds/koboapps).
func Path(hostDir string) string { return filepath.Join(hostDir, "config", FileName) }

// Load reads and validates project.json.
func Load(path string) (Info, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Info{}, fmt.Errorf("cannot read %s: %w", path, err)
	}
	var info Info
	if err := json.Unmarshal(data, &info); err != nil {
		return Info{}, fmt.Errorf("%s is not valid JSON: %w", path, err)
	}
	if err := info.validate(); err != nil {
		return Info{}, fmt.Errorf("%s: %w", path, err)
	}
	return info, nil
}

func (info Info) validate() error {
	switch {
	case info.Name == "":
		return fmt.Errorf("\"name\" is empty; set the project name")
	case !repoPattern.MatchString(info.Repo):
		return fmt.Errorf("\"repo\" must look like owner/name, got %q", info.Repo)
	case len(info.SupportedFirmware) == 0:
		return fmt.Errorf("\"supported_firmware\" is empty; list the firmware builds the mod supports")
	case len(info.SupportedModels) == 0:
		return fmt.Errorf("\"supported_models\" is empty; list the Kobo models the mod supports")
	}
	for _, model := range info.SupportedModels {
		if !modelIDPattern.MatchString(model.ID) || model.Codename == "" {
			return fmt.Errorf("\"supported_models\" entry %+v needs a 3-digit \"id\" and a \"codename\"", model)
		}
	}
	return nil
}

// URL is the project page, the target of the About QR code.
func (info Info) URL() string { return "https://" + gitHubHost + "/" + info.Repo }

// FirmwareRange reads "first – last" for the supported builds (a single build stands alone).
func (info Info) FirmwareRange() string {
	builds := info.SupportedFirmware
	if len(builds) == 0 {
		return ""
	}
	if len(builds) == 1 {
		return builds[0]
	}
	return builds[0] + " – " + builds[len(builds)-1]
}

// SupportsFirmware reports whether build (for example 4.38.23828) is in the supported list.
func (info Info) SupportsFirmware(build string) bool {
	for _, supported := range info.SupportedFirmware {
		if supported == build {
			return true
		}
	}
	return false
}

// SupportsModel reports whether a device id (full, or just its last three digits) is a
// supported model.
func (info Info) SupportsModel(deviceID string) bool {
	if len(deviceID) < modelIDLength {
		return false
	}
	suffix := deviceID[len(deviceID)-modelIDLength:]
	for _, model := range info.SupportedModels {
		if model.ID == suffix {
			return true
		}
	}
	return false
}

// UpdateConfigJSON is the default data/update.json: where releases come from.
func (info Info) UpdateConfigJSON() []byte {
	channel := info.Channel
	if channel == "" {
		channel = "stable"
	}
	var text strings.Builder
	fmt.Fprintf(&text, "{\n  \"repo\": %q,\n  \"channel\": %q\n}\n", info.Repo, channel)
	return []byte(text.String())
}
