package update

import (
	"context"
	"errors"
	"fmt"
	"os"

	"koboapps/internal/manifest"
	"koboapps/internal/release"
)

// appsProgress reports which app a running job is on and what it does to it.
type appsProgress struct {
	layout Layout
	job    AppsJob
}

func (progress *appsProgress) startApp(position int, appID string) {
	progress.job.Done, progress.job.AppID, progress.job.Phase = position, appID, ""
	progress.write()
}

func (progress *appsProgress) phase(phase string) {
	progress.job.Phase = phase
	progress.write()
}

func (progress *appsProgress) write() {
	if err := writeAppsJob(progress.layout, progress.job); err != nil {
		fmt.Fprintln(os.Stderr, "warning: cannot write the app job progress:", err)
	}
}

// beginAppsJob refuses a second job while another process runs one, recovers any interrupted
// swap, and records the job as running. finish records its outcome.
func (updater *Updater) beginAppsJob(kind string, total int) (finish func(Report, error), err error) {
	if previous, found := ReadAppsJob(updater.layout); found && previous.State == StateRunning && previous.PID != os.Getpid() {
		return nil, appError("already_running")
	}
	if err := RecoverApps(updater.layout); err != nil {
		fmt.Fprintln(os.Stderr, "warning: app recovery:", err)
	}
	progress := &appsProgress{layout: updater.layout, job: AppsJob{State: StateRunning, Job: kind, PID: os.Getpid(), Total: total}}
	progress.job.Report = report("appupdate.working")
	if err := writeAppsJob(updater.layout, progress.job); err != nil {
		return nil, fmt.Errorf("cannot record the job: %w", err)
	}
	updater.progress = progress
	return func(result Report, runErr error) {
		done := AppsJob{State: StateDone, Job: kind, AppID: progress.job.AppID, Report: result}
		if runErr != nil {
			done = AppsJob{State: StateFailed, Job: kind, AppID: progress.job.AppID, Report: appFailure(runErr)}
		}
		if err := writeAppsJob(updater.layout, done); err != nil {
			fmt.Fprintln(os.Stderr, "warning: cannot write the app job outcome:", err)
		}
		updater.progress = nil
	}, nil
}

// ApplyApp installs a catalog app that is not installed, or updates an installed one.
func (updater *Updater) ApplyApp(ctx context.Context, id string) (result Report, err error) {
	kind := AppsJobInstall
	if exists(updater.layout.appDir(id)) {
		kind = AppsJobUpdate
	}
	finish, err := updater.beginAppsJob(kind, 1)
	if err != nil {
		return Report{}, err
	}
	defer func() { finish(result, err) }()
	found, err := updater.fetchCatalog(ctx)
	if err != nil {
		return Report{}, err
	}
	if err := updater.acceptCatalog(found); err != nil {
		return Report{}, err
	}
	app, isListed := found.catalog.Find(id)
	if !isListed {
		return Report{}, appError("unknown_app", "app", id)
	}
	updater.progress.startApp(1, id)
	return updater.installApp(ctx, found.repo, app)
}

// ApplyAllApps updates every installed catalog app that has a newer version. A failure on one
// app does not stop the others; the result says how many worked.
func (updater *Updater) ApplyAllApps(ctx context.Context) (result Report, err error) {
	finish, err := updater.beginAppsJob(AppsJobUpdate, 0)
	if err != nil {
		return Report{}, err
	}
	defer func() { finish(result, err) }()
	found, err := updater.fetchCatalog(ctx)
	if err != nil {
		return Report{}, err
	}
	if err := updater.acceptCatalog(found); err != nil {
		return Report{}, err
	}
	var targets []AppEntry
	for _, entry := range AppEntries(found.catalog, InstalledAppVersions(updater.layout), updater.options.Language) {
		if entry.State == AppStateUpdate {
			targets = append(targets, entry)
		}
	}
	if len(targets) == 0 {
		return report("appupdate.up_to_date"), nil
	}
	updater.progress.job.Total = len(targets)
	return updater.installAll(ctx, found, targets)
}

func (updater *Updater) installAll(ctx context.Context, found verifiedCatalog, targets []AppEntry) (Report, error) {
	var failures []error
	updated := 0
	for position, target := range targets {
		updater.progress.startApp(position+1, target.ID)
		app, _ := found.catalog.Find(target.ID)
		if _, err := updater.installApp(ctx, found.repo, app); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", target.ID, err)
			failures = append(failures, err)
			continue
		}
		updated++
	}
	switch {
	case len(failures) == len(targets):
		return Report{}, failures[0]
	case len(failures) > 0:
		return report("appupdate.updated_some", "done", fmt.Sprint(updated), "failed", fmt.Sprint(len(failures))), nil
	}
	return report("appupdate.updated_all", "count", fmt.Sprint(updated)), nil
}

// installApp downloads, verifies and swaps in one catalog app. Installed versions are never
// replaced by an equal or older one.
func (updater *Updater) installApp(ctx context.Context, repo string, app release.CatalogApp) (Report, error) {
	installed, isInstalled := InstalledAppVersions(updater.layout)[app.ID]
	if isInstalled {
		if order, err := release.CompareVersions(app.Version, installed); err != nil || order <= 0 {
			return report("appupdate.up_to_date", "app", app.ID), err
		}
	}
	if err := updater.checkDrawerRequirement(app); err != nil {
		return Report{}, err
	}
	if isInstalled {
		if err := updater.stopRunningApp(app.ID); err != nil {
			return Report{}, err
		}
	}
	if err := updater.fetchAndSwap(ctx, repo, app); err != nil {
		return Report{}, err
	}
	updater.layout.requestRescan()
	key := "appupdate.installed"
	if isInstalled {
		key = "appupdate.updated"
	}
	return report(key, "app", app.ID, "version", app.Version, "previous", installed), nil
}

func (updater *Updater) checkDrawerRequirement(app release.CatalogApp) error {
	meets, err := app.MeetsDrawerRequirement(updater.options.CurrentVersion)
	if err != nil {
		return appError("bad_catalog", "app", app.ID).WithCause(err)
	}
	if !meets {
		required, _ := release.ParseDrawerConstraint(app.Requires.Drawer)
		return appError("needs_newer_drawer", "app", app.ID, "required", required)
	}
	return nil
}

// stopRunningApp asks the guard to stop the installed app; ErrAppBusy becomes "app busy".
func (updater *Updater) stopRunningApp(id string) error {
	if updater.options.Guard == nil {
		return nil
	}
	updater.progress.phase(PhaseStop)
	app, _, err := manifest.Load(updater.layout.appDir(id))
	if err != nil {
		return nil // a folder that does not load runs nothing
	}
	if err := updater.options.Guard.EnsureStopped(app); err != nil {
		if errors.Is(err, ErrAppBusy) {
			return appError("app_busy", "app", id)
		}
		return appError("app_busy", "app", id).WithCause(err)
	}
	return nil
}

func (updater *Updater) fetchAndSwap(ctx context.Context, repo string, app release.CatalogApp) error {
	workDir, err := os.MkdirTemp(updater.options.TempDir, "nickeldrawer-app-")
	if err != nil {
		return fmt.Errorf("cannot create a temp folder in %s: %w", updater.options.TempDir, err)
	}
	defer os.RemoveAll(workDir)
	updater.progress.phase(PhaseDownload)
	zipPath, err := updater.downloadAppZip(ctx, repo, app, workDir)
	if err != nil {
		return err
	}
	if err := checkZipIdentity(zipPath, app); err != nil {
		return err
	}
	updater.progress.phase(PhaseInstall)
	if err := updater.layout.stageAppZip(app, zipPath); err != nil {
		return err
	}
	return updater.swapStaged(app.ID, app.Version)
}

// checkZipIdentity makes sure the zip holds the app and version the signed entry names.
func checkZipIdentity(zipPath string, app release.CatalogApp) error {
	id, version, err := release.IdentifyAppZip(zipPath)
	if err != nil {
		return appError("unsafe_zip", "app", app.ID).WithCause(err)
	}
	if id != app.ID || version != app.Version {
		return appError("invalid_app", "app", app.ID).WithCause(
			fmt.Errorf("the zip holds %s %s, the catalog says %s %s", id, version, app.ID, app.Version))
	}
	return nil
}

// swapStaged replaces the installed app by its staged folder and keeps the old version as the
// rollback copy. A swap that fails is undone and reported as rolled back.
func (updater *Updater) swapStaged(id, version string) error {
	swap := appSwap{layout: updater.layout, id: id, version: version, afterStep: updater.afterSwapStep}
	if _, err := swap.run(); err != nil {
		if errors.Is(err, errPowerCut) {
			return err
		}
		os.RemoveAll(updater.layout.stagedAppDir(id))
		return appError("rolled_back", "app", id).WithCause(err)
	}
	if err := updater.layout.keepAsBackup(id); err != nil {
		fmt.Fprintln(os.Stderr, "warning:", err)
	}
	return nil
}

// RollbackApp puts the previous version of an app back. Its data/ stays where it is.
func (updater *Updater) RollbackApp(id string) (result Report, err error) {
	finish, err := updater.beginAppsJob(AppsJobRollback, 1)
	if err != nil {
		return Report{}, err
	}
	defer func() { finish(result, err) }()
	updater.progress.startApp(1, id)
	backup := updater.layout.appRollbackDir(id)
	app, _, loadErr := manifest.Load(backup)
	if !exists(updater.layout.appDir(id)) || loadErr != nil {
		return Report{}, appError("no_rollback", "app", id)
	}
	if err := updater.stopRunningApp(id); err != nil {
		return Report{}, err
	}
	stagedDir := updater.layout.stagedAppDir(id)
	if err := removeIfNoData(stagedDir); err != nil {
		return Report{}, err
	}
	if err := os.Rename(backup, stagedDir); err != nil {
		return Report{}, fmt.Errorf("cannot prepare the earlier version: %w", err)
	}
	swap := appSwap{layout: updater.layout, id: id, version: app.Version, afterStep: updater.afterSwapStep}
	if _, err := swap.run(); err != nil {
		if errors.Is(err, errPowerCut) {
			return Report{}, err
		}
		if !exists(backup) && !holdsData(stagedDir) {
			_ = os.Rename(stagedDir, backup) // the earlier version stays available
		}
		return Report{}, appError("rolled_back", "app", id).WithCause(err)
	}
	if err := removeIfNoData(updater.layout.oldAppDir(id)); err != nil {
		fmt.Fprintln(os.Stderr, "warning:", err)
	}
	updater.layout.requestRescan()
	return report("appupdate.rolled_back_app", "app", id, "version", app.Version), nil
}

// AppName is the display name of an app for reports: the cached catalog's, else the id.
func (updater *Updater) AppName(id string) string {
	return AppDisplayName(updater.layout, id, updater.options.Language)
}

// AppDisplayName looks the name up in the cached catalog; unknown apps show their id.
func AppDisplayName(layout Layout, id, language string) string {
	if catalog, found := ReadAppsCatalog(layout); found {
		if app, isListed := catalog.Find(id); isListed {
			return app.DisplayName(language)
		}
	}
	return id
}
