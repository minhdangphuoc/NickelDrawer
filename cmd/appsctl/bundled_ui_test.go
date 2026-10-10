package main

import (
	"path/filepath"
	"testing"

	"koboapps/internal/i18n"
	"koboapps/internal/manifest"
	"koboapps/internal/uikit"
)

// Every shipped native screen, the apps' and the built-in settings, passes the validator with no
// errors and no warnings: the content-style rules (no typed capitals, status within 340 px, no
// full stop on a caption, no section over a lone paragraph, A8 promotion) hold for what we ship.
func TestBundledScreensValidateWithoutWarnings(t *testing.T) {
	manifests, err := filepath.Glob(filepath.Join("..", "..", "apps", "*", manifest.ManifestName))
	if err != nil || len(manifests) == 0 {
		t.Fatalf("no apps found: %v", err)
	}
	for _, path := range manifests {
		dir := filepath.Dir(path)
		t.Run(filepath.Base(dir), func(t *testing.T) {
			app, _, loadErr := manifest.Load(dir)
			if loadErr != nil {
				t.Fatal(loadErr)
			}
			if !uikit.HasUI(dir) {
				return
			}
			var actionIDs []string
			for _, action := range app.Actions {
				actionIDs = append(actionIDs, action.ID)
			}
			_, report, err := uikit.LoadDir(dir, actionIDs)
			if err != nil || len(report.Errors) > 0 || len(report.Warnings) > 0 {
				t.Fatalf("err=%v errors=%v warnings=%v", err, report.Errors, report.Warnings)
			}
		})
	}
}

// The host's own strings follow the same rule: sentence case in the files, the renderer uppercases
// section labels ("Installed", "Problems"), so no host text is typed in capitals.
func TestHostLocalesAreNotTypedInCapitals(t *testing.T) {
	for _, file := range []string{"en", "vi"} {
		catalog, err := i18n.LoadCatalog(filepath.Join("..", "..", "host", "locales", file+".json"))
		if err != nil {
			t.Fatal(err)
		}
		for key, text := range catalog {
			if words := uikit.UppercaseWords(text); len(words) > 0 {
				t.Errorf("host/locales/%s.json %s = %q is typed in capitals (%v)", file, key, text, words)
			}
		}
	}
}
