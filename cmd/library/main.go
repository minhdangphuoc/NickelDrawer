// Command library finds Vietnamese EPUBs tagged with the wrong language and duplicate books
// in the Kobo's library folders, and fixes them with a backup (see docs/spec/library.md).
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"koboapps/internal/i18n"
	"koboapps/internal/library"
	"koboapps/internal/storage"
)

const (
	defaultAppDir = "/mnt/onboard/.adds/apps/library"
	exitUsage     = 2
	// shownPaths limits how many file names a dry run prints.
	shownPaths = 10
)

func main() {
	os.Exit(execute(os.Args[1:], os.Stdout))
}

type commonFlags struct {
	appDir   string
	confPath string
	locale   string
	root     string
}

func (common *commonFlags) register(flags *flag.FlagSet) {
	flags.StringVar(&common.appDir, "app-dir", defaultAppDir, "app folder")
	flags.StringVar(&common.confPath, "conf", i18n.DefaultConfPath, "Kobo eReader.conf holding CurrentLocale")
	flags.StringVar(&common.locale, "locale", "", "language code overriding the device language, e.g. vi")
	flags.StringVar(&common.root, "root", storage.DefaultRoot, "storage root (tests and Mac runs)")
}

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

// session is one command run: the service, the language and where to print.
type session struct {
	service  *library.Service
	messages *i18n.Bundle
	out      io.Writer
	options  options
}

type options struct {
	dryRun   bool
	confirm  bool
	selected bool
}

func (session *session) say(key string, pairs ...string) {
	fmt.Fprintln(session.out, session.messages.Message(key, i18n.Params(pairs...)))
}

// command is one subcommand. locked commands change files, so only one runs at a time.
type command struct {
	maxArguments int
	locked       bool
	run          func(session *session, arguments []string) error
}

var commands = map[string]command{
	"scan":             {0, true, (*session).scan},
	"fix-language":     {0, true, (*session).fixLanguage},
	"normalize-nfc":    {0, true, (*session).normalizeNFC},
	"cancel":           {0, false, (*session).cancel},
	"dedupe-list":      {0, false, (*session).dedupeList},
	"remove-duplicate": {1, true, (*session).removeDuplicate},
	"undo":             {1, true, (*session).undo},
	"restore":          {1, true, (*session).restoreTrash},
	"pick":             {1, false, (*session).pick},
	"page":             {0, false, (*session).nextPage},
	"state":            {0, false, (*session).state},
}

func execute(args []string, out io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(out, commonFlags{appDir: defaultAppDir, confPath: i18n.DefaultConfPath}.messages().Message("msg.usage", nil))
		return exitUsage
	}
	selected, known := commands[args[0]]
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.SetOutput(out)
	var common commonFlags
	common.register(flags)
	var chosen options
	flags.BoolVar(&chosen.dryRun, "dry-run", false, "fix-language, normalize-nfc: list what would change, write nothing")
	flags.BoolVar(&chosen.confirm, "confirm", false, "remove-duplicate: required")
	flags.BoolVar(&chosen.selected, "selected", false, "remove-duplicate: use the duplicate picked on the screen")
	if !known {
		fmt.Fprintln(out, common.messages().Message("msg.unknown_command", i18n.Params("command", args[0])))
		return exitUsage
	}
	if err := flags.Parse(args[1:]); err != nil {
		return exitUsage
	}
	messages := common.messages()
	if flags.NArg() > selected.maxArguments {
		fmt.Fprintln(out, messages.Message("msg.usage", nil))
		return exitUsage
	}
	restore := storage.SetRoot(common.root)
	defer restore()
	if err := runLocked(selected, common, messages, out, chosen, flags.Args()); err != nil {
		fmt.Fprintln(out, messages.Message("msg.error", i18n.Params("message", messages.Describe(err))))
		return 1
	}
	return 0
}

func runLocked(selected command, common commonFlags, messages *i18n.Bundle, out io.Writer, chosen options, arguments []string) error {
	paths := library.Paths{AppDir: common.appDir}
	if err := os.MkdirAll(paths.Data(), 0o755); err != nil {
		return i18n.NewError("msg.err_mkdir", "path", paths.Data()).WithCause(err)
	}
	if selected.locked {
		release, err := library.Acquire(paths)
		if err != nil {
			return err
		}
		defer release()
	}
	return selected.run(&session{
		service: library.NewService(paths), messages: messages, out: out, options: chosen,
	}, arguments)
}

func (session *session) scan(_ []string) error {
	report, err := session.service.Scan()
	if errors.Is(err, library.ErrCancelled) {
		session.say("msg.scan_cancelled")
		return nil
	}
	if err != nil {
		return err
	}
	session.printCounts(report.Counts)
	return nil
}

func (session *session) printCounts(counts library.Counts) {
	session.say("msg.scan_summary",
		"books", strconv.Itoa(counts.Files), "wrong", strconv.Itoa(counts.WrongLanguage),
		"not_nfc", strconv.Itoa(counts.NotNFC), "groups", strconv.Itoa(counts.DuplicateGroups))
	if counts.Unreadable > 0 {
		session.say("msg.scan_unreadable", "count", strconv.Itoa(counts.Unreadable))
	}
	if counts.WrongLanguage > 0 {
		session.say("msg.hint_fix")
	}
}

func (session *session) fixLanguage(_ []string) error {
	result, err := session.service.FixLanguage(session.options.dryRun)
	if err != nil {
		return err
	}
	if session.options.dryRun {
		session.say("msg.fix_dry", "count", strconv.Itoa(result.Candidates))
		for index, path := range result.Paths {
			if index == shownPaths {
				break
			}
			fmt.Fprintln(session.out, path)
		}
		return nil
	}
	doneKey := "msg.fix_done"
	if result.Cancelled {
		doneKey = "msg.fix_cancelled"
	}
	session.say(doneKey, "fixed", strconv.Itoa(result.Fixed), "candidates", strconv.Itoa(result.Candidates), "date", result.Date)
	if result.Failed+result.Skipped > 0 {
		session.say("msg.fix_problems", "failed", strconv.Itoa(result.Failed), "skipped", strconv.Itoa(result.Skipped))
	}
	if result.Fixed > 0 {
		session.say("msg.reload_hint")
	}
	return nil
}

func (session *session) normalizeNFC(_ []string) error {
	result, err := session.service.NormalizeNFC(session.options.dryRun)
	if err != nil {
		return err
	}
	if session.options.dryRun {
		session.say("msg.nfc_dry", "count", strconv.Itoa(result.Candidates))
		for index, path := range result.Paths {
			if index == shownPaths {
				break
			}
			fmt.Fprintln(session.out, path)
		}
		return nil
	}
	doneKey := "msg.nfc_done"
	if result.Cancelled {
		doneKey = "msg.nfc_cancelled"
	}
	session.say(doneKey, "fixed", strconv.Itoa(result.Fixed), "candidates", strconv.Itoa(result.Candidates), "date", result.Date)
	if result.Failed > 0 {
		session.say("msg.nfc_problems", "failed", strconv.Itoa(result.Failed))
	}
	if result.Fixed > 0 {
		session.say("msg.reload_hint")
	}
	return nil
}

// cancel asks the running scan or fix to stop after the book it is working on.
func (session *session) cancel(_ []string) error {
	wasRunning, err := library.RequestCancel(session.service.Paths)
	if err != nil {
		return err
	}
	if !wasRunning {
		session.say("msg.nothing_running")
		return nil
	}
	session.say("msg.cancelling")
	return nil
}

func (session *session) dedupeList(_ []string) error {
	summary, scanned, err := library.LoadSummary(session.service.Paths)
	if err != nil {
		return err
	}
	if !scanned {
		return i18n.NewError("msg.err_no_report")
	}
	if len(summary.Duplicates) == 0 {
		session.say("msg.no_duplicates")
		return nil
	}
	for _, group := range summary.Duplicates {
		kindKey := "msg.kind_probable"
		if group.Kind == library.KindIdentical {
			kindKey = "msg.kind_identical"
		}
		session.say("msg.group_header", "group", strconv.Itoa(group.ID), "kind", session.messages.Message(kindKey, nil))
		for _, path := range group.Paths {
			fmt.Fprintln(session.out, "  "+path)
		}
	}
	return nil
}

func (session *session) removeDuplicate(arguments []string) error {
	path := ""
	switch {
	case session.options.selected && len(arguments) == 0:
		path = session.service.Selected()
	case !session.options.selected && len(arguments) == 1:
		path = arguments[0]
	}
	if path == "" {
		return i18n.NewError("msg.err_nothing_selected")
	}
	if err := session.service.RemoveDuplicate(path, session.options.confirm); err != nil {
		return err
	}
	session.say("msg.removed", "path", path, "date", time.Now().Format("2006-01-02"))
	return nil
}

func (session *session) undo(arguments []string) error {
	restored, err := session.service.Undo(optionalArgument(arguments))
	if restored > 0 {
		session.say("msg.undo_done", "count", strconv.Itoa(restored))
		session.say("msg.reload_hint")
	}
	return err
}

func (session *session) restoreTrash(arguments []string) error {
	restored, err := session.service.RestoreTrash(optionalArgument(arguments))
	if restored > 0 {
		session.say("msg.restore_done", "count", strconv.Itoa(restored))
		session.say("msg.reload_hint")
	}
	return err
}

func optionalArgument(arguments []string) string {
	if len(arguments) == 0 {
		return ""
	}
	return arguments[0]
}

func (session *session) pick(arguments []string) error {
	if len(arguments) != 1 {
		return i18n.NewError("msg.err_nothing_selected")
	}
	return session.service.Select(arguments[0])
}

func (session *session) nextPage(_ []string) error { return session.service.NextPage() }

// state prints the ui.json variables as one JSON object. Errors are not shown: the screen then
// keeps its defaults, and the real cause is in data/log.txt.
func (session *session) state(_ []string) error {
	return json.NewEncoder(session.out).Encode(session.service.State(session.messages))
}
