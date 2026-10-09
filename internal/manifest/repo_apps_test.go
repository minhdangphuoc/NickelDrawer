package manifest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"koboapps/internal/i18n"
)

const repoRoot = "../.."

// Keeps spec/app.schema.json, the Go validator and the shipped apps in sync:
// every app in apps/ must pass Load, and the schema must stay well-formed.
func TestRepoAppsAreValid(t *testing.T) {
	manifests, err := filepath.Glob(filepath.Join(repoRoot, "apps", "*", ManifestName))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range manifests {
		dir := filepath.Dir(path)
		t.Run(filepath.Base(dir), func(t *testing.T) {
			app, _, err := Load(dir)
			if err != nil {
				t.Fatal(err)
			}
			if app.Storage == nil {
				t.Errorf("%s must declare its storage reads/writes and network use in app.json", app.ID)
			}
			if info, err := os.Stat(filepath.Join(dir, "icon.png")); err != nil || !info.Mode().IsRegular() {
				t.Errorf("%s needs an icon.png (96x96 outline, see docs/spec/app-format.md)", app.ID)
			}
			problems, warnings := CheckLocales(app)
			if len(problems) > 0 || len(warnings) > 0 {
				t.Fatalf("locale problems=%v warnings=%v", problems, warnings)
			}
		})
	}
}

var (
	messageKeyInSource = regexp.MustCompile(`"(msg\.[a-z0-9_]+)"`)
	messageKeyInScript = regexp.MustCompile(`\bmsg\s+\S+\s+(msg\.[a-z0-9_]+)`)
)

// Every message key used by an app's scripts or Go sources must exist in its English file,
// and every shipped translation must define the same keys.
func TestShippedAppsDefineEveryMessageKeyTheyUse(t *testing.T) {
	manifests, _ := filepath.Glob(filepath.Join(repoRoot, "apps", "*", ManifestName))
	for _, path := range manifests {
		appID := filepath.Base(filepath.Dir(path))
		t.Run(appID, func(t *testing.T) {
			english, err := i18n.LoadCatalog(filepath.Join(repoRoot, "apps", appID, LocalesDirName, "en.json"))
			if err != nil {
				t.Fatal(err)
			}
			for _, key := range usedMessageKeys(t, appID) {
				if _, found := english[key]; !found {
					t.Errorf("%s uses %s but locales/en.json does not define it", appID, key)
				}
			}
			assertTranslationsComplete(t, appID, english)
		})
	}
}

func assertTranslationsComplete(t *testing.T, appID string, english i18n.Catalog) {
	t.Helper()
	files, _ := filepath.Glob(filepath.Join(repoRoot, "apps", appID, LocalesDirName, "*.json"))
	for _, file := range files {
		translated, err := i18n.LoadCatalog(file)
		if err != nil {
			t.Fatal(err)
		}
		for key := range english {
			if _, found := translated[key]; !found && strings.HasPrefix(key, messagePrefix) {
				t.Errorf("%s lacks %s", filepath.Base(file), key)
			}
		}
	}
}

// usedMessageKeys collects keys from the app's shell scripts and, for ghsync, its Go sources.
func usedMessageKeys(t *testing.T, appID string) []string {
	t.Helper()
	var keys []string
	scripts, _ := filepath.Glob(filepath.Join(repoRoot, "apps", appID, "bin", "*.sh"))
	for _, script := range scripts {
		keys = append(keys, matchKeys(t, script, messageKeyInScript)...)
	}
	if appID == "ghsync" {
		sources, _ := filepath.Glob(filepath.Join(repoRoot, "internal", "ghsync", "*.go"))
		mainSource, _ := filepath.Glob(filepath.Join(repoRoot, "cmd", "ghsync", "*.go"))
		for _, source := range append(sources, mainSource...) {
			if !strings.HasSuffix(source, "_test.go") {
				keys = append(keys, matchKeys(t, source, messageKeyInSource)...)
			}
		}
	}
	return keys
}

func matchKeys(t *testing.T, path string, pattern *regexp.Regexp) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var keys []string
	for _, match := range pattern.FindAllStringSubmatch(string(data), -1) {
		keys = append(keys, match[1])
	}
	return keys
}

func TestSchemaFileIsWellFormed(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoRoot, "spec", "app.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Dialect  string   `json:"$schema"`
		Required []string `json:"required"`
	}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("schema is not valid JSON: %v", err)
	}
	if schema.Dialect != "https://json-schema.org/draft/2020-12/schema" || len(schema.Required) != 5 {
		t.Fatalf("unexpected schema header: %+v", schema)
	}
}
