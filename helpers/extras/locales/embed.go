// Package locales is the single source of the Extras helper's texts: the JSON files here are
// installed to the device for the native library, and compiled into extras-tweaks as its English
// fallback (and into the repo's preview tool), so no copy can drift.
package locales

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"

	"koboapps/internal/i18n"
)

//go:embed en.json vi.json
var files embed.FS

// Embedded returns the compiled-in catalog of one language ("en", "vi").
func Embedded(language string) (i18n.Catalog, error) {
	data, err := files.ReadFile(language + ".json")
	if err != nil {
		return nil, fmt.Errorf("no embedded locale %q: %w", language, err)
	}
	var catalog i18n.Catalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("embedded locale %s.json is not a flat JSON object: %w", language, err)
	}
	for key, text := range catalog {
		catalog[key] = strings.ReplaceAll(text, `\n`, "\n")
	}
	return catalog, nil
}

// English is the source language catalog; it panics only if the embedded file is broken,
// which the package tests rule out.
func English() i18n.Catalog {
	catalog, err := Embedded(i18n.SourceLocale)
	if err != nil {
		panic(err)
	}
	return catalog
}
