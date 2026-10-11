package locales

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"koboapps/helpers/extras/internal/tweaks"
	"koboapps/internal/i18n"
)

const stringsSourcePath = "../helper_strings.cc"

func load(t *testing.T, language string) i18n.Catalog {
	t.Helper()
	catalog, err := Embedded(language)
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}

func TestEveryLanguageHasTheSameKeysAndPlaceholders(t *testing.T) {
	english, vietnamese := load(t, "en"), load(t, "vi")
	for key, text := range english {
		translated, found := vietnamese[key]
		if !found {
			t.Errorf("vi.json lacks %q", key)
			continue
		}
		if problem := i18n.CheckPlaceholders(text, translated); problem != "" {
			t.Errorf("%s: %s", key, problem)
		}
	}
	for key := range vietnamese {
		if _, found := english[key]; !found {
			t.Errorf("vi.json has %q which en.json lacks", key)
		}
	}
}

func TestNoEmptyTexts(t *testing.T) {
	for _, language := range []string{"en", "vi"} {
		for key, text := range load(t, language) {
			if strings.TrimSpace(text) == "" {
				t.Errorf("%s.json: %q is empty", language, key)
			}
		}
	}
}

func TestEveryTweakHasItsTexts(t *testing.T) {
	for _, language := range []string{"en", "vi"} {
		catalog := load(t, language)
		for _, id := range tweaks.IDs() {
			required := []string{"tweak." + id + ".label"}
			if tweaks.HasCaption(id) {
				required = append(required, "tweak."+id+".caption")
			}
			if tweaks.HasConfirm(id) {
				required = append(required, "tweak."+id+".confirm")
			}
			for _, key := range required {
				if catalog[key] == "" {
					t.Errorf("%s.json lacks %q", language, key)
				}
			}
		}
	}
}

var (
	tableBlock = regexp.MustCompile(`(?s)static const Entry TABLE\[[^\]]*\] = \{(.*?)\n\};`)
	tableEntry = regexp.MustCompile(`\{"((?:[^"\\]|\\.)*)",\s*"((?:[^"\\]|\\.)*)"\}`)
	cEscape    = regexp.MustCompile(`\\(.)`)
)

// The native library keeps these texts compiled in as a last resort; they must equal en.json.
func TestCompiledTableEqualsEnglish(t *testing.T) {
	source, err := os.ReadFile(stringsSourcePath)
	if os.IsNotExist(err) {
		t.Skipf("%s does not exist yet", stringsSourcePath)
	}
	if err != nil {
		t.Fatal(err)
	}
	block := tableBlock.FindSubmatch(source)
	if block == nil {
		t.Fatalf("no TABLE in %s; update this test if its layout changed", stringsSourcePath)
	}
	english := load(t, "en")
	compiled := map[string]bool{}
	for _, match := range tableEntry.FindAllSubmatch(block[1], -1) {
		key, text := string(match[1]), cEscape.ReplaceAllString(string(match[2]), "$1")
		compiled[key] = true
		if english[key] != text {
			t.Errorf("TABLE %q = %q, en.json says %q", key, text, english[key])
		}
	}
	var missing []string
	for key := range english {
		if (strings.HasPrefix(key, "toast.") || strings.HasPrefix(key, "tweaks.")) && !compiled[key] {
			missing = append(missing, key)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("en.json keys missing from the compiled TABLE: %v", missing)
	}
}
