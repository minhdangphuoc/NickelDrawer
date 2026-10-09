// Command ghsync downloads new books from a private GitHub repo (add-only).
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	// Embedded CA roots: the Kobo may lack a system CA bundle.
	_ "golang.org/x/crypto/x509roots/fallback"

	"koboapps/internal/ghsync"
	"koboapps/internal/i18n"
)

const (
	defaultAppDir  = "/mnt/onboard/.adds/apps/ghsync"
	defaultAPIBase = "https://api.github.com"
	exitUsage      = 2
)

func main() {
	os.Exit(execute(os.Args[1:], os.Stdout))
}

// commonFlags are shared by every subcommand.
type commonFlags struct {
	appDir   string
	confPath string
	locale   string
}

func (common *commonFlags) register(flags *flag.FlagSet) {
	flags.StringVar(&common.appDir, "app-dir", defaultAppDir, "app folder")
	flags.StringVar(&common.confPath, "conf", i18n.DefaultConfPath, "Kobo eReader.conf holding CurrentLocale")
	flags.StringVar(&common.locale, "locale", "", "language code overriding the device language, e.g. vi")
}

// messages loads the app's locale files for the device language. A broken file is reported
// on stderr and the rest still applies; an app without locale files prints message keys.
func (common commonFlags) messages() *i18n.Bundle {
	locale := common.locale
	if locale == "" {
		locale = i18n.CurrentLocale(common.confPath)
	}
	bundle, err := i18n.LoadBundle(filepath.Join(common.appDir, "locales"), locale)
	if err != nil {
		fmt.Fprintln(os.Stderr, "warning:", err)
	}
	return bundle
}

// say prints one localized line.
func say(out io.Writer, bundle *i18n.Bundle, key string, pairs ...string) {
	fmt.Fprintln(out, bundle.Message(key, i18n.Params(pairs...)))
}

func execute(args []string, out io.Writer) int {
	fallback := commonFlags{appDir: defaultAppDir, confPath: i18n.DefaultConfPath}
	if len(args) == 0 {
		say(out, fallback.messages(), "msg.usage")
		return exitUsage
	}
	switch args[0] {
	case "run":
		return runCommand(args[1:], out)
	case "status":
		return statusCommand(args[1:], out)
	case "state":
		return stateCommand(args[1:], out)
	case "pair":
		return pairCommand(args[1:], out)
	default:
		say(out, fallback.messages(), "msg.unknown_command", "command", args[0])
		return exitUsage
	}
}

func runCommand(args []string, out io.Writer) int {
	flags := flag.NewFlagSet("run", flag.ContinueOnError)
	flags.SetOutput(out)
	var common commonFlags
	common.register(flags)
	apiBase := flags.String("api-base", defaultAPIBase, "GitHub API base URL (tests)")
	timeout := flags.Duration("timeout", 10*time.Minute, "overall timeout")
	dryRun := flags.Bool("dry-run", false, "list what would be downloaded")
	redownloadDeleted := flags.Bool("redownload-deleted", false, "download again files the user deleted from the device")
	if err := flags.Parse(args); err != nil {
		return exitUsage
	}
	messages := common.messages()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	summary, err := ghsync.Run(ctx, ghsync.Options{
		Paths:    ghsync.Paths{AppDir: common.appDir},
		APIBase:  *apiBase,
		DryRun:   *dryRun,
		Messages: messages,

		RedownloadDeleted: *redownloadDeleted,
		Backoff:           2 * time.Second,
	})
	if err != nil {
		say(out, messages, "msg.error", "message", messages.Describe(err))
		return 1
	}
	printSummary(out, messages, summary, *dryRun)
	return 0
}

func printSummary(out io.Writer, messages *i18n.Bundle, summary ghsync.Summary, isDryRun bool) {
	headline := "msg.sync_done"
	if isDryRun {
		headline = "msg.dry_run_done"
	}
	say(out, messages, headline, "new", strconv.Itoa(summary.New), "existing", strconv.Itoa(summary.Existing))
	if summary.Deleted > 0 {
		say(out, messages, "msg.deleted_locally", "count", strconv.Itoa(summary.Deleted))
	}
	if summary.Errors > 0 {
		say(out, messages, "msg.file_errors", "count", strconv.Itoa(summary.Errors))
	}
	if summary.Truncated {
		say(out, messages, "msg.truncated")
	}
}

func statusCommand(args []string, out io.Writer) int {
	flags := flag.NewFlagSet("status", flag.ContinueOnError)
	flags.SetOutput(out)
	var common commonFlags
	common.register(flags)
	if err := flags.Parse(args); err != nil {
		return exitUsage
	}
	messages := common.messages()
	state, err := ghsync.LoadState(ghsync.Paths{AppDir: common.appDir}.State())
	if err != nil {
		say(out, messages, "msg.error", "message", messages.Describe(err))
		return 1
	}
	if state.LastRun == "" {
		say(out, messages, "msg.never_synced")
		return 0
	}
	say(out, messages, "msg.last_run", "time", state.LastRun)
	say(out, messages, "msg.last_result",
		"new", strconv.Itoa(state.LastNew), "existing", strconv.Itoa(state.LastExisting), "errors", strconv.Itoa(state.LastErrors))
	say(out, messages, "msg.tracked", "count", strconv.Itoa(len(state.Files)))
	if state.LastError != "" {
		say(out, messages, "msg.error", "message", state.LastError)
	}
	return 0
}
