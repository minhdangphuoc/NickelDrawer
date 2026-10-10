package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestProjectUpdateConfigPrintsTheDefaultUpdateJSON(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"project", "update-config", "--file", "../../host/config/project.json"}, &out, &errOut)
	if code != exitOK {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	var config map[string]string
	if err := json.Unmarshal(out.Bytes(), &config); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out.String())
	}
	if config["repo"] != "minhdangphuoc/NickelDrawer" || config["channel"] != "stable" {
		t.Fatalf("config = %v", config)
	}
}

func TestProjectCommandFailsWithAHintOnBadInput(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"project"}, &out, &errOut); code != exitFailure || !strings.Contains(errOut.String(), "usage") {
		t.Fatalf("no subcommand: exit %d, %q", code, errOut.String())
	}
	errOut.Reset()
	if code := run([]string{"project", "update-config", "--file", "missing.json"}, &out, &errOut); code != exitFailure ||
		!strings.Contains(errOut.String(), "missing.json") {
		t.Fatalf("missing file: exit %d, %q", code, errOut.String())
	}
}
