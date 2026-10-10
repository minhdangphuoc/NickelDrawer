package main

import (
	"context"
	"fmt"
	"io"

	"koboapps/internal/i18n"
	"koboapps/internal/update"
)

// runUpdateApps checks the app catalog and lists every catalog app against what is installed;
// with --all it installs the available updates instead.
func runUpdateApps(options updateFlags, stdout, stderr io.Writer) int {
	ctx, cancel := context.WithTimeout(context.Background(), updateTimeout)
	defer cancel()
	updater := options.newUpdater()
	if options.updateAll {
		result, err := updater.ApplyAllApps(ctx)
		return printAppsResult(options, updater, result, err, stdout, stderr)
	}
	result, err := updater.CheckApps(ctx)
	if code := printAppsResult(options, updater, result, err, stdout, stderr); code != exitOK {
		return code
	}
	printAppsList(options, stdout)
	return exitOK
}

// runUpdateApp installs or updates one catalog app.
func runUpdateApp(options updateFlags, stdout, stderr io.Writer) int {
	if len(options.positions) != 1 {
		fmt.Fprintln(stderr, "usage: appsctl update app <id> [flags]")
		return exitFailure
	}
	ctx, cancel := context.WithTimeout(context.Background(), updateTimeout)
	defer cancel()
	updater := options.newUpdater()
	result, err := updater.ApplyApp(ctx, options.positions[0])
	return printAppsResult(options, updater, result, err, stdout, stderr)
}

// runUpdateRollbackApp restores the version an app had before its last update.
func runUpdateRollbackApp(options updateFlags, stdout, stderr io.Writer) int {
	if len(options.positions) != 1 {
		fmt.Fprintln(stderr, "usage: appsctl update rollback-app <id> [flags]")
		return exitFailure
	}
	updater := options.newUpdater()
	result, err := updater.RollbackApp(options.positions[0])
	return printAppsResult(options, updater, result, err, stdout, stderr)
}

// printAppsResult is printUpdateResult for app jobs: messages name the app.
func printAppsResult(options updateFlags, updater *update.Updater, result update.Report, err error, stdout, stderr io.Writer) int {
	bundle := hostBundleOrWarn(options.localeFlags, stderr)
	if err != nil {
		fmt.Fprintln(stdout, updater.DescribeAppError(bundle, err))
		fmt.Fprintln(stderr, err)
		return exitFailure
	}
	fmt.Fprintln(stdout, updater.RenderAppReport(bundle, result))
	return exitOK
}

// printAppsList writes one line per catalog app: update, installed or not installed.
func printAppsList(options updateFlags, stdout io.Writer) {
	layout := update.Layout{OnboardRoot: options.onboardRoot}
	bundle, _ := options.hostBundle()
	catalog, found := update.ReadAppsCatalog(layout)
	if !found {
		return
	}
	for _, entry := range update.AppEntries(catalog, update.InstalledAppVersions(layout), options.languageCode()) {
		fmt.Fprintln(stdout, bundle.Message("appupdate.cli_"+string(entry.State), appEntryParams(entry)))
	}
}

func appEntryParams(entry update.AppEntry) map[string]string {
	return i18n.Params("name", entry.Name, "id", entry.ID, "installed", entry.Installed, "latest", entry.Latest)
}
