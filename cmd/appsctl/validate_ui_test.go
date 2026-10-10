package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const uiTestManifest = `{"schema":1,"id":"demo","name":"Demo","version":"1.0.0",
 "actions":[{"id":"go","label":"Run","run":"bin/run.sh","output":"show"}]}`

func validateWithUI(t *testing.T, uiJSON string) (int, string) {
	t.Helper()
	appsDir := t.TempDir()
	writeApp(t, appsDir, "demo", uiTestManifest)
	if err := os.WriteFile(filepath.Join(appsDir, "demo", "ui.json"), []byte(uiJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	code := run([]string{"validate", filepath.Join(appsDir, "demo")}, &out, &errOut)
	return code, out.String()
}

func uiWith(body, footer string) string {
	return `{"kit":1,"start":"main","vars":{"pin":"","items":[]},"screens":{"main":{"title":"Demo","body":` + body + `,"footer":` + footer + `}}}`
}

func TestValidateAcceptsAValidUI(t *testing.T) {
	code, text := validateWithUI(t, uiWith(`[{"bigcode":"$pin"}]`, `[{"button":"Run","action":"go","style":"primary"}]`))
	if code != exitOK || !strings.Contains(text, "ok      demo (1 actions, ui.json)") {
		t.Fatalf("exit %d:\n%s", code, text)
	}
}

func TestValidateRejectsBrokenUI(t *testing.T) {
	cases := map[string]struct{ body, footer, want string }{
		"unknown component":   {`[{"carousel":"x"}]`, `[]`, "ui.unknown_component"},
		"missing required":    {`[{"toggle":"Wi-Fi"}]`, `[]`, "ui.missing_prop"},
		"bad binding":         {`[{"text":"costs $5"}]`, `[]`, "ui.bad_binding"},
		"action not in app":   {`[]`, `[{"button":"Run","action":"fly"}]`, "ui.unknown_action"},
		"two primary buttons": {`[]`, `[{"button":"A","action":"go","style":"primary"},{"button":"B","action":"go","style":"primary"}]`, "ui.multiple_primary"},
		"missing locale key":  {`[{"text":"@ui.nope"}]`, `[]`, "ui.missing_locale_key"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			code, text := validateWithUI(t, uiWith(tc.body, tc.footer))
			if code != exitInvalid || !strings.Contains(text, "INVALID demo: ui.json") || !strings.Contains(text, tc.want) {
				t.Fatalf("exit %d, want %s:\n%s", code, tc.want, text)
			}
		})
	}
}

func TestShippedUIScreensValidate(t *testing.T) {
	var out, errOut bytes.Buffer
	for _, dir := range []string{"../../apps/localsend", "../../apps/wallpaper", "../../apps/todo"} {
		if code := run([]string{"validate", dir}, &out, &errOut); code != exitOK {
			t.Fatalf("%s: exit %d\n%s%s", dir, code, out.String(), errOut.String())
		}
	}
	if strings.Count(out.String(), ", ui.json)") != 3 {
		t.Errorf("each of the three apps should report its ui.json:\n%s", out.String())
	}
}
