package main

import (
	"crypto/ed25519"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"koboapps/internal/release"
)

type signOptions struct {
	distRoot   string
	version    string
	outDir     string
	keyPath    string
	publicPath string
	minAppsctl string
	notesPath  string
}

func runSign(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("sign", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var options signOptions
	flags.StringVar(&options.distRoot, "dist", "dist/onboard", "dist tree that mirrors the Kobo onboard root")
	flags.StringVar(&options.version, "version", "", "release version MAJOR.MINOR.PATCH or MAJOR.MINOR.PATCH-prerelease (required)")
	flags.StringVar(&options.outDir, "out", "dist/release", "output folder")
	flags.StringVar(&options.keyPath, "key", defaultPrivateKeyPath(), "private key file")
	flags.StringVar(&options.publicPath, "pub", defaultPublicKey, "public key that appsctl embeds; the private key must match it")
	flags.StringVar(&options.minAppsctl, "min-appsctl", "0.1.0", "oldest appsctl allowed to apply this release")
	flags.StringVar(&options.notesPath, "notes-file", "", "text file with the changelog summary the drawer's update screen shows (optional)")
	if err := flags.Parse(args); err != nil {
		return exitUsage
	}
	if err := sign(options); err != nil {
		return fail(stderr, err)
	}
	fmt.Fprintf(stdout, "signed release %s in %s\n", options.version, options.outDir)
	return exitOK
}

func sign(options signOptions) error {
	if !release.ValidVersion(options.version) {
		return fmt.Errorf("--version %q must look like MAJOR.MINOR.PATCH or 1.0.0-beta.1", options.version)
	}
	privateKey, err := loadPrivateKey(options.keyPath)
	if err != nil {
		return err
	}
	if err := requireMatchingPublicKey(privateKey, options.publicPath); err != nil {
		return err
	}
	manifest, err := release.BuildManifest(options.distRoot, options.version, options.minAppsctl, time.Now())
	if err != nil {
		return err
	}
	if manifest.Notes, err = readNotes(options.notesPath); err != nil {
		return err
	}
	if err := manifest.Validate(); err != nil {
		return fmt.Errorf("release notes are invalid: %w", err)
	}
	manifestBytes, err := manifest.Marshal()
	if err != nil {
		return err
	}
	if err := writeReleaseFiles(options, manifest, manifestBytes, release.Sign(privateKey, manifestBytes)); err != nil {
		return err
	}
	return verifyReleaseDir(options.outDir, options.publicPath)
}

// readNotes returns the trimmed notes file, or "" when no file is given.
func readNotes(notesPath string) (string, error) {
	if notesPath == "" {
		return "", nil
	}
	text, err := os.ReadFile(notesPath)
	if err != nil {
		return "", fmt.Errorf("cannot read %s: %w", notesPath, err)
	}
	return strings.TrimSpace(string(text)), nil
}

func loadPrivateKey(keyPath string) (ed25519.PrivateKey, error) {
	info, err := os.Stat(keyPath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("no signing key at %s; run `ndrelease keygen` first", keyPath)
	}
	if err != nil {
		return nil, fmt.Errorf("cannot inspect %s: %w", keyPath, err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("%s is readable by others (mode %o); run `chmod 600 %s`", keyPath, info.Mode().Perm(), keyPath)
	}
	text, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", keyPath, err)
	}
	return release.ParsePrivateKey(string(text))
}

// requireMatchingPublicKey stops a release that devices built from this source would refuse.
func requireMatchingPublicKey(privateKey ed25519.PrivateKey, publicPath string) error {
	publicKey, err := loadPublicKey(publicPath)
	if err != nil {
		return err
	}
	if !publicKey.Equal(privateKey.Public()) {
		return fmt.Errorf("the private key does not match %s; appsctl would refuse this release", publicPath)
	}
	return nil
}

func loadPublicKey(publicPath string) (ed25519.PublicKey, error) {
	text, err := os.ReadFile(publicPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", publicPath, err)
	}
	publicKey, err := release.ParsePublicKey(string(text))
	if err == release.ErrNoPublicKey {
		return nil, fmt.Errorf("%s is still a placeholder; run `ndrelease keygen` first", publicPath)
	}
	return publicKey, err
}

func writeReleaseFiles(options signOptions, manifest *release.Manifest, manifestBytes []byte, signature string) error {
	if err := os.MkdirAll(options.outDir, 0o755); err != nil {
		return fmt.Errorf("cannot create %s: %w", options.outDir, err)
	}
	zipPath := filepath.Join(options.outDir, zipName(options.version))
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("cannot create %s: %w", zipPath, err)
	}
	if err := release.WriteZip(options.distRoot, manifest, zipFile); err != nil {
		zipFile.Close()
		return err
	}
	if err := zipFile.Close(); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(options.outDir, release.ManifestName), manifestBytes, 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(options.outDir, release.SignatureName), []byte(signature), 0o644)
}

func zipName(version string) string {
	return releaseFilePrefix + version + ".zip"
}
