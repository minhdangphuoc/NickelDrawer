package main

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"koboapps/internal/manifest"
	"koboapps/internal/uikit"
)

func runValidate(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("validate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	appsDir := flags.String("apps-dir", defaultAppsDir(), "folder containing app folders (used when no APP_DIR is given)")
	if err := flags.Parse(args); err != nil {
		return exitFailure
	}
	if flags.NArg() > 0 {
		return reportDirs(flags.Args(), stdout)
	}
	return reportScan(*appsDir, stdout, stderr)
}

func reportDirs(dirs []string, stdout io.Writer) int {
	exitCode := exitOK
	for _, dir := range dirs {
		app, warnings, err := manifest.Load(dir)
		printWarnings(stdout, dir, warnings)
		if err != nil {
			fmt.Fprintf(stdout, "INVALID %s: %v\n", dir, err)
			exitCode = exitInvalid
			continue
		}
		if !reportApp(stdout, app) {
			exitCode = exitInvalid
		}
	}
	return exitCode
}

// reportApp prints the result for one loaded app: locales, then ui.json when present.
func reportApp(out io.Writer, app manifest.App) bool {
	if !reportLocales(out, app) {
		return false
	}
	hasUI, isUIValid := reportUI(out, app)
	if !isUIValid {
		return false
	}
	suffix := ""
	if hasUI {
		suffix = ", " + uikit.FileName
	}
	fmt.Fprintf(out, "ok      %s (%d actions%s)\n", app.ID, len(app.Actions), suffix)
	return true
}

func reportScan(appsDir string, stdout, stderr io.Writer) int {
	scan, err := manifest.LoadAll(appsDir)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitFailure
	}
	for _, warning := range scan.Warnings {
		fmt.Fprintf(stdout, "warning %s\n", warning)
	}
	exitCode := exitOK
	for _, app := range scan.Apps {
		if !reportApp(stdout, app) {
			exitCode = exitInvalid
		}
	}
	for _, failure := range scan.Failures {
		fmt.Fprintf(stdout, "INVALID %s: %v\n", failure.AppID, failure.Err)
		exitCode = exitInvalid
	}
	return exitCode
}

func printWarnings(out io.Writer, label string, warnings []manifest.Warning) {
	for _, warning := range warnings {
		fmt.Fprintf(out, "warning %s: %s\n", label, warning)
	}
}

// reportLocales prints locale warnings and problems; it returns false when the app has problems.
func reportLocales(out io.Writer, app manifest.App) bool {
	problems, warnings := manifest.CheckLocales(app)
	printWarnings(out, app.ID, warnings)
	if len(problems) > 0 {
		fmt.Fprintf(out, "INVALID %s: %s\n", app.ID, strings.Join(problems, "; "))
	}
	return len(problems) == 0
}
