package drawer

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const screenManifest = `{"schema":1,"id":"demo","name":"Demo","version":"1.0.0","actions":[
 {"id":"go","label":"Go","run":"bin/go.sh","output":"show"},
 {"id":"mode","label":"Mode","run":"bin/go.sh","output":"show","menu":false},
 {"id":"state","label":"State","run":"bin/state.sh","output":"show","menu":false}]}`

const screenUI = `{"kit":1,"start":"main",
 "vars":{"status":"idle","mode":"a","auto":false,"name":"","pin":"","items":[],"device":{"name":"Kobo"}},
 "screens":{
  "main":{"title":"@ui.title","status":"$status","action":{"label":"More","push":"more"},
   "body":[
    {"text":"Hello $device.name, $$5"},
    {"toggle":"Auto","bind":"auto"},
    {"radio":"Mode","bind":"mode","options":[["a","A"],["b","B"]]},
    {"field":"PIN","bind":"pin","secret":true},
    {"field":"Name","bind":"name","placeholder":"type"},
    {"list":"$items","max":2,"title":"Items","empty_text":"None"},
    {"button":"Go","action":"go","enabled":"!$auto","confirm":{"text":"Sure?","ok":"Yes"}}],
   "toasts":{"saved":"Saved $name"},
   "dialogs":{"ask":{"title":"Ask","text":"Really?","ok":"Do it","action":"go"}}},
  "more":{"title":"More","body":[{"bigcode":"$pin"}]}}}`

func writeScreenApp(t *testing.T, appsDir string) {
	t.Helper()
	dir := filepath.Join(appsDir, "demo")
	for path, content := range map[string]string{
		"app.json": screenManifest, "ui.json": screenUI, "bin/go.sh": "#!/bin/sh\n", "bin/state.sh": "#!/bin/sh\n",
		"locales/en.json": `{"ui.title":"Demo"}`, "locales/vi.json": `{"ui.title":"Thử"}`,
	} {
		full := filepath.Join(dir, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

// stateRunner answers the state script with stateJSON and every other script with otherOutput.
func stateRunner(stateJSON, otherOutput string) ScriptRunner {
	return func(_ context.Context, script string) ScriptOutput {
		if strings.HasSuffix(script, "state.sh") {
			return ScriptOutput{Stdout: stateJSON}
		}
		return ScriptOutput{Stdout: otherOutput}
	}
}

func findComponent(t *testing.T, components []ScreenComponent, kind string, index int) ScreenComponent {
	t.Helper()
	for _, component := range components {
		if component.Type == kind {
			if index == 0 {
				return component
			}
			index--
		}
	}
	t.Fatalf("no %s component", kind)
	return ScreenComponent{}
}

func TestScreenResolvesStateBindings(t *testing.T) {
	config, _ := testConfig(t, stateRunner(
		`{"status":"busy","auto":true,"mode":"b","pin":"1234","items":["x",["y","2"],"z"],"undeclared":"ignored"}`, ""))
	writeScreenApp(t, config.AppsDir)
	reply, err := config.Screen("demo", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if reply.Screen != "main" || reply.Title != "Demo" || reply.Status != "busy" || !reply.Live {
		t.Errorf("header: %+v", reply)
	}
	if reply.Action == nil || reply.Action.Push != "more" {
		t.Errorf("header action: %+v", reply.Action)
	}
	if text := findComponent(t, reply.Body, "text", 0).Text; text != "Hello Kobo, $5" {
		t.Errorf("template: %q", text)
	}
	toggle := findComponent(t, reply.Body, "toggle", 0)
	if toggle.Value != true || toggle.Enabled == nil || *toggle.Enabled || toggle.Action != "" {
		t.Errorf("toggle without an action named like its variable must be read-only: %+v", toggle)
	}
	radio := findComponent(t, reply.Body, "radio", 0)
	if radio.Value != "b" || !*radio.Enabled || radio.Action != "mode" || radio.Options[1] != [2]string{"b", "B"} {
		t.Errorf("radio: %+v", radio)
	}
	if secret := findComponent(t, reply.Body, "field", 0); secret.Value != "••••" || !secret.Secret {
		t.Errorf("a secret value must never reach the mod: %+v", secret)
	}
	list := findComponent(t, reply.Body, "list", 0)
	if list.Items == nil || len(*list.Items) != 2 || (*list.Items)[0] != [2]string{"y", "2"} || (*list.Items)[1][0] != "z" {
		t.Errorf("list keeps the last max items: %+v", list.Items)
	}
	button := findComponent(t, reply.Body, "button", 0)
	if *button.Enabled || button.Style != "secondary" || button.Confirm == nil || button.Confirm.OK != "Yes" {
		t.Errorf("button: %+v", button)
	}
}

func TestScreenWithoutStateUsesDefaults(t *testing.T) {
	for name, runner := range map[string]ScriptRunner{
		"bad json":  stateRunner("not json", ""),
		"exit code": func(context.Context, string) ScriptOutput { return ScriptOutput{ExitCode: 1, Stdout: `{"status":"x"}`} },
		"timeout":   func(context.Context, string) ScriptOutput { return ScriptOutput{TimedOut: true} },
	} {
		t.Run(name, func(t *testing.T) {
			config, _ := testConfig(t, runner)
			writeScreenApp(t, config.AppsDir)
			reply, err := config.Screen("demo", "", nil)
			if err != nil {
				t.Fatal(err)
			}
			if reply.Status != "idle" || findComponent(t, reply.Body, "radio", 0).Value != "a" {
				t.Errorf("want the ui.json defaults, got %+v", reply)
			}
			if list := findComponent(t, reply.Body, "list", 0); list.Items == nil || len(*list.Items) != 0 {
				t.Errorf("an empty list must still send items: %+v", list)
			}
		})
	}
}

func TestScreenLocaleAndOtherScreens(t *testing.T) {
	config, _ := testConfig(t, stateRunner(`{"pin":"98"}`, ""))
	config.Locale = "vi_VN"
	writeScreenApp(t, config.AppsDir)
	reply, err := config.Screen("demo", "more", nil)
	if err != nil {
		t.Fatal(err)
	}
	code := findComponent(t, reply.Body, "bigcode", 0)
	if reply.Title != "More" || code.Text != "98" || code.Digits != 4 {
		t.Errorf("more screen: %+v", reply)
	}
	main, _ := config.Screen("demo", "", nil)
	if main.Title != "Thử" {
		t.Errorf("vi_VN must use locales/vi.json, got %q", main.Title)
	}
}

func TestScreenErrors(t *testing.T) {
	config, _ := testConfig(t, stateRunner("{}", ""))
	writeScreenApp(t, config.AppsDir)
	writeStatusApp(t, config.AppsDir, "plain")
	for name, call := range map[string][2]string{
		"no ui.json": {"plain", ""}, "unknown screen": {"demo", "nope"}, "unknown app": {"ghost", ""},
		"host item": {"_refresh", ""},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := config.Screen(call[0], call[1], nil); err == nil {
				t.Error("want an error")
			}
		})
	}
	if err := os.WriteFile(filepath.Join(config.AppsDir, "demo", "ui.json"), []byte(`{"kit":2}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := config.Screen("demo", "", nil); err == nil || !strings.Contains(err.Error(), "invalid_app") {
		t.Errorf("an invalid ui.json must be reported: %v", err)
	}
}

func TestRunEventPassesValueAndOpensDeclaredToast(t *testing.T) {
	var seenScript string
	config, _ := testConfig(t, func(_ context.Context, script string) ScriptOutput {
		if strings.HasSuffix(script, "state.sh") {
			return ScriptOutput{Stdout: `{"name":"Ann"}`}
		}
		seenScript = script
		return ScriptOutput{Stdout: `{"toast":"saved"}` + "\n"}
	})
	writeScreenApp(t, config.AppsDir)
	value := "b"
	result, err := config.RunEvent(Event{AppID: "demo", ActionID: "mode", Value: &value, ScreenID: "main"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(seenScript, "go.sh") || result.Toast != "Saved Ann" || result.Text != "" || !result.OK {
		t.Errorf("unexpected result %+v (script %s)", result, seenScript)
	}
}

func TestRunEventDialogAndPlainText(t *testing.T) {
	output := `{"dialog":"ask"}`
	config, _ := testConfig(t, func(_ context.Context, script string) ScriptOutput { return ScriptOutput{Stdout: output} })
	writeScreenApp(t, config.AppsDir)
	result, err := config.RunEvent(Event{AppID: "demo", ActionID: "go", ScreenID: "main"})
	if err != nil || result.Dialog == nil || result.Dialog.Action != "go" || result.Dialog.OK != "Do it" {
		t.Errorf("dialog: %+v %v", result, err)
	}
	output = `{"toast":"unknown"}`
	if result, _ := config.RunEvent(Event{AppID: "demo", ActionID: "go", ScreenID: "main"}); result.Text != output {
		t.Errorf("an undeclared toast id stays plain text: %+v", result)
	}
	if _, err := config.RunEvent(Event{AppID: "demo", ActionID: "nope"}); err == nil {
		t.Error("only declared actions can run")
	}
}

func TestEventValueReachesTheScriptEnvironment(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "echo.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho \"[$"+ValueEnvName+"]\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	config := Config{ScriptEnv: []string{ValueEnvName + "=on"}}
	if output := config.runScript()(context.Background(), script); output.Stdout != "[on]\n" {
		t.Errorf("got %q", output.Stdout)
	}
	if output := (Config{}).runScript()(context.Background(), script); output.Stdout != "[]\n" {
		t.Errorf("no value must leave the variable unset, got %q", output.Stdout)
	}
}

func TestModelMarksAppsWithScreens(t *testing.T) {
	config, _ := testConfig(t, stateRunner("{}", ""))
	writeScreenApp(t, config.AppsDir)
	writeStatusApp(t, config.AppsDir, "plain")
	model, err := config.BuildModel()
	if err != nil {
		t.Fatal(err)
	}
	screens := map[string]bool{}
	for _, item := range model.Sections[0].Items {
		screens[item.ID] = item.Screen
	}
	if !screens["demo"] || screens["plain"] {
		t.Errorf("screen flags: %v", screens)
	}
}
