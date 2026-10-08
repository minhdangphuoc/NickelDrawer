package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"time"

	// Embedded CA roots: the Kobo may lack a system CA bundle, and updates need TLS.
	_ "golang.org/x/crypto/x509roots/fallback"

	"koboapps/internal/i18n"
	"koboapps/internal/update"
)

const (
	defaultOnboardRoot = "/mnt/onboard"
	defaultAPIBase     = "https://api.github.com"
	defaultTempDir     = "/tmp"
	updateTimeout      = 20 * time.Minute
)

// updateFlags are shared by the update subcommands. --onboard-root moves every default path
// (conf, koboapps folder, menu file) so tests and a Mac can point at a temp tree.
type updateFlags struct {
	onboardRoot    string
	apiBase        string
	tempDir        string
	allowDowngrade bool
	localeFlags
}

func (f *updateFlags) register(flags *flag.FlagSet) {
	flags.StringVar(&f.onboardRoot, "onboard-root", defaultOnboardRoot, "Kobo onboard storage root")
	flags.StringVar(&f.apiBase, "api-base", defaultAPIBase, "GitHub API base URL (tests)")
	flags.StringVar(&f.tempDir, "tmp-dir", defaultTempDir, "folder for downloads; keep it off the onboard storage")
	flags.BoolVar(&f.allowDowngrade, "allow-downgrade", false, "apply a release older than the installed one")
	f.localeFlags.register(flags)
}

// resolve derives the locale defaults from --onboard-root unless the flag was given.
func (f *updateFlags) resolve(flags *flag.FlagSet) {
	given := map[string]bool{}
	flags.Visit(func(visited *flag.Flag) { given[visited.Name] = true })
	if !given["conf"] {
		f.confPath = filepath.Join(f.onboardRoot, ".kobo", "Kobo", "Kobo eReader.conf")
	}
	if !given["koboapps-dir"] {
		f.hostDir = filepath.Join(f.onboardRoot, ".adds", "koboapps")
	}
}

func (f updateFlags) menuOptions() genMenuOptions {
	root := f.onboardRoot
	return genMenuOptions{
		appsDir:     filepath.Join(root, ".adds", "apps"),
		deviceApps:  defaultAppsDir(),
		deviceHost:  defaultKoboappsDir(),
		menuFile:    filepath.Join(root, ".adds", "nm", "koboapps"),
		errorsFile:  filepath.Join(root, ".adds", "koboapps", "data", "errors.txt"),
		localeFlags: f.localeFlags,
	}
}

func (f updateFlags) newUpdater() *update.Updater {
	return update.New(update.Options{
		OnboardRoot:    f.onboardRoot,
		APIBase:        f.apiBase,
		TempDir:        f.tempDir,
		CurrentVersion: version,
		AllowDowngrade: f.allowDowngrade,
		RegenerateMenu: func() error {
			_, err := generateMenu(f.menuOptions())
			return err
		},
	})
}

const updateUsage = "usage: appsctl update check|apply|status|rollback [flags]\n"

func runUpdate(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, updateUsage)
		return exitFailure
	}
	actions := map[string]func(updateFlags, io.Writer, io.Writer) int{
		"check":    runUpdateCheck,
		"apply":    runUpdateApply,
		"rollback": runUpdateRollback,
		"status":   runUpdateStatus,
	}
	action, found := actions[args[0]]
	if !found {
		fmt.Fprintf(stderr, "unknown update command %q\n%s", args[0], updateUsage)
		return exitFailure
	}
	flags := flag.NewFlagSet("update "+args[0], flag.ContinueOnError)
	flags.SetOutput(stderr)
	var options updateFlags
	options.register(flags)
	if err := flags.Parse(args[1:]); err != nil {
		return exitFailure
	}
	options.resolve(flags)
	return action(options, stdout, stderr)
}

func runUpdateCheck(options updateFlags, stdout, stderr io.Writer) int {
	ctx, cancel := context.WithTimeout(context.Background(), updateTimeout)
	defer cancel()
	result, err := options.newUpdater().Check(ctx)
	return printUpdateResult(options, result, err, stdout, stderr)
}

func runUpdateApply(options updateFlags, stdout, stderr io.Writer) int {
	ctx, cancel := context.WithTimeout(context.Background(), updateTimeout)
	defer cancel()
	result, err := options.newUpdater().Apply(ctx)
	return printUpdateResult(options, result, err, stdout, stderr)
}

func runUpdateRollback(options updateFlags, stdout, stderr io.Writer) int {
	ctx, cancel := context.WithTimeout(context.Background(), updateTimeout)
	defer cancel()
	result, err := options.newUpdater().Rollback(ctx)
	return printUpdateResult(options, result, err, stdout, stderr)
}

// printUpdateResult shows the localized outcome on stdout (a dialog on the Kobo) and the
// technical error on stderr.
func printUpdateResult(options updateFlags, result update.Report, err error, stdout, stderr io.Writer) int {
	bundle := hostBundleOrWarn(options.localeFlags, stderr)
	if err != nil {
		fmt.Fprintln(stdout, bundle.Describe(err))
		fmt.Fprintln(stderr, err)
		return exitFailure
	}
	fmt.Fprintln(stdout, result.Render(bundle))
	return exitOK
}

// runUpdateStatus prints the installed version and the outcome of the last update run.
func runUpdateStatus(options updateFlags, stdout, stderr io.Writer) int {
	bundle := hostBundleOrWarn(options.localeFlags, stderr)
	fmt.Fprintln(stdout, bundle.Message("update.status_version", i18n.Params("version", version)))
	status, found, err := update.ReadStatus(update.Layout{OnboardRoot: options.onboardRoot})
	if err != nil {
		fmt.Fprintln(stdout, bundle.Describe(i18n.NewError("update.failed").WithCause(err)))
		fmt.Fprintln(stderr, err)
		return exitFailure
	}
	if !found {
		fmt.Fprintln(stdout, bundle.Message("update.status_never", nil))
		return exitOK
	}
	fmt.Fprintln(stdout, status.Report.Render(bundle))
	return exitOK
}

func hostBundleOrWarn(options localeFlags, stderr io.Writer) *i18n.Bundle {
	bundle, err := options.hostBundle()
	if err != nil {
		fmt.Fprintln(stderr, "warning:", err)
	}
	return bundle
}
