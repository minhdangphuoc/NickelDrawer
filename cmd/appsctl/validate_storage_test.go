package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestValidateRejectsUnknownStorageNames(t *testing.T) {
	appsDir := t.TempDir()
	writeApp(t, appsDir, "good-app", `{"schema":1,"id":"good-app","name":"Good","version":"1.0.0",
 "storage":{"reads":["pictures"],"writes":["books","inbox"]},"network":true,
 "actions":[{"id":"go","label":"Run","run":"bin/run.sh","output":"show"}]}`)
	writeApp(t, appsDir, "bad-app", `{"schema":1,"id":"bad-app","name":"Bad","version":"1.0.0",
 "storage":{"reads":["passwords"],"writes":["/etc"]},
 "actions":[{"id":"go","label":"Run","run":"bin/run.sh","output":"show"}]}`)

	var out, errOut bytes.Buffer
	code := run([]string{"validate", "--apps-dir", appsDir}, &out, &errOut)

	if code != exitInvalid {
		t.Fatalf("exit code %d, want %d", code, exitInvalid)
	}
	text := out.String()
	for _, want := range []string{"ok      good-app", "INVALID bad-app", `unknown folder "passwords"`, `unknown folder "/etc"`} {
		if !strings.Contains(text, want) {
			t.Errorf("output lacks %q:\n%s", want, text)
		}
	}
}

func TestEveryShippedAppValidatesWithItsStorageDeclaration(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"validate", "--apps-dir", "../../apps"}, &out, &errOut); code != exitOK {
		t.Fatalf("shipped apps do not validate (exit %d):\n%s%s", code, out.String(), errOut.String())
	}
	for _, app := range []string{"localsend", "ghsync", "wallpaper"} {
		if !strings.Contains(out.String(), "ok      "+app) {
			t.Errorf("%s did not validate:\n%s", app, out.String())
		}
	}
}
