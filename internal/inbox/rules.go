// Package inbox routes files received on the Kobo (LocalSend, USB) into the library folders.
//
// Everything is an allowlist: a file is moved only when its extension is a known type, its
// content (magic bytes) agrees with that type and it is within the size cap. The rules are data
// (host/config/routing.json), see docs/spec/inbox.md. Received files are never executed or unpacked.
package inbox

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"koboapps/internal/storage"
)

const rulesVersion = 1

// libraryFolders are the only destinations a rule may name: where a user expects their files.
var libraryFolders = []storage.Folder{storage.Books, storage.Documents, storage.Comics, storage.Pictures}

// Rule maps one file type to a content check, a size cap and a destination.
type Rule struct {
	Type        string         `json:"type"`
	Extensions  []string       `json:"extensions"`
	Check       string         `json:"check"`
	MaxBytes    int64          `json:"max_bytes"`
	Destination storage.Folder `json:"destination"`
}

// Rules is the validated allowlist.
type Rules struct {
	Version int    `json:"version"`
	Rules   []Rule `json:"rules"`
}

// Override is data/routing.json. It can only narrow the shipped rules or move a type to
// another library folder; it cannot add types, extensions, checks or size caps.
type Override struct {
	Remove       []string          `json:"remove"`
	Destinations map[string]string `json:"destinations"`
}

// RulesError says which rules file is unusable and why.
type RulesError struct {
	File  string
	Cause error
}

func (err *RulesError) Error() string { return fmt.Sprintf("%s: %v", err.File, err.Cause) }
func (err *RulesError) Unwrap() error { return err.Cause }

// ShippedRulesPath is the rules file that ships with the host.
func ShippedRulesPath() string {
	return filepath.Join(storage.KoboappsDir(), "config", "routing.json")
}

// OverrideRulesPath is the optional user file; the installer never writes it.
func OverrideRulesPath() string {
	return filepath.Join(storage.KoboappsDir(), "data", "routing.json")
}

// LoadRules reads the shipped rules and applies the override when there is one.
//
// A missing or broken shipped file is an error (routing without rules would be unsafe). A
// missing override is normal; a broken one is reported in overrideErr and the shipped defaults
// are used unchanged.
func LoadRules(shippedPath, overridePath string) (rules Rules, overrideErr error, err error) {
	rules, err = readShipped(shippedPath)
	if err != nil {
		return Rules{}, nil, &RulesError{File: shippedPath, Cause: err}
	}
	override, found, err := readOverride(overridePath)
	if err != nil {
		return rules, &RulesError{File: overridePath, Cause: err}, nil
	}
	if !found {
		return rules, nil, nil
	}
	narrowed, err := rules.apply(override)
	if err != nil {
		return rules, &RulesError{File: overridePath, Cause: err}, nil
	}
	return narrowed, nil, nil
}

func readShipped(path string) (Rules, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Rules{}, fmt.Errorf("cannot read the routing rules (reinstall koboapps): %w", err)
	}
	var rules Rules
	if err := decodeStrict(data, &rules); err != nil {
		return Rules{}, err
	}
	return rules, rules.validate()
}

func readOverride(path string) (Override, bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Override{}, false, nil
	}
	if err != nil {
		return Override{}, false, fmt.Errorf("cannot read the override: %w", err)
	}
	var override Override
	if err := decodeStrict(data, &override); err != nil {
		return Override{}, false, err
	}
	return override, true, nil
}

// decodeStrict rejects unknown fields and trailing data, so a typo never silently changes nothing.
func decodeStrict(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	if decoder.More() {
		return errors.New("invalid JSON: unexpected data after the first object")
	}
	return nil
}

func (rules Rules) validate() error {
	if rules.Version != rulesVersion {
		return fmt.Errorf("version must be %d (got %d)", rulesVersion, rules.Version)
	}
	seenTypes := map[string]bool{}
	seenExtensions := map[string]bool{}
	for _, rule := range rules.Rules {
		if err := rule.validate(seenTypes, seenExtensions); err != nil {
			return fmt.Errorf("rule %q: %w", rule.Type, err)
		}
	}
	return nil
}

func (rule Rule) validate(seenTypes, seenExtensions map[string]bool) error {
	if rule.Type == "" || seenTypes[rule.Type] {
		return errors.New("type must be set and unique")
	}
	seenTypes[rule.Type] = true
	if !isKnownCheck(rule.Check) {
		return fmt.Errorf("unknown check %q", rule.Check)
	}
	if rule.MaxBytes <= 0 {
		return errors.New("max_bytes must be positive")
	}
	if !isLibraryFolder(string(rule.Destination)) {
		return fmt.Errorf("destination %q must be one of %s", rule.Destination, libraryFolderNames())
	}
	if len(rule.Extensions) == 0 {
		return errors.New("needs at least one extension")
	}
	for _, extension := range rule.Extensions {
		if len(extension) < 2 || extension[0] != '.' || extension != strings.ToLower(extension) {
			return fmt.Errorf("extension %q must be lowercase and start with a dot", extension)
		}
		if seenExtensions[extension] {
			return fmt.Errorf("extension %q is used by two rules", extension)
		}
		seenExtensions[extension] = true
	}
	return nil
}

// apply returns a copy of the rules with the override applied; the receiver is unchanged.
func (rules Rules) apply(override Override) (Rules, error) {
	known := map[string]bool{}
	for _, rule := range rules.Rules {
		known[rule.Type] = true
	}
	removed := map[string]bool{}
	for _, name := range override.Remove {
		if !known[name] {
			return rules, fmt.Errorf("remove: unknown type %q; known types are %s", name, rules.typeNames())
		}
		removed[name] = true
	}
	for name, folder := range override.Destinations {
		if !known[name] {
			return rules, fmt.Errorf("destinations: unknown type %q; known types are %s", name, rules.typeNames())
		}
		if !isLibraryFolder(folder) {
			return rules, fmt.Errorf("destinations: %q is not a known folder; use one of %s", folder, libraryFolderNames())
		}
	}
	narrowed := Rules{Version: rules.Version}
	for _, rule := range rules.Rules {
		if removed[rule.Type] {
			continue
		}
		if folder, found := override.Destinations[rule.Type]; found {
			rule.Destination = storage.Folder(folder)
		}
		narrowed.Rules = append(narrowed.Rules, rule)
	}
	return narrowed, nil
}

func (rules Rules) typeNames() string {
	names := make([]string, 0, len(rules.Rules))
	for _, rule := range rules.Rules {
		names = append(names, rule.Type)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

// Match finds the rule whose extension is the longest suffix of name ("a.kepub.epub" beats ".epub").
// extension is returned in lower case.
func (rules Rules) Match(name string) (rule Rule, extension string, found bool) {
	lower := strings.ToLower(name)
	for _, candidate := range rules.Rules {
		for _, suffix := range candidate.Extensions {
			if strings.HasSuffix(lower, suffix) && len(suffix) > len(extension) {
				rule, extension, found = candidate, suffix, true
			}
		}
	}
	return rule, extension, found
}

// TypeFor returns the rule type of a known extension, "" otherwise.
func (rules Rules) TypeFor(name string) string {
	rule, _, found := rules.Match(name)
	if !found {
		return ""
	}
	return rule.Type
}

func isLibraryFolder(name string) bool {
	for _, folder := range libraryFolders {
		if string(folder) == name {
			return true
		}
	}
	return false
}

func libraryFolderNames() string {
	names := make([]string, 0, len(libraryFolders))
	for _, folder := range libraryFolders {
		names = append(names, string(folder))
	}
	return strings.Join(names, ", ")
}
