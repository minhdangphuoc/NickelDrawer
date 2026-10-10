package manifest

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"koboapps/internal/i18n"
)

const (
	LocalesDirName = "locales"
	messagePrefix  = "msg."
	screenPrefix   = "ui."     // texts referenced as "@ui.key" from ui.json
	formatPrefix   = "format." // date and time patterns of the locale (internal/i18n datetime.go)
	keyName        = "name"
	keyDescription = "description"
)

func actionLabelKey(actionID string) string { return "actions." + actionID + ".label" }

func actionConfirmKey(actionID string) string { return "actions." + actionID + ".confirm" }

// Localize returns a copy of app with name and labels in the bundle's language.
// A translation that would break a rule (length, ':') is skipped in favour of English,
// so one bad translation can never corrupt the generated menu.
func (a App) Localize(bundle *i18n.Bundle) App {
	localized := a
	localized.Name = translatedOrEnglish(bundle, keyName, a.Name, maxNameLen)
	localized.Description = translatedOrEnglish(bundle, keyDescription, a.Description, maxDescriptionN)
	localized.Actions = make([]Action, len(a.Actions))
	for i, action := range a.Actions {
		action.Label = translatedOrEnglish(bundle, actionLabelKey(action.ID), action.Label, maxLabelLen)
		if action.Confirm != "" {
			action.Confirm = translatedOrEnglish(bundle, actionConfirmKey(action.ID), action.Confirm, maxDescriptionN)
		}
		localized.Actions[i] = action
	}
	return localized
}

func translatedOrEnglish(bundle *i18n.Bundle, key, english string, maxLen int) string {
	translated, found := bundle.Lookup(key)
	if !found || len(validateText(key, translated, 1, maxLen)) > 0 {
		return english
	}
	return translated
}

// LoadLocales reads the app's locale files for the given locale. The error lists broken files;
// the bundle is usable regardless.
func (a App) LoadLocales(locale string) (*i18n.Bundle, error) {
	return i18n.LoadBundle(filepath.Join(a.Dir, LocalesDirName), locale)
}

// CheckLocales validates <app>/locales/*.json against the English strings (app.json plus
// locales/en.json). Problems break the rules of the format; warnings are unknown keys.
func CheckLocales(app App) (problems []string, warnings []Warning) {
	files, err := localeFiles(filepath.Join(app.Dir, LocalesDirName))
	if err != nil {
		return []string{err.Error()}, nil
	}
	catalogs := map[string]i18n.Catalog{}
	for _, name := range sortedNames(files) {
		catalog, loadErr := i18n.LoadCatalog(files[name])
		if loadErr != nil {
			problems = append(problems, fmt.Sprintf("%s/%s.json: %v", LocalesDirName, name, errors.Unwrap(loadErr)))
			continue
		}
		catalogs[name] = catalog
	}
	english := englishStrings(app, catalogs[i18n.SourceLocale])
	for _, name := range sortedNames(catalogs) {
		problem, warning := checkCatalog(name, catalogs[name], english, app, hasEnglishFile(files))
		problems = append(problems, problem...)
		warnings = append(warnings, warning...)
	}
	return problems, warnings
}

// isAppTextKey is true for keys an app defines itself: script messages and ui.json texts.
func isAppTextKey(key string) bool {
	return strings.HasPrefix(key, messagePrefix) || strings.HasPrefix(key, screenPrefix) || strings.HasPrefix(key, formatPrefix)
}

func hasEnglishFile(files map[string]string) bool {
	_, found := files[i18n.SourceLocale]
	return found
}

// localeFiles maps locale name to path; a missing folder means no translations.
func localeFiles(dir string) (map[string]string, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", dir, err)
	}
	files := map[string]string{}
	for _, entry := range entries {
		if name, isJSON := strings.CutSuffix(entry.Name(), ".json"); isJSON && !entry.IsDir() {
			files[name] = filepath.Join(dir, entry.Name())
		}
	}
	return files, nil
}

// englishStrings is the reference every translation is compared with.
func englishStrings(app App, englishFile i18n.Catalog) i18n.Catalog {
	english := i18n.Catalog{keyName: app.Name, keyDescription: app.Description}
	for _, action := range app.Actions {
		english[actionLabelKey(action.ID)] = action.Label
		if action.Confirm != "" {
			english[actionConfirmKey(action.ID)] = action.Confirm
		}
	}
	for key, text := range englishFile {
		if isAppTextKey(key) {
			english[key] = text
		}
	}
	return english
}

func checkCatalog(locale string, catalog, english i18n.Catalog, app App, hasEnglish bool) (problems []string, warnings []Warning) {
	where := fmt.Sprintf("%s/%s.json", LocalesDirName, locale)
	keys := make([]string, 0, len(catalog))
	for key := range catalog {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		text := catalog[key]
		englishText, isKnown := english[key]
		switch {
		case !isKnown && !(isAppTextKey(key) && !hasEnglish):
			warnings = append(warnings, Warning(fmt.Sprintf("%s: unknown key %q ignored", where, key)))
		case isKnown && locale != i18n.SourceLocale:
			problems = append(problems, checkTranslation(where, key, englishText, text)...)
		}
		if isAppTextKey(key) && !hasEnglish {
			problems = append(problems, fmt.Sprintf("%s: key %q needs the English text in %s/en.json", where, key, LocalesDirName))
		}
	}
	return problems, warnings
}

func checkTranslation(where, key, englishText, text string) []string {
	var problems []string
	if mismatch := i18n.CheckPlaceholders(englishText, text); mismatch != "" {
		problems = append(problems, fmt.Sprintf("%s: %q %s", where, key, mismatch))
	}
	for _, problem := range validateTranslatedField(key, text) {
		problems = append(problems, fmt.Sprintf("%s: %s", where, problem))
	}
	return problems
}

// validateTranslatedField applies the length and ':' rules of the English field.
func validateTranslatedField(key, text string) []string {
	switch {
	case key == keyName:
		return validateText(key, text, 1, maxNameLen)
	case key == keyDescription:
		if utf8.RuneCountInString(text) > maxDescriptionN {
			return []string{fmt.Sprintf("description is longer than %d characters; shorten it", maxDescriptionN)}
		}
	case strings.HasPrefix(key, "actions.") && strings.HasSuffix(key, ".confirm"):
		return validateText(key, text, 1, maxDescriptionN)
	case strings.HasPrefix(key, "actions."):
		return validateText(key, text, 1, maxLabelLen)
	}
	return nil
}

func sortedNames[V any](items map[string]V) []string {
	names := make([]string, 0, len(items))
	for name := range items {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
