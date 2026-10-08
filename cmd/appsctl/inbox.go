package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"time"

	"koboapps/internal/i18n"
	"koboapps/internal/inbox"
	"koboapps/internal/storage"
)

const (
	// routeFileEnv is the only way a received file's path reaches `route --from-env`. localgo
	// exports it for the hook; the path never travels through a shell string.
	routeFileEnv = "LOCALGO_FILE"
	// defaultSource labels journal lines from `route <path>` and `sort`.
	defaultSource = "manual"

	rejectedKeepDays = 7
	maxStatusReasons = 3
	inboxUsage       = "usage: appsctl inbox route [--from-env | PATH] [--source NAME] | sort [--source-dir DIR] | status | purge"
)

var sourceNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,31}$`)

// summaryFolders are the library folders `status` lists, in this order.
var summaryFolders = []storage.Folder{storage.Books, storage.Documents, storage.Comics, storage.Pictures}

// runInbox dispatches `appsctl inbox <subcommand>`.
func runInbox(args []string, stdout, stderr io.Writer) int {
	subcommands := map[string]func([]string, io.Writer, io.Writer) int{
		"route":  runInboxRoute,
		"sort":   runInboxSort,
		"status": runInboxStatus,
		"purge":  runInboxPurge,
	}
	if len(args) == 0 {
		fmt.Fprintln(stderr, inboxUsage)
		return exitFailure
	}
	subcommand, found := subcommands[args[0]]
	if !found {
		fmt.Fprintf(stderr, "unknown inbox command %q\n%s\n", args[0], inboxUsage)
		return exitFailure
	}
	return subcommand(args[1:], stdout, stderr)
}

// inboxSession is what every subcommand needs: the language and the layout root.
type inboxSession struct {
	onboardRoot string
	locale      localeFlags
	bundle      *i18n.Bundle
	restore     func()
}

func (session *inboxSession) register(flags *flag.FlagSet) {
	flags.StringVar(&session.onboardRoot, "onboard-root", storage.DefaultRoot, "user storage root (tests and a Mac use a temp folder)")
	session.locale.register(flags)
}

// open applies the root and loads the host strings. Call close when done.
func (session *inboxSession) open(stderr io.Writer) {
	session.restore = storage.SetRoot(session.onboardRoot)
	if session.locale.hostDir == defaultKoboappsDir() {
		session.locale.hostDir = storage.KoboappsDir()
	}
	bundle, err := session.locale.hostBundle()
	if err != nil {
		fmt.Fprintln(stderr, "warning:", err)
	}
	session.bundle = bundle
}

func (session *inboxSession) close() { session.restore() }

func (session *inboxSession) say(key string, pairs ...string) string {
	return session.bundle.Message(key, i18n.Params(pairs...))
}

// router builds the router from the shipped rules plus the optional override. A broken
// override only warns; broken shipped rules stop the command.
func (session *inboxSession) router(stderr io.Writer) (*inbox.Router, error) {
	rules, overrideErr, err := inbox.LoadRules(inbox.ShippedRulesPath(), inbox.OverrideRulesPath())
	if err != nil {
		return nil, err
	}
	if overrideErr != nil {
		fmt.Fprintln(stderr, session.say("inbox.warn_override", "cause", overrideErr.Error()))
	}
	return inbox.NewRouter(rules, inbox.JournalPath()), nil
}

func (session *inboxSession) fail(stderr io.Writer, err error) int {
	var rulesErr *inbox.RulesError
	switch {
	case errors.Is(err, inbox.ErrOutsideSource):
		fmt.Fprintln(stderr, session.say("inbox.err_outside"))
	case errors.Is(err, inbox.ErrIsFolder):
		fmt.Fprintln(stderr, session.say("inbox.err_folder"))
	case errors.As(err, &rulesErr):
		fmt.Fprintln(stderr, session.say("inbox.err_rules", "cause", err.Error()))
	default:
		fmt.Fprintln(stderr, session.say("inbox.err_failed", "cause", err.Error()))
	}
	return exitFailure
}

func runInboxRoute(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("inbox route", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var session inboxSession
	session.register(flags)
	fromEnv := flags.Bool("from-env", false, "take the file path from the "+routeFileEnv+" environment variable")
	source := flags.String("source", defaultSource, "who sent the file, for the journal (e.g. localsend)")
	if err := flags.Parse(args); err != nil {
		return exitFailure
	}
	path, err := routePathArgument(*fromEnv, flags.Args())
	if err == nil && !sourceNamePattern.MatchString(*source) {
		err = fmt.Errorf("--source %q must match %s", *source, sourceNamePattern)
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		fmt.Fprintln(stderr, inboxUsage)
		return exitFailure
	}
	session.open(stderr)
	defer session.close()
	router, err := session.router(stderr)
	if err != nil {
		return session.fail(stderr, err)
	}
	outcome, routeErr := router.Route(path, *source)
	if errors.Is(routeErr, inbox.ErrGone) {
		fmt.Fprintln(stdout, session.say("inbox.already_handled"))
		return exitOK
	}
	if outcome.Destination != "" {
		fmt.Fprintln(stdout, session.describe(outcome))
	}
	if routeErr != nil {
		return session.fail(stderr, routeErr)
	}
	return exitOK
}

// routePathArgument enforces the two ways of naming the file. With --from-env a path argument
// is an error, so a hook cannot be tricked into routing something the environment did not name.
func routePathArgument(fromEnv bool, positional []string) (string, error) {
	switch {
	case fromEnv && len(positional) > 0:
		return "", errors.New("--from-env takes the path from " + routeFileEnv + " only; remove the path argument")
	case fromEnv:
		path := os.Getenv(routeFileEnv)
		if path == "" {
			return "", errors.New(routeFileEnv + " is empty; --from-env only works inside the LocalSend hook")
		}
		return path, nil
	case len(positional) == 1:
		return positional[0], nil
	}
	return "", errors.New("give exactly one PATH, or use --from-env")
}

func (session *inboxSession) describe(outcome inbox.Outcome) string {
	if outcome.IsRejected() {
		return session.say("inbox.rejected_file", "name", outcome.Name,
			"reason", session.say("inbox.reason."+string(outcome.Reason)))
	}
	return session.say("inbox.routed_file", "name", outcome.StoredAs,
		"folder", session.say("inbox.folder."+string(outcome.Destination)))
}

func runInboxSort(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("inbox sort", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var session inboxSession
	session.register(flags)
	sourceDir := flags.String("source-dir", "", "folder to sort: the inbox (default) or the old LocalSend folder")
	source := flags.String("source", defaultSource, "label for the journal")
	if err := flags.Parse(args); err != nil || flags.NArg() > 0 || !sourceNamePattern.MatchString(*source) {
		fmt.Fprintln(stderr, inboxUsage)
		return exitFailure
	}
	session.open(stderr)
	defer session.close()
	if *sourceDir == "" {
		*sourceDir = storage.Path(storage.Inbox)
	}
	router, err := session.router(stderr)
	if err != nil {
		return session.fail(stderr, err)
	}
	report, err := router.SortFolder(*sourceDir, *source)
	if err != nil {
		return session.fail(stderr, err)
	}
	fmt.Fprintln(stdout, session.say("inbox.sorted",
		"count", strconv.Itoa(report.Routed+report.Rejected), "rejected", strconv.Itoa(report.Rejected)))
	for _, failure := range report.Failures {
		session.fail(stderr, failure)
	}
	if len(report.Failures) > 0 {
		return exitFailure
	}
	return exitOK
}

func runInboxStatus(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("inbox status", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var session inboxSession
	session.register(flags)
	if err := flags.Parse(args); err != nil || flags.NArg() > 0 {
		fmt.Fprintln(stderr, inboxUsage)
		return exitFailure
	}
	session.open(stderr)
	defer session.close()
	entries, _, err := inbox.ReadEntries(inbox.JournalPath())
	if err != nil {
		return session.fail(stderr, err)
	}
	fmt.Fprint(stdout, session.formatSummary(inbox.SummarizeLastSession(entries)))
	return exitOK
}

// formatSummary prints at most 8 short lines: 4 folders, the rejected count, 3 top reasons.
func (session *inboxSession) formatSummary(summary inbox.Summary) string {
	if summary.IsEmpty() {
		return session.say("inbox.status_empty") + "\n"
	}
	var text string
	for _, folder := range summaryFolders {
		if count := summary.Routed[folder]; count > 0 {
			text += session.say("inbox.count."+string(folder), "count", strconv.Itoa(count)) + "\n"
		}
	}
	if summary.Rejected > 0 {
		text += session.say("inbox.count.rejected", "count", strconv.Itoa(summary.Rejected)) + "\n"
	}
	for index, reason := range summary.Reasons {
		if index == maxStatusReasons {
			break
		}
		text += session.say("inbox.reason_line", "reason", session.say("inbox.reason."+string(reason.Reason)),
			"count", strconv.Itoa(reason.Count)) + "\n"
	}
	return text
}

func runInboxPurge(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("inbox purge", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var session inboxSession
	session.register(flags)
	if err := flags.Parse(args); err != nil || flags.NArg() > 0 {
		fmt.Fprintln(stderr, inboxUsage)
		return exitFailure
	}
	session.open(stderr)
	defer session.close()
	router := inbox.NewRouter(inbox.Rules{}, inbox.JournalPath())
	deleted, err := router.PurgeRejected(rejectedKeepDays * 24 * time.Hour)
	fmt.Fprintln(stdout, session.say("inbox.purged", "count", strconv.Itoa(deleted)))
	if err != nil {
		return session.fail(stderr, err)
	}
	return exitOK
}
