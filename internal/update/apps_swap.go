package update

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"koboapps/internal/manifest"
)

// Steps of an app swap, named for the power-cut tests: after each one the next start must
// still find the user's data/ in exactly one folder.
const (
	stepDataMoved  = "data-moved"
	stepOldRenamed = "old-renamed"
	stepNewRenamed = "new-renamed"
	stepValidated  = "validated"
)

// errPowerCut is what a test hook returns to stop the swap dead, as a power cut would: no
// rollback runs, so the recovery on the next start is what gets tested.
var errPowerCut = errors.New("simulated power cut")

// appSwap replaces <id> by a staged folder without ever copying, reading or deleting data/.
// The user's data folder is moved into the staged folder first (renames on one file system),
// so at every instant it lies inside exactly one of <id>, .<id>.old or .<id>.new.
type appSwap struct {
	layout  Layout
	id      string
	version string // the version the installed app must report afterwards
	// afterStep, when set, is called after each step; tests make it fail to cut the power.
	afterStep func(step string) error
}

func (swap appSwap) reached(step string) error {
	if swap.afterStep == nil {
		return nil
	}
	return swap.afterStep(step)
}

// run puts the staged folder in place of the installed app. hadOld says whether an older
// version was installed, in which case it is left in .<id>.old (holding no data/).
func (swap appSwap) run() (hadOld bool, err error) {
	appDir, oldDir, stagedDir := swap.layout.appDir(swap.id), swap.layout.oldAppDir(swap.id), swap.layout.stagedAppDir(swap.id)
	hadOld = exists(appDir)
	if exists(oldDir) {
		return false, fmt.Errorf("%s is left over from an earlier update; run any appsctl update command to recover it", oldDir)
	}
	if hadOld {
		if err := swap.moveDataToStaged(appDir, stagedDir); err != nil {
			return hadOld, err
		}
		if err := os.Rename(appDir, oldDir); err != nil {
			return hadOld, swap.undo(hadOld, fmt.Errorf("cannot move %s aside: %w", appDir, err))
		}
		if err := swap.reached(stepOldRenamed); err != nil {
			return hadOld, err
		}
	}
	if err := os.Rename(stagedDir, appDir); err != nil {
		return hadOld, swap.undo(hadOld, fmt.Errorf("cannot move the new version into place: %w", err))
	}
	if err := swap.reached(stepNewRenamed); err != nil {
		return hadOld, err
	}
	if err := swap.validate(); err != nil {
		return hadOld, swap.undo(hadOld, err)
	}
	return hadOld, swap.reached(stepValidated)
}

func (swap appSwap) moveDataToStaged(appDir, stagedDir string) error {
	if !holdsData(appDir) {
		return nil
	}
	if holdsData(stagedDir) {
		return fmt.Errorf("the new version in %s already has a data folder; refusing to touch either", stagedDir)
	}
	if err := os.Rename(filepath.Join(appDir, dataFolder), filepath.Join(stagedDir, dataFolder)); err != nil {
		return fmt.Errorf("cannot move the app's data folder: %w", err)
	}
	return swap.reached(stepDataMoved)
}

// validate loads the installed app with the same loader the drawer uses.
func (swap appSwap) validate() error {
	app, _, err := manifest.Load(swap.layout.appDir(swap.id))
	if err != nil {
		return fmt.Errorf("the new version of %s is not a valid app: %w", swap.id, err)
	}
	if app.Version != swap.version {
		return fmt.Errorf("the new %s reports version %s, expected %s", swap.id, app.Version, swap.version)
	}
	return nil
}

// undo puts the old version and its data back after a failed step. It is not run for a
// simulated power cut. The returned error says why the swap failed and, if undoing failed too,
// that the next start will finish the recovery.
func (swap appSwap) undo(hadOld bool, cause error) error {
	appDir, oldDir, stagedDir := swap.layout.appDir(swap.id), swap.layout.oldAppDir(swap.id), swap.layout.stagedAppDir(swap.id)
	if err := swap.restoreOld(hadOld, appDir, oldDir, stagedDir); err != nil {
		return errors.Join(cause, fmt.Errorf("could not undo; the next appsctl start recovers it: %w", err))
	}
	return cause
}

func (swap appSwap) restoreOld(hadOld bool, appDir, oldDir, stagedDir string) error {
	if exists(appDir) && !hadOld {
		return os.RemoveAll(appDir) // a failed fresh install: no data/ can be in it
	}
	if hadOld && exists(oldDir) {
		if err := returnDataTo(oldDir, appDir, stagedDir); err != nil {
			return err
		}
		if err := removeIfNoData(appDir); err != nil {
			return err
		}
		if err := os.Rename(oldDir, appDir); err != nil {
			return err
		}
	} else if hadOld {
		// The old folder never moved: data/ went to the staged folder, bring it home.
		if err := moveData(stagedDir, appDir); err != nil {
			return err
		}
	}
	return nil // the staged folder stays: the caller decides whether it is dropped or kept
}

// returnDataTo moves data/ from whichever of the folders holds it into target (the old folder).
func returnDataTo(target string, sources ...string) error {
	for _, source := range sources {
		if err := moveData(source, target); err != nil {
			return err
		}
	}
	return nil
}

// moveData renames source/data to target/data when source has one and target has none.
func moveData(source, target string) error {
	if !holdsData(source) {
		return nil
	}
	if holdsData(target) {
		return fmt.Errorf("both %s and %s hold a data folder; leaving both", source, target)
	}
	return os.Rename(filepath.Join(source, dataFolder), filepath.Join(target, dataFolder))
}

// removeIfNoData deletes a leftover folder, but never one that holds data/.
func removeIfNoData(folder string) error {
	if !exists(folder) {
		return nil
	}
	if holdsData(folder) {
		return fmt.Errorf("%s holds a data folder; not deleting it", folder)
	}
	return os.RemoveAll(folder)
}

// keepAsBackup turns the old version into the app's rollback copy, replacing the earlier one.
// The old folder holds no data/ here; if it does, it is left alone for the recovery to sort out.
func (layout Layout) keepAsBackup(id string) error {
	oldDir := layout.oldAppDir(id)
	if !exists(oldDir) {
		return nil
	}
	if holdsData(oldDir) {
		return fmt.Errorf("%s holds a data folder; not turning it into a backup", oldDir)
	}
	backup := layout.appRollbackDir(id)
	if err := os.RemoveAll(backup); err != nil {
		return fmt.Errorf("cannot replace the earlier backup %s: %w", backup, err)
	}
	if err := os.MkdirAll(filepath.Dir(backup), 0o755); err != nil {
		return err
	}
	return os.Rename(oldDir, backup)
}

// RecoverApps finishes or undoes any app swap a power cut or crash interrupted, for every app
// with a leftover .<id>.old or .<id>.new folder. It never deletes a folder that holds data/.
// The problems it could not fix are returned; the user's data is never lost to them.
func RecoverApps(layout Layout) error {
	var problems []error
	for _, id := range layout.leftoverIDs() {
		if err := recoverApp(layout, id); err != nil {
			problems = append(problems, fmt.Errorf("app %s: %w", id, err))
		}
	}
	return errors.Join(problems...)
}

func recoverApp(layout Layout, id string) error {
	appDir, oldDir, stagedDir := layout.appDir(id), layout.oldAppDir(id), layout.stagedAppDir(id)
	if !exists(appDir) && exists(oldDir) {
		if err := os.Rename(oldDir, appDir); err != nil {
			return fmt.Errorf("cannot restore %s: %w", oldDir, err)
		}
	}
	if !exists(appDir) {
		return removeIfNoData(stagedDir) // nothing was installed; a lone staged folder is only a partial download
	}
	if err := returnDataTo(appDir, oldDir, stagedDir); err != nil {
		return err
	}
	if err := fallBackIfBroken(layout, id); err != nil {
		return err
	}
	return errors.Join(removeIfNoData(oldDir), removeIfNoData(stagedDir))
}

// fallBackIfBroken restores the old version when the installed folder does not load, from the
// old folder or else the rollback copy. data/ moves along.
func fallBackIfBroken(layout Layout, id string) error {
	appDir, oldDir := layout.appDir(id), layout.oldAppDir(id)
	if _, _, err := manifest.Load(appDir); err == nil {
		return nil
	}
	if !exists(oldDir) && exists(layout.appRollbackDir(id)) {
		if err := os.Rename(layout.appRollbackDir(id), oldDir); err != nil {
			return fmt.Errorf("cannot use the rollback copy: %w", err)
		}
	}
	if !exists(oldDir) {
		return nil // nothing to fall back to; the broken app stays for the user to see
	}
	if err := moveData(appDir, oldDir); err != nil {
		return err
	}
	if err := removeIfNoData(appDir); err != nil {
		return err
	}
	return os.Rename(oldDir, appDir)
}
