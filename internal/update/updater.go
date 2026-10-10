package update

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"koboapps/internal/i18n"
	"koboapps/internal/manifest"
	"koboapps/internal/release"
)

const selfCheckTimeout = 10 * time.Second

// Options configure an Updater. Paths and the API base are settable so tests run against a
// temp folder and a fake GitHub.
type Options struct {
	OnboardRoot    string
	APIBase        string
	DownloadBase   string // where release downloads live (default https://github.com); tests use a fake
	TempDir        string // downloads land here, e.g. /tmp, never on the onboard storage
	CurrentVersion string // version of the running appsctl, i.e. of the installed release
	AllowDowngrade bool
	HTTPClient     *http.Client
	// PublicKey returns the trusted release key; ErrNoPublicKey refuses every update.
	PublicKey func() (ed25519.PublicKey, error)
	// Guard stops a running app before it is replaced; nil means no app is ever running.
	Guard AppGuard
	// Language picks app names in reports ("vi"); empty means English.
	Language string
}

// Updater applies signed releases to one onboard root.
type Updater struct {
	options Options
	layout  Layout
	github  *githubClient
	// progress is set while an app job runs, so its steps reach the screen.
	progress *appsProgress
	// afterSwapStep lets tests cut the power in the middle of an app swap.
	afterSwapStep func(step string) error
}

// New builds an Updater; zero-valued optional fields get safe defaults.
func New(options Options) *Updater {
	if options.PublicKey == nil {
		options.PublicKey = release.EmbeddedPublicKey
	}
	if options.TempDir == "" {
		options.TempDir = os.TempDir()
	}
	return &Updater{
		options: options,
		layout:  Layout{OnboardRoot: options.OnboardRoot},
		github:  newGithubClient(options.HTTPClient, options.APIBase),
	}
}

// Check reports whether a newer release exists. It downloads and verifies only release.json
// (signature first) and installs nothing; the result is cached for the drawer
// (data/update/available.json).
func (updater *Updater) Check(ctx context.Context) (result Report, err error) {
	previous, hadPrevious, err := updater.beginCheck()
	if err != nil {
		return Report{}, err
	}
	defer func() {
		updater.recordCheck(result, err)
		updater.finishCheck(previous, hadPrevious, result, err)
	}()
	signed, err := updater.fetchSignedManifest(ctx)
	if err != nil {
		return Report{}, err
	}
	order, err := release.CompareVersions(signed.manifest.Version, updater.options.CurrentVersion)
	if err != nil {
		return Report{}, err
	}
	isAvailable := order > 0
	if err := writeAvailability(updater.layout, availabilityOf(signed.manifest, updater.options.CurrentVersion, isAvailable)); err != nil {
		return Report{}, fmt.Errorf("cannot save the check result: %w", err)
	}
	if !isAvailable {
		return report("update.up_to_date", "version", updater.options.CurrentVersion), nil
	}
	return report("update.available", "version", signed.manifest.Version, "current", updater.options.CurrentVersion), nil
}

// Apply installs the latest release if it is newer. On any failure after the first file
// changed, the previous files are restored.
func (updater *Updater) Apply(ctx context.Context) (result Report, err error) {
	if err := updater.beginRun(); err != nil {
		return Report{}, err
	}
	defer func() { updater.finishRun(result, err) }()
	signed, err := updater.fetchSignedManifest(ctx)
	if err != nil {
		return Report{}, err
	}
	if upToDate, err := updater.checkVersionOrder(signed.manifest.Version); err != nil || upToDate {
		return report("update.up_to_date", "version", updater.options.CurrentVersion), err
	}
	workDir, err := os.MkdirTemp(updater.options.TempDir, "nickeldrawer-update-")
	if err != nil {
		return Report{}, fmt.Errorf("cannot create a temp folder in %s: %w", updater.options.TempDir, err)
	}
	defer os.RemoveAll(workDir)
	zipPath, err := updater.downloadVerifiedZip(ctx, signed, workDir)
	if err != nil {
		return Report{}, err
	}
	isModStaged, err := updater.install(signed.manifest, zipPath)
	if err != nil {
		return Report{}, err
	}
	updater.forgetAvailability()
	key := "update.applied"
	if isModStaged {
		key = "update.applied_reboot"
	}
	return report(key, "version", signed.manifest.Version, "previous", updater.options.CurrentVersion), nil
}

// forgetAvailability drops the cached check result after an install; it only says "1 update"
// for a version that is now installed, and a stale cache is also ignored by the readers.
func (updater *Updater) forgetAvailability() {
	if err := os.Remove(updater.layout.availableFile()); err != nil && !os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "warning: cannot remove the cached check result:", err)
	}
}

type signedRelease struct {
	manifest *release.Manifest
	info     releaseInfo
}

// fetchSignedManifest finds the release, downloads release.json and its signature, and only
// returns the manifest if the signature verifies under the trusted key.
func (updater *Updater) fetchSignedManifest(ctx context.Context) (signedRelease, error) {
	config, err := LoadConfig(updater.layout)
	if err != nil {
		return signedRelease{}, err
	}
	key, err := updater.trustedKey()
	if err != nil {
		return signedRelease{}, err
	}
	info, err := updater.github.findRelease(ctx, config)
	if err != nil {
		return signedRelease{}, err
	}
	manifestBytes, err := updater.downloadAsset(ctx, info, release.ManifestName, maxReleaseJSON)
	if err != nil {
		return signedRelease{}, err
	}
	signature, err := updater.downloadAsset(ctx, info, release.SignatureName, maxSignature)
	if err != nil {
		return signedRelease{}, err
	}
	if err := release.Verify(key, manifestBytes, string(signature)); err != nil {
		return signedRelease{}, i18n.NewError("update.bad_signature").WithCause(err)
	}
	parsed, err := release.Parse(manifestBytes)
	if err != nil {
		return signedRelease{}, i18n.NewError("update.bad_manifest").WithCause(err)
	}
	if err := updater.checkMinimumAppsctl(parsed); err != nil {
		return signedRelease{}, err
	}
	return signedRelease{manifest: parsed, info: info}, nil
}

func (updater *Updater) trustedKey() (ed25519.PublicKey, error) {
	key, err := updater.options.PublicKey()
	if errors.Is(err, release.ErrNoPublicKey) {
		return nil, i18n.NewError("update.no_public_key")
	}
	if err != nil {
		return nil, i18n.NewError("update.bad_public_key").WithCause(err)
	}
	return key, nil
}

func (updater *Updater) downloadAsset(ctx context.Context, info releaseInfo, name string, limit int64) ([]byte, error) {
	asset, err := info.asset(name)
	if err != nil {
		return nil, err
	}
	return updater.github.download(ctx, asset.URL, limit)
}

func (updater *Updater) checkMinimumAppsctl(parsed *release.Manifest) error {
	order, err := release.CompareVersions(updater.options.CurrentVersion, parsed.MinAppsctl)
	if err != nil {
		return err
	}
	if order < 0 {
		return i18n.NewError("update.needs_newer_appsctl", "version", parsed.Version, "required", parsed.MinAppsctl)
	}
	return nil
}

// checkVersionOrder says whether the release is the installed version (nothing to do) and
// refuses older releases unless downgrades are allowed.
func (updater *Updater) checkVersionOrder(version string) (isUpToDate bool, err error) {
	order, err := release.CompareVersions(version, updater.options.CurrentVersion)
	if err != nil {
		return false, err
	}
	if order == 0 {
		return true, nil
	}
	if order < 0 && !updater.options.AllowDowngrade {
		return false, i18n.NewError("update.downgrade", "version", version, "current", updater.options.CurrentVersion)
	}
	return false, nil
}

func (updater *Updater) downloadVerifiedZip(ctx context.Context, signed signedRelease, workDir string) (string, error) {
	asset, err := signed.info.asset("nickeldrawer-" + signed.manifest.Version + ".zip")
	if err != nil {
		return "", err
	}
	zipPath := filepath.Join(workDir, "release.zip")
	if err := updater.github.downloadToFile(ctx, asset.URL, zipPath, maxZipBytes); err != nil {
		return "", err
	}
	if err := release.VerifyZip(signed.manifest, zipPath); err != nil {
		return "", i18n.NewError("update.file_mismatch").WithCause(err)
	}
	return zipPath, nil
}

// install stages the changed files, backs up what they replace, swaps them in and checks
// the result. Everything before the swap leaves the device untouched on failure. When the
// release carries a changed Nickel mod, it is staged last as .kobo/KoboRoot.tgz and the
// result says a reboot installs it.
func (updater *Updater) install(parsed *release.Manifest, zipPath string) (isModStaged bool, err error) {
	changed, err := updater.changedFiles(parsed)
	if err != nil {
		return false, err
	}
	mod, hasMod := findFile(changed, release.ModPath)
	if hasMod {
		if err := updater.layout.ensureModSlotFree(mod); err != nil {
			return false, err
		}
	}
	appsBefore := updater.invalidAppIDs()
	if err := updater.stage(parsed, zipPath, changed); err != nil {
		return false, err
	}
	defer os.RemoveAll(updater.layout.stagingDir())
	backupDir, err := updater.layout.createBackup(updater.options.CurrentVersion, parsed.Version, changed)
	if err != nil {
		return false, err
	}
	if err := updater.swap(changed); err != nil {
		return false, updater.undo(backupDir, err)
	}
	if err := updater.selfCheck(parsed, changed, appsBefore); err != nil {
		return false, updater.undo(backupDir, err)
	}
	if hasMod {
		if err := updater.layout.stageMod(); err != nil {
			return false, updater.undo(backupDir, err)
		}
	}
	if err := updater.layout.pruneBackups(); err != nil {
		return hasMod, fmt.Errorf("update applied, but old backups could not be pruned: %w", err)
	}
	return hasMod, nil
}

// changedFiles keeps the manifest files whose installed copy is missing or different.
func (updater *Updater) changedFiles(parsed *release.Manifest) ([]release.File, error) {
	var changed []release.File
	for _, file := range parsed.Files {
		current, err := hashIfExists(updater.layout.target(file.Path), release.HashFile)
		if err != nil {
			return nil, fmt.Errorf("cannot read installed %s: %w", file.Path, err)
		}
		if current != file.SHA256 {
			changed = append(changed, file)
		}
	}
	return changed, nil
}

func (updater *Updater) stage(parsed *release.Manifest, zipPath string, changed []release.File) error {
	stagingDir := updater.layout.stagingDir()
	if err := os.RemoveAll(stagingDir); err != nil {
		return fmt.Errorf("cannot clear %s: %w", stagingDir, err)
	}
	for _, file := range changed {
		staged := filepath.Join(stagingDir, filepath.FromSlash(file.Path))
		if err := release.ExtractFile(parsed, zipPath, file.Path, staged); err != nil {
			os.RemoveAll(stagingDir)
			return i18n.NewError("update.file_mismatch").WithCause(err)
		}
		mode, err := file.ParseMode()
		if err == nil {
			err = setMode(staged, os.FileMode(mode))
		}
		if err == nil && file.Path == release.ModPath {
			err = checkModArchive(staged)
			if err != nil {
				err = i18n.NewError("update.bad_mod").WithCause(err)
			}
		}
		if err != nil {
			os.RemoveAll(stagingDir)
			return err
		}
	}
	return nil
}

func (updater *Updater) swap(changed []release.File) error {
	for _, file := range changed {
		target := updater.layout.target(file.Path)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("cannot create folder for %s: %w", file.Path, err)
		}
		staged := filepath.Join(updater.layout.stagingDir(), filepath.FromSlash(file.Path))
		if err := os.Rename(staged, target); err != nil {
			return fmt.Errorf("cannot replace %s: %w", file.Path, err)
		}
	}
	return nil
}

// selfCheck proves the new files work: no installed app became invalid and a replaced appsctl runs.
func (updater *Updater) selfCheck(parsed *release.Manifest, changed []release.File, appsBefore map[string]bool) error {
	if err := updater.checkApps(appsBefore); err != nil {
		return err
	}
	if !replacedAppsctl(changed) {
		return nil
	}
	return updater.checkAppsctlVersion(parsed.Version)
}

func replacedAppsctl(changed []release.File) bool {
	for _, file := range changed {
		if file.Path == ".adds/koboapps/bin/appsctl" {
			return true
		}
	}
	return false
}

// invalidAppIDs lists apps that fail validation now, so an app that was already broken
// does not block updates.
func (updater *Updater) invalidAppIDs() map[string]bool {
	invalid := map[string]bool{}
	scan, err := manifest.LoadAll(updater.layout.appsDir())
	if err != nil {
		return invalid
	}
	for _, failure := range scan.Failures {
		invalid[failure.AppID] = true
	}
	return invalid
}

// checkApps only proves the update broke no installed app; a drawer update never installs or
// changes apps, so it does not expect any app version.
func (updater *Updater) checkApps(appsBefore map[string]bool) error {
	scan, err := manifest.LoadAll(updater.layout.appsDir())
	if err != nil {
		return fmt.Errorf("apps cannot be listed: %w", err)
	}
	for _, failure := range scan.Failures {
		if !appsBefore[failure.AppID] {
			return fmt.Errorf("app %s is invalid after the update: %s", failure.AppID, failure.Reason())
		}
	}
	return nil
}

func (updater *Updater) checkAppsctlVersion(wantVersion string) error {
	ctx, cancel := context.WithTimeout(context.Background(), selfCheckTimeout)
	defer cancel()
	output, err := exec.CommandContext(ctx, updater.layout.appsctlPath(), "--version").CombinedOutput()
	if err != nil {
		return fmt.Errorf("the new appsctl does not run: %w", err)
	}
	if !strings.Contains(string(output), wantVersion) {
		return fmt.Errorf("the new appsctl reports %q, expected version %s", strings.TrimSpace(string(output)), wantVersion)
	}
	return nil
}

// undo restores the backup after a failed swap or self-check and reports why the update failed.
func (updater *Updater) undo(backupDir string, cause error) error {
	if restoreErr := updater.layout.restore(backupDir); restoreErr != nil {
		return i18n.NewError("update.restore_failed", "dir", backupDir).WithCause(errors.Join(cause, restoreErr))
	}
	os.RemoveAll(backupDir) // restored, so it is no longer a rollback point
	return i18n.NewError("update.failed_restored").WithCause(cause)
}

// beginRun refuses a second run and records that this process is working.
func (updater *Updater) beginRun() error {
	status, found, err := ReadStatus(updater.layout)
	if err == nil && found && status.State == StateRunning && status.PID != os.Getpid() {
		return i18n.NewError("update.already_running")
	}
	return writeStatus(updater.layout, Status{State: StateRunning, PID: os.Getpid(), Report: report("update.running")})
}

func (updater *Updater) finishRun(result Report, err error) {
	status := Status{State: StateDone, Report: result}
	if err != nil {
		status = Status{State: StateFailed, Report: reportOfError(err)}
	}
	if writeErr := writeStatus(updater.layout, status); writeErr != nil {
		fmt.Fprintln(os.Stderr, "warning: cannot write update status:", writeErr)
	}
}

// beginCheck refuses a check while an update runs and shows "working" in the status until it
// ends. It returns the earlier status so a successful check can put it back.
func (updater *Updater) beginCheck() (previous Status, hadPrevious bool, err error) {
	previous, hadPrevious, err = ReadStatus(updater.layout)
	if err == nil && hadPrevious && previous.State == StateRunning && previous.PID != os.Getpid() {
		return Status{}, false, i18n.NewError("update.already_running")
	}
	return previous, hadPrevious, writeStatus(updater.layout, Status{State: StateRunning, Job: JobCheck, PID: os.Getpid(), Report: report("update.checking")})
}

// finishCheck puts the earlier status back, so "Updated to 1.1.0, restart to finish" is not
// replaced by the outcome of a check (that is in last-check.json). Without an earlier status
// the check's own outcome becomes the status, tagged so the settings screen skips it.
func (updater *Updater) finishCheck(previous Status, hadPrevious bool, result Report, err error) {
	status := previous
	switch {
	case hadPrevious:
	case err != nil:
		status = Status{State: StateFailed, Job: JobCheck, Report: reportOfError(err)}
	default:
		status = Status{State: StateDone, Job: JobCheck, Report: result}
	}
	if writeErr := writeStatus(updater.layout, status); writeErr != nil {
		fmt.Fprintln(os.Stderr, "warning: cannot write update status:", writeErr)
	}
}
