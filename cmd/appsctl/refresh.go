package main

import (
	"flag"
	"fmt"
	"io"
	"strconv"

	"koboapps/internal/i18n"
)

// runRefresh is what the "Refresh list" menu item runs; its stdout is shown in a dialog.
func runRefresh(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("refresh", flag.ContinueOnError)
	flags.SetOutput(stderr)
	options := deviceGenMenuOptions()
	options.localeFlags.register(flags)
	if err := flags.Parse(args); err != nil {
		return exitFailure
	}
	result, err := generateMenu(options)
	if err != nil {
		bundle, _ := options.hostBundle()
		fmt.Fprintln(stdout, bundle.Message("refresh.failed", nil))
		fmt.Fprintln(stderr, err)
		return exitFailure
	}
	printLocaleWarnings(stderr, result.warnings)
	fmt.Fprint(stdout, refreshSummary(result))
	return exitOK
}

func deviceGenMenuOptions() genMenuOptions {
	return genMenuOptions{
		appsDir:    defaultAppsDir(),
		deviceApps: defaultAppsDir(),
		deviceHost: defaultKoboappsDir(),
		menuFile:   defaultMenuFile,
		errorsFile: defaultErrorsFile,
	}
}

func refreshSummary(result genMenuResult) string {
	summary := result.bundle.Message("refresh.summary", i18n.Params(
		"apps", strconv.Itoa(result.appCount), "items", strconv.Itoa(result.itemCount))) + "\n"
	if len(result.failures) > 0 {
		summary += result.bundle.Message("refresh.failures", i18n.Params("count", strconv.Itoa(len(result.failures)))) + "\n"
	}
	return summary
}
