package update

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"koboapps/internal/i18n"
	"koboapps/internal/release"
)

// spaceMargin is kept free on top of what an install needs, so the storage never fills up.
const spaceMargin = 2 << 20

// freeBytes is the space a normal user can still write in the folder's file system.
func freeBytes(folder string) (uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(folder, &stat); err != nil {
		return 0, err
	}
	return uint64(stat.Bavail) * uint64(stat.Bsize), nil //nolint:unconvert // field types differ by platform
}

// requireSpace fails with the localized no-space error when folder has less than needed free.
func requireSpace(folder string, needed uint64, appID string) error {
	free, err := freeBytes(folder)
	if err != nil {
		return nil // the free space is unknown; the write itself reports a full disk
	}
	if free < needed+spaceMargin {
		return appError("no_space", "app", appID)
	}
	return nil
}

func isDiskFull(err error) bool { return errors.Is(err, syscall.ENOSPC) }

// downloadAppZip fetches an app's zip (one byte over the signed size is read, so a longer file
// is caught as a size mismatch) and checks it against the signed catalog entry: exact
// size first, then sha256, both before the zip is opened.
func (updater *Updater) downloadAppZip(ctx context.Context, repo string, app release.CatalogApp, workDir string) (string, error) {
	if err := requireSpace(workDir, uint64(app.Size), app.ID); err != nil {
		return "", err
	}
	zipPath := filepath.Join(workDir, app.Zip)
	url := updater.downloadBase() + "/" + repo + "/" + app.DownloadPath()
	if err := updater.github.downloadToFile(ctx, url, zipPath, app.Size+1); err != nil {
		if isDiskFull(err) {
			return "", appError("no_space", "app", app.ID)
		}
		return "", appFetchError(err, app.ID)
	}
	if err := verifyDownloadedZip(app, zipPath); err != nil {
		os.Remove(zipPath)
		return "", err
	}
	return zipPath, nil
}

func verifyDownloadedZip(app release.CatalogApp, zipPath string) error {
	info, err := os.Stat(zipPath)
	if err != nil {
		return fmt.Errorf("cannot inspect the download: %w", err)
	}
	if info.Size() != app.Size {
		return appError("hash_mismatch", "app", app.ID).WithCause(fmt.Errorf("downloaded %d bytes, the catalog says %d", info.Size(), app.Size))
	}
	hash, err := hashOfFile(zipPath)
	if err != nil {
		return err
	}
	if hash != app.SHA256 {
		return appError("hash_mismatch", "app", app.ID).WithCause(errors.New("sha256 differs from the signed catalog"))
	}
	return nil
}

func hashOfFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// stageAppZip checks the zip against the safety rules and the catalog entry, then unpacks it
// into the staged folder next to the app (same file system, so the final move is a rename).
func (layout Layout) stageAppZip(app release.CatalogApp, zipPath string) error {
	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		return appError("unsafe_zip", "app", app.ID).WithCause(err)
	}
	defer archive.Close()
	entries, err := release.CheckAppZip(&archive.Reader, app.ID)
	if err != nil {
		return appError("unsafe_zip", "app", app.ID).WithCause(err)
	}
	if err := requireSpace(layout.appsDir(), unpackedSize(entries), app.ID); err != nil {
		return err
	}
	stagedDir := layout.stagedAppDir(app.ID)
	if err := removeIfNoData(stagedDir); err != nil {
		return err
	}
	if err := unpackEntries(entries, stagedDir); err != nil {
		os.RemoveAll(stagedDir)
		if isDiskFull(err) {
			return appError("no_space", "app", app.ID)
		}
		return appError("unsafe_zip", "app", app.ID).WithCause(err)
	}
	return nil
}

func unpackedSize(entries []release.AppZipEntry) uint64 {
	var total uint64
	for _, entry := range entries {
		total += uint64(entry.Size)
	}
	return total
}

func unpackEntries(entries []release.AppZipEntry, stagedDir string) error {
	if err := os.MkdirAll(stagedDir, 0o755); err != nil {
		return err
	}
	for _, entry := range entries {
		target := filepath.Join(stagedDir, filepath.FromSlash(entry.Name))
		var err error
		if entry.IsDir {
			err = os.MkdirAll(target, 0o755)
		} else {
			err = unpackFile(entry, target)
		}
		if err != nil {
			return fmt.Errorf("%s: %w", entry.Name, err)
		}
	}
	return nil
}

// unpackFile writes one file, reading at most one byte more than its declared size so a zip
// that lies about its size cannot fill the storage.
func unpackFile(entry release.AppZipEntry, target string) (err error) {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	source, err := entry.File.Open()
	if err != nil {
		return err
	}
	defer source.Close()
	output, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := output.Close(); err == nil {
			err = closeErr
		}
	}()
	written, err := io.Copy(output, io.LimitReader(source, entry.Size+1))
	if err != nil {
		return err
	}
	if written != entry.Size {
		return fmt.Errorf("holds %d bytes but declares %d", written, entry.Size)
	}
	return setMode(target, entry.ModeOf())
}

// appError builds a localized error of the app updater ("appupdate.error.<code>").
func appError(code string, pairs ...string) *i18n.MessageError {
	return i18n.NewError("appupdate.error."+code, pairs...)
}

// appFetchError names a download failure by what the user can do: connect, or try later.
func appFetchError(err error, appID string) error {
	var messageErr *i18n.MessageError
	if errors.As(err, &messageErr) {
		switch messageErr.Key {
		case "update.no_network":
			return appError("no_network", "app", appID).WithCause(err)
		case "update.rate_limited":
			return appError("rate_limited", "app", appID).WithCause(err)
		}
	}
	return appError("unreachable", "app", appID).WithCause(err)
}
