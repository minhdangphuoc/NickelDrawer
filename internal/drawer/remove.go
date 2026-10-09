package drawer

import (
	"os"
	"path/filepath"
	"strings"

	"koboapps/internal/manifest"
)

// RemoveResult is the response of `drawer remove`.
type RemoveResult struct {
	V    int    `json:"v"`
	OK   bool   `json:"ok"`
	Text string `json:"text"`
}

// Remove deletes one app folder. The folder must be a real directory directly under the apps
// root: a symlink, `..`, a host id or anything resolving elsewhere is refused, so a hostile
// or broken entry can never delete files outside the apps root. Broken apps can be removed.
func (c Config) Remove(appID string) (RemoveResult, error) {
	if IsHostID(appID) {
		return RemoveResult{}, newError(CodeHostItem, "app", appID)
	}
	if !isPlainName(appID) {
		return RemoveResult{}, newError(CodeRemoveRefused, "app", appID, "detail", "invalid app id")
	}
	target, err := c.containedAppDir(appID)
	if err != nil {
		return RemoveResult{}, err
	}
	name := c.displayName(appID)
	if err := os.RemoveAll(target); err != nil {
		return RemoveResult{}, newError(CodeIO, "detail", err.Error())
	}
	return RemoveResult{V: ProtocolVersion, OK: true, Text: c.message("drawer.removed", map[string]string{"name": name})}, nil
}

// containedAppDir returns the resolved folder to delete, or an error when it is not exactly
// <resolved apps root>/<id>.
func (c Config) containedAppDir(appID string) (string, error) {
	root, err := filepath.EvalSymlinks(c.AppsDir)
	if err != nil {
		return "", newError(CodeIO, "detail", err.Error())
	}
	candidate := filepath.Join(c.AppsDir, appID)
	linkInfo, err := os.Lstat(candidate)
	if err != nil {
		return "", newError(CodeUnknownApp, "app", appID)
	}
	if linkInfo.Mode()&os.ModeSymlink != 0 {
		return "", newError(CodeRemoveRefused, "app", appID, "detail", "it is a symbolic link")
	}
	if !linkInfo.IsDir() {
		return "", newError(CodeUnknownApp, "app", appID)
	}
	resolved, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "", newError(CodeIO, "detail", err.Error())
	}
	if resolved != filepath.Join(root, appID) || strings.Contains(appID, string(filepath.Separator)) {
		return "", newError(CodeRemoveRefused, "app", appID, "detail", "it resolves outside the apps folder")
	}
	return resolved, nil
}

// displayName is the localized app name when the manifest loads, else the id.
func (c Config) displayName(appID string) string {
	app, _, err := manifest.Load(filepath.Join(c.AppsDir, appID))
	if err != nil {
		return appID
	}
	if bundle, _ := app.LoadLocales(c.Locale); bundle != nil {
		app = app.Localize(bundle)
	}
	return app.Name
}
