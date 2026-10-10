package drawer

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"koboapps/internal/i18n"
)

// writeRequestApp installs an app whose "go" action prints scriptOutput and whose state prints
// stateJSON. requests is the raw JSON array of the manifest's "requests" ("" for none).
func writeRequestApp(t *testing.T, appsDir, requests string) {
	t.Helper()
	field := ""
	if requests != "" {
		field = `"requests":` + requests + `,`
	}
	manifest := `{"schema":1,"id":"demo","name":"Demo","version":"1.0.0",` + field + `"actions":[
 {"id":"go","label":"Go","run":"bin/go.sh","output":"show"},
 {"id":"state","label":"State","run":"bin/state.sh","output":"show","menu":false}]}`
	ui := `{"kit":1,"start":"main","vars":{"x":""},"screens":{"main":{"title":"Demo","body":[{"text":"hi"}]}}}`
	for path, content := range map[string]string{
		"app.json": manifest, "ui.json": ui, "bin/go.sh": "#!/bin/sh\n", "bin/state.sh": "#!/bin/sh\n",
		"locales/en.json": `{}`,
	} {
		full := filepath.Join(appsDir, "demo", filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDeclaredRequestsReachTheReply(t *testing.T) {
	config, _ := testConfig(t, stateRunner("{}",
		`{"request":"rescan","mode":"full"}`+"\n"+`{"request":"wifi"}`+"\n"+`{"toast":"saved"}`))
	writeRequestApp(t, config.AppsDir, `["rescan","wifi"]`)

	result, err := config.Run("demo", "go")
	if err != nil {
		t.Fatal(err)
	}
	want := []Request{{Name: "rescan", Mode: "full"}, {Name: "wifi"}}
	if !reflect.DeepEqual(result.Requests, want) {
		t.Fatalf("requests = %+v, want %+v", result.Requests, want)
	}
	if result.Text != "" {
		t.Fatalf("text = %q: a signal must never reach the screen as prose", result.Text)
	}
}

func TestRescanModeDefaultsToQuick(t *testing.T) {
	config, _ := testConfig(t, stateRunner("{}", `{"request":"rescan"}`))
	writeRequestApp(t, config.AppsDir, `["rescan"]`)
	result, _ := config.Run("demo", "go")
	if want := []Request{{Name: "rescan", Mode: "quick"}}; !reflect.DeepEqual(result.Requests, want) {
		t.Fatalf("requests = %+v, want %+v", result.Requests, want)
	}
}

func TestUndeclaredOrInvalidRequestsAreDropped(t *testing.T) {
	cases := map[string]string{
		"not in the allowlist": `{"request":"wifi"}`,
		"unknown request":      `{"request":"reboot"}`,
		"unknown mode":         `{"request":"rescan","mode":"deep"}`,
	}
	for name, output := range cases {
		t.Run(name, func(t *testing.T) {
			config, _ := testConfig(t, stateRunner("{}", output))
			writeRequestApp(t, config.AppsDir, `["rescan"]`)
			var debug strings.Builder
			config.DebugLog = &debug
			result, err := config.Run("demo", "go")
			if err != nil || len(result.Requests) != 0 {
				t.Fatalf("result = %+v err = %v", result, err)
			}
			if !strings.Contains(debug.String(), "dropped") {
				t.Errorf("a dropped request must be logged, log = %q", debug.String())
			}
		})
	}
}

func TestRequestsWorkWithoutAScreen(t *testing.T) {
	config, _ := testConfig(t, stateRunner("{}", `{"request":"wifi"}`))
	writeRequestApp(t, config.AppsDir, `["wifi"]`)
	result, err := config.RunEvent(Event{AppID: "demo", ActionID: "go"})
	if err != nil || len(result.Requests) != 1 || result.Toast != "" {
		t.Fatalf("result = %+v err = %v", result, err)
	}
}

func TestStateRequestsFromAFinishedJobReachTheScreenReply(t *testing.T) {
	config, _ := testConfig(t, stateRunner(
		`{"x":"1","requests":[{"request":"rescan","mode":"quick"},{"request":"wifi"},{"request":"rescan"}]}`, ""))
	writeRequestApp(t, config.AppsDir, `["rescan"]`)
	reply, err := config.Screen("demo", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if want := []Request{{Name: "rescan", Mode: "quick"}}; !reflect.DeepEqual(reply.Requests, want) {
		t.Fatalf("requests = %+v, want one quick rescan (wifi undeclared, duplicate collapsed)", reply.Requests)
	}
}

func TestProseAroundASignalIsNotASignal(t *testing.T) {
	config, _ := testConfig(t, stateRunner("{}", "Starting...\n"+`{"request":"wifi"}`))
	writeRequestApp(t, config.AppsDir, `["wifi"]`)
	result, _ := config.Run("demo", "go")
	if len(result.Requests) != 0 || result.Text != "" {
		t.Fatalf("result = %+v", result)
	}
}

func TestNeedsNewerHostShowsAsStatus(t *testing.T) {
	config, _ := testConfig(t, func(context.Context, string) ScriptOutput { return ScriptOutput{Stdout: "ready"} })
	config.Version = "1.0.0"
	bundle, err := i18n.LoadBundle(filepath.Join("..", "..", "host", "locales"), "en")
	if err != nil {
		t.Fatal(err)
	}
	config.Bundle = bundle
	writeStatusApp(t, config.AppsDir, "old")
	writeStatusApp(t, config.AppsDir, "new")
	manifestPath := filepath.Join(config.AppsDir, "new", "app.json")
	data, _ := os.ReadFile(manifestPath)
	patched := strings.Replace(string(data), `"version":"1.0.0",`, `"version":"1.0.0","min_host":"1.2.0",`, 1)
	if err := os.WriteFile(manifestPath, []byte(patched), 0o644); err != nil {
		t.Fatal(err)
	}
	model, err := config.BuildModel()
	if err != nil {
		t.Fatal(err)
	}
	statuses := map[string]string{}
	for _, item := range model.Sections[0].Items {
		statuses[item.ID] = item.Status
	}
	if statuses["old"] != "ready" || statuses["new"] != "Needs a newer version" {
		t.Fatalf("statuses = %v", statuses)
	}
}

func writeListMenuApp(t *testing.T, appsDir string) {
	t.Helper()
	manifest := `{"schema":1,"id":"demo","name":"Demo","version":"1.0.0","actions":[
 {"id":"remove","label":"Remove","run":"bin/go.sh","output":"show"},
 {"id":"other","label":"Other","run":"bin/go.sh","output":"show"},
 {"id":"state","label":"State","run":"bin/state.sh","output":"show"}]}`
	ui := `{"kit":1,"start":"main","vars":{"items":[]},"screens":{"main":{"title":"Demo","body":[
 {"list":"$items","menu":[{"id":"remove","label":"Remove","confirm":true}]}]}}}`
	for path, content := range map[string]string{
		"app.json": manifest, "ui.json": ui, "bin/go.sh": "#!/bin/sh\n", "bin/state.sh": "#!/bin/sh\n", "locales/en.json": `{}`,
	} {
		full := filepath.Join(appsDir, "demo", filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

func TestListRowMenuIsSentAndItsEventCarriesListIndexAndValue(t *testing.T) {
	config, _ := testConfig(t, nil) // the real runner: the script must see its environment
	writeListMenuApp(t, config.AppsDir)
	script := filepath.Join(config.AppsDir, "demo", "bin", "go.sh")
	envDump := filepath.Join(t.TempDir(), "env.txt")
	body := "#!/bin/sh\nenv > " + envDump + "\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}

	reply, err := config.Screen("demo", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	menu, ok := findComponent(t, reply.Body, "list", 0).Menu.([]ListMenuItem)
	if !ok || len(menu) != 1 || menu[0].ID != "remove" || menu[0].Label != "Remove" || !menu[0].Confirm {
		t.Fatalf("menu = %+v", menu)
	}

	value := "b"
	if _, err := config.RunEvent(Event{AppID: "demo", ActionID: "remove", ScreenID: "main", Value: &value, List: "items", Index: 1}); err != nil {
		t.Fatal(err)
	}
	dump, err := os.ReadFile(envDump)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"NICKELDRAWER_LIST=items", "NICKELDRAWER_INDEX=1", "NICKELDRAWER_VALUE=b"} {
		if !strings.Contains(string(dump), want) {
			t.Errorf("script env lacks %s:\n%s", want, dump)
		}
	}
}

func TestMenuEventCannotReachAnActionTheListDoesNotOffer(t *testing.T) {
	config, _ := testConfig(t, stateRunner("{}", ""))
	writeListMenuApp(t, config.AppsDir)
	_, err := config.RunEvent(Event{AppID: "demo", ActionID: "other", ScreenID: "main", List: "items"})
	if err == nil {
		t.Fatal("an action outside the list menu must be refused")
	}
	if _, err = config.RunEvent(Event{AppID: "demo", ActionID: "remove", ScreenID: "main", List: "nolist"}); err == nil {
		t.Fatal("an unknown list must be refused")
	}
}
