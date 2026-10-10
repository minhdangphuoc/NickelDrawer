package update

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"koboapps/internal/i18n"
	"koboapps/internal/release"
)

const (
	stagedModRelativePath = ".kobo/KoboRoot.tgz"
	modLibraryFolder      = "usr/local/Kobo/imageformats"
)

func (layout Layout) stagedModPath() string { return layout.target(stagedModRelativePath) }

// installedModPath is the copy of the mod archive the last update (or the installer) staged.
// It records which mod version Nickel was last asked to install; Nickel deletes the staged
// KoboRoot.tgz itself after the reboot.
func (layout Layout) installedModPath() string { return layout.target(release.ModPath) }

// findFile returns the changed file with the given manifest path.
func findFile(files []release.File, manifestPath string) (release.File, bool) {
	for _, file := range files {
		if file.Path == manifestPath {
			return file, true
		}
	}
	return release.File{}, false
}

// ensureModSlotFree refuses an update whose mod cannot be staged: Nickel installs one
// KoboRoot.tgz per reboot, so a different archive that is still waiting must not be replaced.
func (layout Layout) ensureModSlotFree(mod release.File) error {
	staged, err := hashIfExists(layout.stagedModPath(), release.HashFile)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", layout.stagedModPath(), err)
	}
	if staged != "" && staged != mod.SHA256 {
		return i18n.NewError("update.mod_busy")
	}
	return nil
}

// checkModArchive proves the archive installs nothing but mod libraries under
// usr/local/Kobo/imageformats, the only place a NickelHook mod lives. Even a signed release
// must not be able to make Nickel overwrite other system files.
func checkModArchive(archivePath string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("cannot open %s: %w", archivePath, err)
	}
	defer file.Close()
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("the mod archive is not a gzip file: %w", err)
	}
	defer gzipReader.Close()
	reader := tar.NewReader(gzipReader)
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("the mod archive is damaged: %w", err)
		}
		if !isAllowedModEntry(header) {
			return fmt.Errorf("the mod archive contains %q; only %s/*.so is allowed", header.Name, modLibraryFolder)
		}
	}
}

func isAllowedModEntry(header *tar.Header) bool {
	name := strings.TrimSuffix(strings.TrimPrefix(header.Name, "./"), "/")
	if header.Typeflag == tar.TypeDir {
		return name == "" || name == "usr" || name == "usr/local" || name == "usr/local/Kobo" || name == modLibraryFolder
	}
	if header.Typeflag != tar.TypeReg {
		return false
	}
	return path.Dir(name) == modLibraryFolder && strings.HasSuffix(name, ".so") && path.Base(name) != ".so"
}

// stageMod copies the installed mod archive to .kobo/KoboRoot.tgz, where Nickel picks it up
// at the next reboot.
func (layout Layout) stageMod() error {
	if err := copyFileAtomically(layout.installedModPath(), layout.stagedModPath(), 0o644); err != nil {
		return i18n.NewError("update.mod_stage_failed").WithCause(err)
	}
	return nil
}

// dropStagedModIfSameAs removes a staged KoboRoot.tgz that equals the archive about to be
// rolled back, so a rollback does not leave the newer mod waiting for the next reboot.
func (layout Layout) dropStagedModIfSameAs(archivePath string) error {
	staged, err := hashIfExists(layout.stagedModPath(), release.HashFile)
	if err != nil || staged == "" {
		return err
	}
	installed, err := hashIfExists(archivePath, release.HashFile)
	if err != nil || installed != staged {
		return err
	}
	if err := os.Remove(layout.stagedModPath()); err != nil {
		return fmt.Errorf("cannot remove %s: %w", filepath.ToSlash(layout.stagedModPath()), err)
	}
	return nil
}
