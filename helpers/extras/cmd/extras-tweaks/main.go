// Command extras-tweaks is the backend of the Extras helper's Tweaks panel. The native library
// runs it as a child process:
//
//	extras-tweaks list           the tweaks and their state, as JSON
//	extras-tweaks apply <id>     applies one tweak, replies with a toast text and the new value
//
// Contract: helpers/extras/docs/tweaks.md. Exit 0 with a document on stdout, or exit 2 with
// {"v":1,"ok":false,"error":"..."} (the text is already localized).
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"

	"koboapps/helpers/extras/internal/tweaks"
	"koboapps/helpers/extras/locales"
	"koboapps/internal/i18n"
)

const (
	exitOK      = 0
	exitFailure = 2

	listCommand  = "list"
	applyCommand = "apply"
)

// options are the command line flags; empty values mean the device defaults.
type options struct {
	locale, confPath, localesDir, flagsDir string
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	settings, positions, err := parseArguments(args, stderr)
	if err != nil {
		return writeFailure(stdout, settings.bundle(), err)
	}
	config := settings.config()
	switch {
	case len(positions) == 1 && positions[0] == listCommand:
		return writeJSON(stdout, config.List())
	case len(positions) == 2 && positions[0] == applyCommand:
		result, err := config.Apply(positions[1])
		if err != nil {
			return writeFailure(stdout, config.Bundle, err)
		}
		return writeJSON(stdout, result)
	}
	return writeFailure(stdout, config.Bundle, &tweaks.Error{Message: config.Bundle.Message("tweak.err.usage", nil)})
}

// parseArguments reads flags that may come before or after the command words.
func parseArguments(args []string, stderr io.Writer) (options, []string, error) {
	var settings options
	var positions []string
	flags := flag.NewFlagSet("extras-tweaks", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&settings.locale, "locale", "", "locale code (default: from Kobo eReader.conf)")
	flags.StringVar(&settings.confPath, "conf", "", "path of Kobo eReader.conf")
	flags.StringVar(&settings.localesDir, "locales-dir", "", "folder of <locale>.json files")
	flags.StringVar(&settings.flagsDir, "flags-dir", "", "the helper's folder (flag files)")
	for len(args) > 0 {
		if err := flags.Parse(args); err != nil {
			return settings, nil, &tweaks.Error{Message: err.Error()}
		}
		if flags.NArg() == 0 {
			break
		}
		positions = append(positions, flags.Arg(0))
		args = flags.Args()[1:]
	}
	return settings, positions, nil
}

// withDefaults fills every empty option with the device default.
func (settings options) withDefaults() options {
	defaults := tweaks.DefaultConfig(nil)
	if settings.confPath == "" {
		settings.confPath = defaults.ConfPath
	}
	if settings.flagsDir == "" {
		settings.flagsDir = defaults.FlagsDir
	}
	if settings.localesDir == "" {
		settings.localesDir = filepath.Join(settings.flagsDir, "locales")
	}
	if settings.locale == "" {
		settings.locale = i18n.CurrentLocale(settings.confPath)
	}
	return settings
}

func (settings options) config() tweaks.Config {
	settings = settings.withDefaults()
	config := tweaks.DefaultConfig(settings.bundle())
	config.ConfPath = settings.confPath
	config.FlagsDir = settings.flagsDir
	return config
}

// bundle resolves texts through the locale files on the device (exact locale, language, en) and
// ends with the compiled-in English, so a missing locales folder still gives readable text.
func (settings options) bundle() *i18n.Bundle {
	settings = settings.withDefaults()
	var catalogs []i18n.Catalog
	for _, name := range i18n.Chain(settings.locale) {
		if catalog, err := i18n.LoadCatalog(filepath.Join(settings.localesDir, name+".json")); err == nil {
			catalogs = append(catalogs, catalog)
		}
	}
	return i18n.NewBundle(append(catalogs, locales.English())...)
}

func writeJSON(stdout io.Writer, value any) int {
	encoder := json.NewEncoder(stdout)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return exitFailure
	}
	return exitOK
}

// writeFailure prints the failure document; the exit code tells the library to read it.
func writeFailure(stdout io.Writer, bundle *i18n.Bundle, err error) int {
	message := err.Error()
	var tweakErr *tweaks.Error
	if errors.As(err, &tweakErr) {
		message = tweakErr.Message
	}
	if bundle == nil {
		bundle = i18n.NewBundle(locales.English())
	}
	writeJSON(stdout, tweaks.Failure{V: tweaks.ProtocolVersion, Error: message})
	return exitFailure
}
