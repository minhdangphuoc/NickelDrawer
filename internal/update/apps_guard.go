package update

import (
	"errors"

	"koboapps/internal/manifest"
)

// ErrAppBusy means an app is running and cannot be stopped, so it must not be replaced now.
var ErrAppBusy = errors.New("the app is running and has no stop action")

// AppGuard makes sure an app is not running before its folder is replaced: it stops the app
// through its declared "stop" action, or returns ErrAppBusy when it runs and cannot be stopped.
type AppGuard interface {
	EnsureStopped(app manifest.App) error
}
