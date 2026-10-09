package main

import (
	"encoding/json"
	"errors"
	"flag"
	"io"
	"strings"

	"koboapps/internal/drawer"
	"koboapps/internal/i18n"
	"koboapps/internal/manifest"
)

// drawerOptions are the flags shared by the drawer subcommands; the defaults are the device
// paths, tests and the installer override them.
type drawerOptions struct {
	localeFlags
	appsDir    string
	cacheDir   string
	menuFile   string
	errorsFile string
	confirm    bool
	screenID   string
	value      optionalValue
}

// optionalValue is a string flag that remembers whether it was given, because an empty
// field value ("--value ”") differs from no value at all.
type optionalValue struct {
	text  string
	isSet bool
}

func (v *optionalValue) String() string { return v.text }

func (v *optionalValue) Set(text string) error {
	v.text, v.isSet = text, true
	return nil
}

func (v optionalValue) pointer() *string {
	if !v.isSet {
		return nil
	}
	text := v.text
	return &text
}

func (o *drawerOptions) register(flags *flag.FlagSet) {
	o.localeFlags.register(flags)
	flags.StringVar(&o.appsDir, "apps-dir", defaultAppsDir(), "folder containing app folders")
	flags.StringVar(&o.cacheDir, "cache-dir", "", "folder for the status cache (default: the system temp folder)")
	flags.StringVar(&o.menuFile, "out", defaultMenuFile, "NickelMenu config file regenerated after changes")
	flags.StringVar(&o.errorsFile, "errors-file", defaultErrorsFile, "file receiving one line per invalid app")
	flags.BoolVar(&o.confirm, "confirm", false, "confirm a destructive command (remove)")
	flags.StringVar(&o.screenID, "screen", "", "screen id (screen, event; default: the start screen)")
	flags.Var(&o.value, "value", "value of a toggle, radio option, field or list row (event)")
}

// drawerSession is one parsed invocation with its collaborators.
type drawerSession struct {
	options   drawerOptions
	config    drawer.Config
	bundle    *i18n.Bundle
	positions []string
	stdout    io.Writer
}

// drawerHooks lets tests replace the script runners and the network check.
var drawerHooks = func(config *drawer.Config) {}

func runDrawer(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return writeDrawerFailure(stdout, fallbackBundle(), drawerUsageError())
	}
	subcommands := map[string]func(drawerSession) (any, error){
		"model":  drawerModel,
		"run":    drawerRun,
		"info":   drawerInfo,
		"remove": drawerRemove,
		"screen": drawerScreen,
		"event":  drawerEvent,
	}
	handler, found := subcommands[args[0]]
	if !found {
		return writeDrawerFailure(stdout, fallbackBundle(), drawerUsageError())
	}
	session, err := newDrawerSession(args[0], args[1:], stdout, stderr)
	if err != nil {
		return writeDrawerFailure(stdout, session.bundle, err)
	}
	response, err := handler(session)
	if err != nil {
		return writeDrawerFailure(stdout, session.bundle, err)
	}
	if err := writeDrawerJSON(stdout, response); err != nil {
		return exitFailure
	}
	return exitOK
}

func newDrawerSession(name string, args []string, stdout, stderr io.Writer) (drawerSession, error) {
	session := drawerSession{stdout: stdout, bundle: i18n.NewBundle()}
	flags := flag.NewFlagSet("drawer "+name, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	session.options.register(flags)
	positions, err := parseInterspersed(flags, args)
	bundle, bundleErr := session.options.hostBundle()
	session.bundle = bundle
	if bundleErr != nil {
		io.WriteString(stderr, "warning: "+bundleErr.Error()+"\n")
	}
	if err != nil {
		return session, drawerUsageError()
	}
	session.positions = positions
	session.config = drawer.Config{
		AppsDir: session.options.appsDir, HostDir: session.options.hostDir,
		Locale: session.options.effectiveLocale(), Bundle: bundle, CacheDir: session.options.cacheDir,
	}
	drawerHooks(&session.config)
	return session, nil
}

// parseInterspersed accepts flags before and after positional arguments, because the protocol
// calls `remove <id> --confirm`.
func parseInterspersed(flags *flag.FlagSet, args []string) ([]string, error) {
	var positions []string
	for {
		if err := flags.Parse(args); err != nil {
			return nil, err
		}
		if flags.NArg() == 0 {
			return positions, nil
		}
		positions = append(positions, flags.Arg(0))
		args = flags.Args()[1:]
	}
}

// fallbackBundle is the embedded English text, for failures before flags are parsed.
func fallbackBundle() *i18n.Bundle {
	bundle, _ := drawerOptions{}.hostBundle()
	return bundle
}

func drawerUsageError() error {
	return &drawer.Error{Code: drawer.CodeUsage, Params: map[string]string{}}
}

func drawerModel(session drawerSession) (any, error) {
	if len(session.positions) != 0 {
		return nil, drawerUsageError()
	}
	return session.config.BuildModel()
}

func drawerInfo(session drawerSession) (any, error) {
	if len(session.positions) != 1 {
		return nil, drawerUsageError()
	}
	return session.config.AppInfo(session.positions[0])
}

func drawerRun(session drawerSession) (any, error) {
	if len(session.positions) != 2 {
		return nil, drawerUsageError()
	}
	appID, actionID := session.positions[0], session.positions[1]
	if drawer.IsHostID(appID) {
		return runHostItem(session, appID, actionID)
	}
	return session.config.Run(appID, actionID)
}

// runHostItem runs the built-in items the model lists under "system".
func runHostItem(session drawerSession, itemID, actionID string) (any, error) {
	switch {
	case itemID == drawer.HostRefreshID && actionID == drawer.HostRefreshAct:
		return refreshHostItem(session)
	case itemID == drawer.HostUpdatesID && actionID == drawer.HostUpdatesAct:
		return session.config.RunAction(session.options.hostDir, manifest.Action{
			ID: actionID, Label: actionID, Run: "bin/update-apply.sh", Output: manifest.OutputShow, Wifi: true,
		})
	}
	return nil, &drawer.Error{Code: drawer.CodeUnknownAction, Params: map[string]string{"app": itemID, "action": actionID}}
}

func refreshHostItem(session drawerSession) (any, error) {
	result, err := regenerateMenu(session)
	if err != nil {
		return nil, &drawer.Error{Code: drawer.CodeIO, Params: map[string]string{"detail": err.Error()}}
	}
	return drawer.RunResult{
		V: drawer.ProtocolVersion, OK: true, Text: strings.TrimRight(refreshSummary(result), "\n"), Rescan: manifest.RescanNone,
	}, nil
}

func drawerScreen(session drawerSession) (any, error) {
	if len(session.positions) != 1 {
		return nil, drawerUsageError()
	}
	return session.config.Screen(session.positions[0], session.options.screenID)
}

func drawerEvent(session drawerSession) (any, error) {
	if len(session.positions) != 2 {
		return nil, drawerUsageError()
	}
	return session.config.RunEvent(drawer.Event{
		AppID: session.positions[0], ActionID: session.positions[1],
		Value: session.options.value.pointer(), ScreenID: session.options.screenID,
	})
}

func drawerRemove(session drawerSession) (any, error) {
	if len(session.positions) != 1 {
		return nil, drawerUsageError()
	}
	if !session.options.confirm {
		return nil, &drawer.Error{Code: drawer.CodeNeedConfirm, Params: map[string]string{}}
	}
	removed, err := session.config.Remove(session.positions[0])
	if err != nil {
		return nil, err
	}
	if _, err := regenerateMenu(session); err != nil {
		return nil, &drawer.Error{Code: drawer.CodeIO, Params: map[string]string{"detail": err.Error()}}
	}
	return removed, nil
}

func regenerateMenu(session drawerSession) (genMenuResult, error) {
	return generateMenu(genMenuOptions{
		appsDir: session.options.appsDir, deviceApps: session.options.appsDir,
		deviceHost: session.options.hostDir, menuFile: session.options.menuFile,
		errorsFile: session.options.errorsFile, localeFlags: session.options.localeFlags,
	})
}

func writeDrawerJSON(out io.Writer, value any) error {
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(value)
}

type drawerErrorResponse struct {
	V     int              `json:"v"`
	Error drawerErrorField `json:"error"`
}

type drawerErrorField struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// writeDrawerFailure prints the protocol error document; the exit code tells the mod to read it.
func writeDrawerFailure(stdout io.Writer, bundle *i18n.Bundle, err error) int {
	var drawerErr *drawer.Error
	if !errors.As(err, &drawerErr) {
		drawerErr = &drawer.Error{Code: drawer.CodeIO, Params: map[string]string{"detail": err.Error()}}
	}
	response := drawerErrorResponse{V: drawer.ProtocolVersion, Error: drawerErrorField{
		Code:    drawerErr.Code,
		Message: bundle.Message("drawer.err."+drawerErr.Code, drawerErr.Params),
	}}
	_ = writeDrawerJSON(stdout, response)
	return exitFailure
}
