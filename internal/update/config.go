// Package update downloads, verifies and applies signed NickelDrawer releases on the Kobo,
// and can roll the last update back. See docs/UPDATES.md.
package update

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"

	"koboapps/internal/i18n"
)

// Channel names accepted in update.json.
const (
	ChannelStable = "stable"
	ChannelBeta   = "beta"
)

var repoPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*/[A-Za-z0-9._-]+$`)

// Config is .adds/koboapps/data/update.json: where releases come from.
type Config struct {
	Repo    string `json:"repo"`
	Channel string `json:"channel"`
}

// Layout maps the onboard root to the files updates use.
type Layout struct {
	OnboardRoot string
	// AppsDir overrides <root>/.adds/apps; set by callers (and tests) that keep apps elsewhere.
	AppsDir string
}

func (layout Layout) hostDir() string { return filepath.Join(layout.OnboardRoot, ".adds", "koboapps") }
func (layout Layout) appsDir() string {
	if layout.AppsDir != "" {
		return layout.AppsDir
	}
	return filepath.Join(layout.OnboardRoot, ".adds", "apps")
}
func (layout Layout) configFile() string {
	return filepath.Join(layout.hostDir(), "data", "update.json")
}
func (layout Layout) workDir() string     { return filepath.Join(layout.hostDir(), "data", "update") }
func (layout Layout) statusFile() string  { return filepath.Join(layout.workDir(), "status.json") }
func (layout Layout) stagingDir() string  { return filepath.Join(layout.workDir(), "staging") }
func (layout Layout) rollbackDir() string { return filepath.Join(layout.hostDir(), "data", "rollback") }
func (layout Layout) appsctlPath() string { return filepath.Join(layout.hostDir(), "bin", "appsctl") }

// target resolves a manifest path (already checked by release.CheckPath) under the root.
func (layout Layout) target(manifestPath string) string {
	return filepath.Join(layout.OnboardRoot, filepath.FromSlash(manifestPath))
}

// LoadConfig reads and validates update.json. Every failure is a localized MessageError.
func LoadConfig(layout Layout) (Config, error) {
	file := layout.configFile()
	data, err := os.ReadFile(file)
	if os.IsNotExist(err) {
		return Config{}, i18n.NewError("update.no_config", "file", file)
	}
	if err != nil {
		return Config{}, i18n.NewError("update.bad_config", "file", file).WithCause(err)
	}
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return Config{}, i18n.NewError("update.bad_config", "file", file).WithCause(err)
	}
	if config.Channel == "" {
		config.Channel = ChannelStable
	}
	if config.Repo == "" {
		return Config{}, i18n.NewError("update.no_repo", "file", file)
	}
	if !repoPattern.MatchString(config.Repo) {
		return Config{}, i18n.NewError("update.bad_repo", "file", file, "repo", config.Repo)
	}
	if config.Channel != ChannelStable && config.Channel != ChannelBeta {
		return Config{}, i18n.NewError("update.bad_channel", "file", file, "channel", config.Channel)
	}
	return config, nil
}
