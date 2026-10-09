package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"koboapps/internal/i18n"
	"koboapps/internal/manifest"
	"koboapps/internal/nmconfig"
)

type genMenuOptions struct {
	appsDir    string // where manifests are scanned (a mounted Kobo on a Mac)
	deviceApps string // apps folder as the Kobo sees it; written into menu commands
	deviceHost string // koboapps folder as the Kobo sees it
	menuFile   string
	errorsFile string
	localeFlags
}

type genMenuResult struct {
	appCount  int
	itemCount int
	failures  []manifest.Failure
	warnings  []string // broken locale files; the menu still renders in English for them
	bundle    *i18n.Bundle
}

func runGenMenu(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("gen-menu", flag.ContinueOnError)
	flags.SetOutput(stderr)
	options := genMenuOptions{}
	flags.StringVar(&options.appsDir, "apps-dir", defaultAppsDir(), "folder containing app folders")
	flags.StringVar(&options.deviceApps, "device-apps-root", defaultAppsDir(), "apps folder path on the Kobo, used in generated commands")
	flags.StringVar(&options.deviceHost, "device-koboapps-root", defaultKoboappsDir(), "koboapps folder path on the Kobo, used in generated commands")
	flags.StringVar(&options.menuFile, "out", defaultMenuFile, "NickelMenu config file to write")
	flags.StringVar(&options.errorsFile, "errors-file", defaultErrorsFile, "file receiving one line per invalid app")
	options.localeFlags.register(flags)
	if err := flags.Parse(args); err != nil {
		return exitFailure
	}
	result, err := generateMenu(options)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitFailure
	}
	printLocaleWarnings(stderr, result.warnings)
	fmt.Fprintln(stdout, result.bundle.Message("genmenu.summary", i18n.Params(
		"file", options.menuFile,
		"apps", strconv.Itoa(result.appCount),
		"items", strconv.Itoa(result.itemCount),
		"invalid", strconv.Itoa(len(result.failures)))))
	return exitOK
}

// generateMenu scans the apps folder, then writes the errors file before the menu,
// so the errors item never points at a stale list.
func generateMenu(options genMenuOptions) (genMenuResult, error) {
	scan, err := manifest.LoadAll(options.appsDir)
	if err != nil {
		return genMenuResult{}, err
	}
	locale := options.effectiveLocale()
	apps, warnings := localizeApps(scan.Apps, locale)
	bundle, bundleErr := options.hostBundle()
	if bundleErr != nil {
		warnings = append(warnings, bundleErr.Error())
	}
	paths := nmconfig.Paths{AppsRoot: options.deviceApps, KoboappsRoot: options.deviceHost}
	labels := nmconfig.Labels{
		Refresh:      bundle.Message("menu.refresh", nil),
		Errors:       bundle.Message("menu.errors", i18n.Params("count", strconv.Itoa(len(scan.Failures)))),
		UpdateCheck:  bundle.Message("menu.update_check", nil),
		UpdateStatus: bundle.Message("menu.update_status", nil),
	}
	config := nmconfig.Render(apps, len(scan.Failures), paths, labels, options.hasUpdateConfig())
	itemCount := nmconfig.ItemCount(scan.Apps)
	errorsText := formatErrorLines(scan.Failures)
	if itemCount > nmconfig.MaxAppItems {
		note := fmt.Sprintf("menu: %d app items exceed %d; NickelMenu cannot scroll. Hide actions with \"menu\": false in app.json.",
			itemCount, nmconfig.MaxAppItems)
		warnings = append(warnings, note)
		errorsText += note + "\n"
	}
	if err := writeFileAtomically(options.errorsFile, []byte(errorsText)); err != nil {
		return genMenuResult{}, err
	}
	if err := writeFileAtomically(options.menuFile, []byte(config)); err != nil {
		return genMenuResult{}, err
	}
	return genMenuResult{
		appCount: len(scan.Apps), itemCount: itemCount,
		failures: scan.Failures, warnings: warnings, bundle: bundle,
	}, nil
}

// hasUpdateConfig is true when update.json exists; without it the update menu items do nothing useful.
func (options genMenuOptions) hasUpdateConfig() bool {
	_, err := os.Stat(filepath.Join(options.hostDir, "data", "update.json"))
	return err == nil
}

func localizeApps(apps []manifest.App, locale string) (localized []manifest.App, warnings []string) {
	for _, app := range apps {
		bundle, err := app.LoadLocales(locale)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("%s: %v", app.ID, err))
		}
		localized = append(localized, app.Localize(bundle))
	}
	return localized, warnings
}

func printLocaleWarnings(stderr io.Writer, warnings []string) {
	for _, warning := range warnings {
		fmt.Fprintln(stderr, "warning:", warning)
	}
}

func formatErrorLines(failures []manifest.Failure) string {
	var text strings.Builder
	for _, failure := range failures {
		fmt.Fprintf(&text, "%s: %s\n", failure.AppID, failure.Reason())
	}
	return text.String()
}
