package manifest

import (
	"strings"
	"testing"

	"koboapps/internal/i18n"
)

func loadStaged(t *testing.T, name string) App {
	t.Helper()
	app, _, err := Load(stageApp(t, name))
	if err != nil {
		t.Fatal(err)
	}
	return app
}

func TestCheckLocalesAcceptsValidTranslation(t *testing.T) {
	problems, warnings := CheckLocales(loadStaged(t, "locales-valid"))
	if len(problems) > 0 || len(warnings) > 0 {
		t.Fatalf("problems=%v warnings=%v", problems, warnings)
	}
}

func TestCheckLocalesFindsEveryProblem(t *testing.T) {
	problems, warnings := CheckLocales(loadStaged(t, "locales-bad"))
	all := strings.Join(problems, "\n")
	for _, want := range []string{
		"fr.json",
		"name \"Tên: có hai chấm\" must not contain ':'",
		"actions.go.label must be 1-32",
		"placeholders {ten} must match English {name}",
	} {
		if !strings.Contains(all, want) {
			t.Errorf("problems lack %q:\n%s", want, all)
		}
	}
	if len(warnings) != 1 || !strings.Contains(string(warnings[0]), `unknown key "extra.key"`) {
		t.Errorf("warnings = %v", warnings)
	}
}

func TestCheckLocalesRequiresEnglishForMessages(t *testing.T) {
	problems, _ := CheckLocales(loadStaged(t, "locales-no-en"))
	if len(problems) != 1 || !strings.Contains(problems[0], "locales/en.json") {
		t.Fatalf("problems = %v", problems)
	}
}

func TestAppWithoutLocalesFolderHasNothingToCheck(t *testing.T) {
	problems, warnings := CheckLocales(loadStaged(t, "valid-full"))
	if len(problems) > 0 || len(warnings) > 0 {
		t.Fatalf("problems=%v warnings=%v", problems, warnings)
	}
}

func TestLocalizeUsesTranslationsAndSkipsInvalidOnes(t *testing.T) {
	app := loadStaged(t, "locales-valid")
	bundle, err := app.LoadLocales("vi")
	if err != nil {
		t.Fatal(err)
	}
	localized := app.Localize(bundle)
	if localized.Name != "Mẫu" || localized.Actions[0].Label != "Chạy" || localized.Description != "Demo app" {
		t.Fatalf("localized = %+v", localized)
	}
	if app.Name != "Demo" {
		t.Fatal("Localize must not modify the original")
	}
	broken := i18n.NewBundle(i18n.Catalog{"name": "Bad:Name", "actions.go.label": ""})
	fallback := app.Localize(broken)
	if fallback.Name != "Demo" || fallback.Actions[0].Label != "Go" {
		t.Fatalf("invalid translations must fall back: %+v", fallback)
	}
}

// Texts that ui.json references as "@ui.key" are app-defined keys, like "msg.*": known, not warned about.
func TestCheckLocalesKnowsScreenTextKeys(t *testing.T) {
	app := loadStaged(t, "locales-valid")
	english := englishStrings(app, i18n.Catalog{"ui.stop": "Stop", "msg.hello": "Hi"})
	if english["ui.stop"] != "Stop" || english["msg.hello"] != "Hi" {
		t.Fatalf("english reference lacks app-defined keys: %v", english)
	}
	problems, warnings := checkCatalog("vi", i18n.Catalog{"ui.stop": "Dừng"}, english, app, true)
	if len(problems) > 0 || len(warnings) > 0 {
		t.Fatalf("problems=%v warnings=%v", problems, warnings)
	}
	_, warnings = checkCatalog("vi", i18n.Catalog{"other.stop": "x"}, english, app, true)
	if len(warnings) != 1 {
		t.Fatalf("a key with an unknown prefix must still warn: %v", warnings)
	}
}
