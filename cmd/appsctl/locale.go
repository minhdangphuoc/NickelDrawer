package main

import (
	"flag"

	"koboapps/internal/i18n"
)

// localeFlags select the language and where host strings live. On the Kobo the defaults are
// right; the installer overrides them to read the mounted Kobo from a Mac.
type localeFlags struct {
	locale   string // overrides the device language when set
	confPath string // Kobo eReader.conf
	hostDir  string // koboapps folder; its locales/ holds the host strings
}

func (f *localeFlags) register(flags *flag.FlagSet) {
	flags.StringVar(&f.locale, "locale", "", "language code overriding the device language, e.g. vi")
	flags.StringVar(&f.confPath, "conf", i18n.DefaultConfPath, "Kobo eReader.conf holding CurrentLocale")
	flags.StringVar(&f.hostDir, "koboapps-dir", defaultKoboappsDir(), "koboapps folder whose locales/ holds the host strings")
}

func (f localeFlags) effectiveLocale() string {
	if f.locale != "" {
		return f.locale
	}
	return i18n.CurrentLocale(f.confPath)
}
