package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"koboapps/internal/i18n"
)

// englishHostStrings is a copy of host/locales/en.json (go:embed cannot reach outside this
// package folder; `make sync-embedded` refreshes it and a test checks it). It keeps the host
// menu and messages readable when the locales folder on the device is missing or broken.
//
//go:embed embedded/en.json
var englishHostStrings []byte

func embeddedEnglish() (i18n.Catalog, error) {
	var catalog i18n.Catalog
	if err := json.Unmarshal(englishHostStrings, &catalog); err != nil {
		return nil, fmt.Errorf("embedded English host strings are broken: %w", err)
	}
	for key, text := range catalog {
		catalog[key] = strings.ReplaceAll(text, `\n`, "\n")
	}
	return catalog, nil
}

// hostBundle never fails: it reads the locale chain from koboapps/locales and ends with the
// embedded English strings, so a message key never reaches the screen. The error lists
// locale files that exist but are broken.
func (f localeFlags) hostBundle() (*i18n.Bundle, error) {
	var catalogs []i18n.Catalog
	var problems []error
	for _, name := range i18n.Chain(f.effectiveLocale()) {
		catalog, err := i18n.LoadCatalog(filepath.Join(f.hostDir, "locales", name+".json"))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			problems = append(problems, err)
			continue
		}
		catalogs = append(catalogs, catalog)
	}
	english, err := embeddedEnglish()
	if err != nil {
		problems = append(problems, err)
	}
	return i18n.NewBundle(append(catalogs, english)...), errors.Join(problems...)
}
