package main

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"koboapps/internal/i18n"
	"koboapps/internal/manifest"
)

// runMsg prints one app message in the device language. A script must never fail because a
// string is missing, so an unknown key prints the key, warns on stderr and exits 0.
func runMsg(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("msg", flag.ContinueOnError)
	flags.SetOutput(stderr)
	appID := flags.String("app", "", "app id (required)")
	appsDir := flags.String("apps-dir", defaultAppsDir(), "folder containing app folders")
	var localeOptions localeFlags
	localeOptions.register(flags)
	if err := flags.Parse(args); err != nil {
		return exitFailure
	}
	if *appID == "" || flags.NArg() < 1 {
		fmt.Fprintln(stderr, "usage: appsctl msg --app <id> <key> [name=value ...]")
		return exitFailure
	}
	key, params := flags.Arg(0), parseParams(flags.Args()[1:], stderr)
	bundle, err := appBundle(*appsDir, *appID, localeOptions.effectiveLocale())
	if err != nil {
		fmt.Fprintln(stderr, "warning:", err)
	}
	if _, found := bundle.Lookup(key); !found {
		fmt.Fprintf(stderr, "warning: no message %q for app %q in locales/; add it to locales/en.json\n", key, *appID)
	}
	fmt.Fprintln(stdout, bundle.Message(key, params))
	return exitOK
}

func appBundle(appsDir, appID, locale string) (*i18n.Bundle, error) {
	if appID != filepath.Base(appID) || strings.HasPrefix(appID, ".") {
		return i18n.NewBundle(), fmt.Errorf("invalid app id %q", appID)
	}
	return i18n.LoadBundle(filepath.Join(appsDir, appID, manifest.LocalesDirName), locale)
}

func parseParams(pairs []string, stderr io.Writer) map[string]string {
	params := map[string]string{}
	for _, pair := range pairs {
		name, value, found := strings.Cut(pair, "=")
		if !found || name == "" {
			fmt.Fprintf(stderr, "warning: ignoring %q; parameters look like name=value\n", pair)
			continue
		}
		params[name] = value
	}
	return params
}
