package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"koboapps/internal/release"
)

const defaultAppsReleaseDir = "dist/apps-release"

// idList is a flag that takes ids repeatedly or comma separated: --only a,b --only c.
type idList []string

func (list *idList) String() string { return strings.Join(*list, ",") }

func (list *idList) Set(value string) error {
	for _, id := range strings.Split(value, ",") {
		if id = strings.TrimSpace(id); id != "" {
			*list = append(*list, id)
		}
	}
	return nil
}

type signAppsOptions struct {
	appsDir, outDir, keyPath, publicPath string
	sequence                             int64
	only                                 idList
}

// runSignApps signs a catalog of every app zip in --apps-dir. The catalog always lists all
// current app versions; --only picks the zips that are new and must be uploaded with it.
func runSignApps(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("sign-apps", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var options signAppsOptions
	flags.StringVar(&options.appsDir, "apps-dir", "dist/apps", "folder with the app-<id>-<version>.zip files (make dist)")
	flags.StringVar(&options.outDir, "out", defaultAppsReleaseDir, "output folder")
	flags.StringVar(&options.keyPath, "key", defaultPrivateKeyPath(), "private key file")
	flags.StringVar(&options.publicPath, "pub", defaultPublicKey, "public key that appsctl embeds; the private key must match it")
	flags.Int64Var(&options.sequence, "sequence", 0, "catalog sequence number: one more than the last published catalog (required)")
	flags.Var(&options.only, "only", "app id whose zip is copied to --out for upload (repeatable; default: every app)")
	if err := flags.Parse(args); err != nil {
		return exitUsage
	}
	if err := signApps(options); err != nil {
		return fail(stderr, err)
	}
	fmt.Fprintf(stdout, "signed app catalog, sequence %d, in %s\n", options.sequence, options.outDir)
	return exitOK
}

func signApps(options signAppsOptions) error {
	if options.sequence < 1 {
		return fmt.Errorf("--sequence is required and must be 1 or more; use one more than the catalog now published")
	}
	privateKey, err := loadPrivateKey(options.keyPath)
	if err != nil {
		return err
	}
	if err := requireMatchingPublicKey(privateKey, options.publicPath); err != nil {
		return err
	}
	catalog, err := release.BuildCatalog(options.appsDir, options.sequence, time.Now())
	if err != nil {
		return err
	}
	toUpload, err := zipsToUpload(catalog, options.only)
	if err != nil {
		return err
	}
	catalogBytes, err := release.MarshalCatalog(catalog)
	if err != nil {
		return err
	}
	if err := writeAppsRelease(options, toUpload, catalogBytes, release.SignCatalog(privateKey, catalogBytes)); err != nil {
		return err
	}
	return verifyAppsRelease(options.outDir, options.appsDir, options.publicPath)
}

// zipsToUpload names the zips of --only, or of every app when none is given.
func zipsToUpload(catalog *release.Catalog, only idList) ([]release.CatalogApp, error) {
	if len(only) == 0 {
		return catalog.Apps, nil
	}
	var chosen []release.CatalogApp
	for _, id := range only {
		app, found := catalog.Find(id)
		if !found {
			return nil, fmt.Errorf("--only %s: no zip for that app in the apps folder", id)
		}
		chosen = append(chosen, app)
	}
	return chosen, nil
}

func writeAppsRelease(options signAppsOptions, toUpload []release.CatalogApp, catalogBytes []byte, signature string) error {
	if err := os.RemoveAll(options.outDir); err != nil {
		return fmt.Errorf("cannot clear %s: %w", options.outDir, err)
	}
	if err := os.MkdirAll(options.outDir, 0o755); err != nil {
		return fmt.Errorf("cannot create %s: %w", options.outDir, err)
	}
	for _, app := range toUpload {
		data, err := os.ReadFile(filepath.Join(options.appsDir, app.Zip))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(options.outDir, app.Zip), data, 0o644); err != nil {
			return err
		}
	}
	// The signature file is written last, so an interrupted run never leaves a signature
	// that belongs to no catalog.
	if err := os.WriteFile(filepath.Join(options.outDir, release.CatalogName), catalogBytes, 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(options.outDir, release.CatalogSignatureName), []byte(signature), 0o644)
}

func runVerifyApps(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("verify-apps", flag.ContinueOnError)
	flags.SetOutput(stderr)
	dir := flags.String("dir", defaultAppsReleaseDir, "folder holding apps.json, its signature and the zips to upload")
	appsDir := flags.String("apps-dir", "", "also require a matching zip here for every catalog entry (make dist output)")
	publicPath := flags.String("pub", defaultPublicKey, "public key file")
	if err := flags.Parse(args); err != nil {
		return exitUsage
	}
	if err := verifyAppsRelease(*dir, *appsDir, *publicPath); err != nil {
		return fail(stderr, err)
	}
	fmt.Fprintf(stdout, "%s verifies: catalog signature and every zip it holds\n", *dir)
	return exitOK
}

// verifyAppsRelease checks the signature first, then each catalog entry whose zip is in dir.
// With appsDir set, every entry must have a matching zip there.
func verifyAppsRelease(dir, appsDir, publicPath string) error {
	publicKey, err := loadPublicKey(publicPath)
	if err != nil {
		return err
	}
	catalogBytes, err := os.ReadFile(filepath.Join(dir, release.CatalogName))
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", release.CatalogName, err)
	}
	signature, err := os.ReadFile(filepath.Join(dir, release.CatalogSignatureName))
	if err != nil {
		return fmt.Errorf("cannot read the catalog signature: %w", err)
	}
	if err := release.VerifyCatalog(publicKey, catalogBytes, string(signature)); err != nil {
		return err
	}
	catalog, err := release.ParseCatalog(catalogBytes)
	if err != nil {
		return err
	}
	for _, app := range catalog.Apps {
		if err := verifyCatalogEntry(app, dir, appsDir); err != nil {
			return err
		}
	}
	return nil
}

// verifyCatalogEntry checks the entry against the upload folder's zip when one is there (a
// refresh may upload only the new zips) and, with appsDir, requires and checks the full set.
func verifyCatalogEntry(app release.CatalogApp, dir, appsDir string) error {
	if err := verifyZipIfPresent(app, filepath.Join(dir, app.Zip)); err != nil {
		return err
	}
	if appsDir == "" {
		return nil
	}
	return release.VerifyCatalogZip(app, filepath.Join(appsDir, app.Zip))
}

func verifyZipIfPresent(app release.CatalogApp, zipPath string) error {
	if _, err := os.Stat(zipPath); os.IsNotExist(err) {
		return nil
	}
	return release.VerifyCatalogZip(app, zipPath)
}
