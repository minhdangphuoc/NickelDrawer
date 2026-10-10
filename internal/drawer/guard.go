package drawer

import (
	"context"
	"path/filepath"

	"koboapps/internal/manifest"
	"koboapps/internal/update"
)

const stopActionID = "stop"

// EnsureStopped implements update.AppGuard. An app that reports itself running through its
// "status" action is stopped through its "stop" action before its folder is replaced; one that
// runs and has no stop action, or does not stop, is busy.
func (c Config) EnsureStopped(app manifest.App) error {
	if !c.isAppRunning(app) {
		return nil
	}
	stop, hasStop := findAction(app, stopActionID)
	if !hasStop {
		return update.ErrAppBusy
	}
	if _, err := c.runAction(app.Dir, stop, true); err != nil {
		return err
	}
	if c.isAppRunning(app) {
		return update.ErrAppBusy
	}
	return nil
}

// isAppRunning asks the app's status action; an app without one, or whose status cannot be
// read in time, counts as not running (the same budget the drawer list uses).
func (c Config) isAppRunning(app manifest.App) bool {
	action, hasStatus := hasStatusAction(app)
	if !hasStatus {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), c.statusTimeout())
	defer cancel()
	output := c.runScript()(ctx, filepath.Join(app.Dir, filepath.FromSlash(action.Run)))
	return output.Err == nil && !output.TimedOut && parseStatus(output.Stdout).Running
}
