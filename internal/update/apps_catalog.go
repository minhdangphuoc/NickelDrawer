package update

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"koboapps/internal/release"
)

const (
	defaultDownloadBase = "https://github.com"
	maxCatalogSignature = 1 << 10
)

func (updater *Updater) downloadBase() string {
	if updater.options.DownloadBase != "" {
		return updater.options.DownloadBase
	}
	return defaultDownloadBase
}

// verifiedCatalog is the catalog after its signature and the anti-rollback rule passed.
type verifiedCatalog struct {
	catalog *release.Catalog
	raw     []byte
	repo    string
}

// fetchCatalog downloads apps.json and its signature from the fixed `apps` release, verifies
// the signature over the raw bytes before parsing them, and refuses a catalog whose sequence is
// older than one this device already accepted. Plain downloads, no GitHub API.
func (updater *Updater) fetchCatalog(ctx context.Context) (verifiedCatalog, error) {
	config, err := LoadConfig(updater.layout)
	if err != nil {
		return verifiedCatalog{}, err
	}
	key, err := updater.trustedKey()
	if err != nil {
		return verifiedCatalog{}, err
	}
	base := updater.downloadBase() + "/" + config.Repo + "/releases/download/" + release.CatalogTag + "/"
	raw, err := updater.fetchCatalogFile(ctx, base+release.CatalogName, release.MaxCatalogBytes)
	if err != nil {
		return verifiedCatalog{}, err
	}
	signature, err := updater.fetchCatalogFile(ctx, base+release.CatalogSignatureName, maxCatalogSignature)
	if err != nil {
		return verifiedCatalog{}, err
	}
	if err := release.VerifyCatalog(key, raw, string(signature)); err != nil {
		return verifiedCatalog{}, appError("bad_signature").WithCause(err)
	}
	catalog, err := release.ParseCatalog(raw)
	if err != nil {
		return verifiedCatalog{}, appError("bad_catalog").WithCause(err)
	}
	if known := readAppsSequence(updater.layout); catalog.Sequence < known {
		return verifiedCatalog{}, appError("stale_catalog").WithCause(
			fmt.Errorf("catalog sequence %d is older than the accepted %d", catalog.Sequence, known))
	}
	return verifiedCatalog{catalog: catalog, raw: raw, repo: config.Repo}, nil
}

func (updater *Updater) fetchCatalogFile(ctx context.Context, url string, limit int64) ([]byte, error) {
	body, status, err := updater.github.get(ctx, url, limit)
	if err != nil {
		return nil, appFetchError(err, "")
	}
	switch status {
	case http.StatusOK:
		return body, nil
	case http.StatusNotFound:
		return nil, appError("no_catalog")
	}
	return nil, appError("unreachable").WithCause(fmt.Errorf("%s answered HTTP %d", url, status))
}

// acceptCatalog keeps the verified catalog for the screen and remembers its sequence.
func (updater *Updater) acceptCatalog(found verifiedCatalog) error {
	if err := WriteFileAtomically(updater.layout.appsCatalogFile(), found.raw, 0o644); err != nil {
		return fmt.Errorf("cannot save the app catalog: %w", err)
	}
	if found.catalog.Sequence > readAppsSequence(updater.layout) {
		return writeAppsSequence(updater.layout, found.catalog.Sequence)
	}
	return nil
}

// CheckApps fetches and verifies the catalog and records it; nothing is installed. It does not
// depend on the drawer check: one can fail while the other works.
func (updater *Updater) CheckApps(ctx context.Context) (result Report, err error) {
	finish, err := updater.beginAppsJob(AppsJobCheck, 0)
	if err != nil {
		return Report{}, err
	}
	defer func() { finish(result, err) }()
	found, err := updater.fetchCatalog(ctx)
	if err == nil {
		err = updater.acceptCatalog(found)
	}
	updater.recordAppsCheck(err)
	if err != nil {
		return Report{}, err
	}
	entries := AppEntries(found.catalog, InstalledAppVersions(updater.layout), updater.options.Language)
	return report("appupdate.checked", "updates", fmt.Sprint(countState(entries, AppStateUpdate))), nil
}

func countState(entries []AppEntry, state AppState) int {
	count := 0
	for _, entry := range entries {
		if entry.State == state {
			count++
		}
	}
	return count
}

func (updater *Updater) recordAppsCheck(err error) {
	record := AppsCheckRecord{Result: CheckUpToDate}
	if err != nil {
		failure := appFailure(err)
		record = AppsCheckRecord{Result: CheckFailed, Report: &failure}
	}
	if writeErr := writeAppsCheck(updater.layout, record); writeErr != nil {
		fmt.Fprintln(os.Stderr, "warning: cannot write the app check outcome:", writeErr)
	}
}
