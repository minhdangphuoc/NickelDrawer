// Package i18n loads flat JSON locale files and resolves messages for the device language
// (see docs/spec/app-format.md, "Localization"). English is the source language.
package i18n

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	// SourceLocale is the language every string is written in first; it is the last fallback.
	SourceLocale = "en"
	// DefaultConfPath is the Nickel settings file holding the device language.
	DefaultConfPath = "/mnt/onboard/.kobo/Kobo/Kobo eReader.conf"

	confSection = "[ApplicationPreferences]"
	confKey     = "CurrentLocale"
)

var (
	placeholderPattern = regexp.MustCompile(`\{([A-Za-z][A-Za-z0-9_]*)\}`)
	localePattern      = regexp.MustCompile(`^[A-Za-z]{2,3}([_-][A-Za-z0-9]{2,8})?$`)
)

// Catalog maps message keys to text in one language.
type Catalog map[string]string

// LoadCatalog reads one flat JSON file of string values.
func LoadCatalog(path string) (Catalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read locale file %s: %w", path, err)
	}
	var catalog Catalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("locale file %s must be a flat JSON object of strings: %w", path, err)
	}
	for key, text := range catalog {
		catalog[key] = strings.ReplaceAll(text, `\n`, "\n")
	}
	return catalog, nil
}

// CurrentLocale returns the device language from Kobo eReader.conf, or "en" when
// the file is missing, unreadable or has no usable value.
func CurrentLocale(confPath string) string {
	data, err := os.ReadFile(confPath)
	if err != nil {
		return SourceLocale
	}
	inSection := false
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") {
			inSection = line == confSection
			continue
		}
		name, value, found := strings.Cut(line, "=")
		if inSection && found && strings.TrimSpace(name) == confKey {
			return normalizeLocale(value)
		}
	}
	return SourceLocale
}

// normalizeLocale rejects anything that is not a plain locale code, because the value
// becomes part of a file name.
func normalizeLocale(value string) string {
	value = strings.Trim(strings.TrimSpace(value), `"`)
	if !localePattern.MatchString(value) {
		return SourceLocale
	}
	return strings.ReplaceAll(value, "-", "_")
}

// Chain lists the locale files to try, most specific first: pt_BR, pt, en.
func Chain(locale string) []string {
	locale = normalizeLocale(locale)
	chain := []string{locale}
	if language, _, found := strings.Cut(locale, "_"); found {
		chain = append(chain, language)
	}
	if chain[len(chain)-1] != SourceLocale {
		chain = append(chain, SourceLocale)
	}
	return chain
}

// Bundle resolves keys through a chain of catalogs, key by key.
type Bundle struct {
	catalogs []Catalog
}

// NewBundle builds a bundle; the first catalog wins.
func NewBundle(catalogs ...Catalog) *Bundle {
	return &Bundle{catalogs: catalogs}
}

// LoadBundle reads <dir>/<locale>.json for every entry of the locale chain. A missing file is
// normal. The returned bundle is always usable; the error lists files that exist but are broken.
func LoadBundle(dir, locale string) (*Bundle, error) {
	var catalogs []Catalog
	var problems []error
	for _, name := range Chain(locale) {
		path := filepath.Join(dir, name+".json")
		catalog, err := LoadCatalog(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			problems = append(problems, err)
			continue
		}
		catalogs = append(catalogs, catalog)
	}
	return NewBundle(catalogs...), errors.Join(problems...)
}

// Lookup returns the raw template for key.
func (bundle *Bundle) Lookup(key string) (string, bool) {
	if bundle == nil {
		return "", false
	}
	for _, catalog := range bundle.catalogs {
		if text, found := catalog[key]; found {
			return text, true
		}
	}
	return "", false
}

// Message returns the text for key with {name} placeholders filled from params.
// An unknown key returns the key itself, so a missing string never breaks a caller.
func (bundle *Bundle) Message(key string, params map[string]string) string {
	template, found := bundle.Lookup(key)
	if !found {
		return key
	}
	return Substitute(template, params)
}

// Substitute replaces {name} with params[name]; placeholders without a value stay as written.
func Substitute(template string, params map[string]string) string {
	return placeholderPattern.ReplaceAllStringFunc(template, func(match string) string {
		if value, found := params[match[1:len(match)-1]]; found {
			return value
		}
		return match
	})
}

// Params builds a placeholder map from name, value pairs.
func Params(pairs ...string) map[string]string {
	params := make(map[string]string, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		params[pairs[i]] = pairs[i+1]
	}
	return params
}

// Placeholders returns the sorted, unique placeholder names used in text.
func Placeholders(text string) []string {
	seen := map[string]bool{}
	var names []string
	for _, match := range placeholderPattern.FindAllStringSubmatch(text, -1) {
		if !seen[match[1]] {
			seen[match[1]] = true
			names = append(names, match[1])
		}
	}
	sort.Strings(names)
	return names
}

// CheckPlaceholders returns "" when translated uses exactly the placeholders of english,
// otherwise a sentence naming the difference.
func CheckPlaceholders(english, translated string) string {
	want, got := Placeholders(english), Placeholders(translated)
	if strings.Join(want, ",") == strings.Join(got, ",") {
		return ""
	}
	return fmt.Sprintf("placeholders {%s} must match English {%s}", strings.Join(got, "}, {"), strings.Join(want, "}, {"))
}
