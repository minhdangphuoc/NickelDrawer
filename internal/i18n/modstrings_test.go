package i18n

import (
	"os"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// The native mod reads its own texts ("mod.*" keys) from host/locales and keeps the same texts
// compiled in as a last resort (mod/drawer/fallback_strings.cc). These tests keep the two
// from drifting and keep the compiled table honest.

const (
	fallbackStringsPath = "../../mod/drawer/fallback_strings.cc"
	hostLocalesDir      = "../../host/locales"
	modKeyPrefix        = "mod."
)

var (
	tableBlock = regexp.MustCompile(`(?s)static const Entry TABLE\[\] = \{(.*?)\n\};`)
	cString    = regexp.MustCompile(`"((?:[^"\\]|\\.)*)"`)
	cEscape    = regexp.MustCompile(`\\(.)`)
)

type modEntry struct{ key, english, vietnamese string }

func unescapeC(text string) string {
	return cEscape.ReplaceAllString(text, "$1")
}

func compiledEntries(t *testing.T) []modEntry {
	t.Helper()
	source, err := os.ReadFile(fallbackStringsPath)
	if err != nil {
		t.Fatalf("cannot read the mod's fallback table: %v", err)
	}
	block := tableBlock.FindSubmatch(source)
	if block == nil {
		t.Fatalf("no TABLE in %s; update modstrings_test.go if its layout changed", fallbackStringsPath)
	}
	var literals []string
	for _, match := range cString.FindAllSubmatch(block[1], -1) {
		literals = append(literals, unescapeC(string(match[1])))
	}
	if len(literals)%3 != 0 || len(literals) == 0 {
		t.Fatalf("TABLE has %d string literals; each entry is key, English, Vietnamese", len(literals))
	}
	var entries []modEntry
	for index := 0; index < len(literals); index += 3 {
		entries = append(entries, modEntry{literals[index], literals[index+1], literals[index+2]})
	}
	return entries
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func hostCatalog(t *testing.T, language string) Catalog {
	t.Helper()
	catalog, err := LoadCatalog(hostLocalesDir + "/" + language + ".json")
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}

func TestCompiledTableEqualsTheHostLocaleFiles(t *testing.T) {
	english, vietnamese := hostCatalog(t, "en"), hostCatalog(t, "vi")
	for _, entry := range compiledEntries(t) {
		if got := english[entry.key]; got != entry.english {
			t.Errorf("%s: host/locales/en.json has %q, the compiled table %q", entry.key, got, entry.english)
		}
		if got := vietnamese[entry.key]; got != entry.vietnamese {
			t.Errorf("%s: host/locales/vi.json has %q, the compiled table %q", entry.key, got, entry.vietnamese)
		}
	}
}

func TestEveryModKeyInTheLocaleFilesIsCompiledIn(t *testing.T) {
	compiled := map[string]bool{}
	for _, entry := range compiledEntries(t) {
		compiled[entry.key] = true
	}
	for _, language := range []string{"en", "vi"} {
		for key := range hostCatalog(t, language) {
			if strings.HasPrefix(key, modKeyPrefix) && !compiled[key] {
				t.Errorf("%s has %s that the mod does not know; add it to TABLE or delete it", language, key)
			}
		}
	}
}

func TestModStringsKeepTheirPlaceholdersAndStayShort(t *testing.T) {
	for _, entry := range compiledEntries(t) {
		if problem := CheckPlaceholders(entry.english, entry.vietnamese); problem != "" {
			t.Errorf("%s: %s", entry.key, problem)
		}
		for _, text := range []string{entry.english, entry.vietnamese} {
			if !norm.NFC.IsNormalString(text) {
				t.Errorf("%s: %q is not NFC", entry.key, text)
			}
			if utf8.RuneCountInString(text) > 80 {
				t.Errorf("%s: %q is longer than 80 characters; the screen is small", entry.key, text)
			}
		}
	}
}

func TestModStringsAreFreeOfInternalJargon(t *testing.T) {
	for _, entry := range compiledEntries(t) {
		for _, text := range []string{entry.english, entry.vietnamese} {
			if strings.Contains(strings.ToLower(text), "appsctl") || strings.Contains(text, "(code") || strings.Contains(text, "(mã") {
				t.Errorf("%s: %q shows a tool name or an exit code to the reader; log the code instead", entry.key, text)
			}
		}
	}
}
