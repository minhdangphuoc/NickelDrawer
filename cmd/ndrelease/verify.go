package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"koboapps/internal/release"
)

func runVerify(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("verify", flag.ContinueOnError)
	flags.SetOutput(stderr)
	dir := flags.String("dir", "dist/release", "folder holding release.json, its signature and the zip")
	publicPath := flags.String("pub", defaultPublicKey, "public key file")
	if err := flags.Parse(args); err != nil {
		return exitUsage
	}
	if err := verifyReleaseDir(*dir, *publicPath); err != nil {
		return fail(stderr, err)
	}
	fmt.Fprintf(stdout, "%s verifies: signature, every file and app zip sha256 and size\n", *dir)
	return exitOK
}

// verifyReleaseDir checks the detached signature and then the zip against the signed manifest.
func verifyReleaseDir(dir, publicPath string) error {
	publicKey, err := loadPublicKey(publicPath)
	if err != nil {
		return err
	}
	manifestBytes, err := os.ReadFile(filepath.Join(dir, release.ManifestName))
	if err != nil {
		return fmt.Errorf("cannot read release.json: %w", err)
	}
	signature, err := os.ReadFile(filepath.Join(dir, release.SignatureName))
	if err != nil {
		return fmt.Errorf("cannot read the signature: %w", err)
	}
	if err := release.Verify(publicKey, manifestBytes, string(signature)); err != nil {
		return err
	}
	manifest, err := release.Parse(manifestBytes)
	if err != nil {
		return err
	}
	if err := release.VerifyZip(manifest, filepath.Join(dir, zipName(manifest.Version))); err != nil {
		return err
	}
	return release.VerifyAppPackages(manifest, dir)
}
