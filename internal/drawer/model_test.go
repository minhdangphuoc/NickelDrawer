package drawer

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"koboapps/internal/manifest"
)

func writeFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestAppIconIsThePngNextToTheManifest(t *testing.T) {
	config, _ := testConfig(t, nil)
	writeStatusApp(t, config.AppsDir, "with-icon")
	writeStatusApp(t, config.AppsDir, "no-icon")
	writeStatusApp(t, config.AppsDir, "svg-only")
	writeFile(t, filepath.Join(config.AppsDir, "with-icon", "icon.png"))
	writeFile(t, filepath.Join(config.AppsDir, "svg-only", "icon.svg"))
	model, err := config.BuildModel()
	if err != nil {
		t.Fatal(err)
	}
	icons := map[string]string{}
	for _, item := range model.Sections[0].Items {
		icons[item.ID] = item.Icon
	}
	want := filepath.Join(config.AppsDir, "with-icon", "icon.png")
	if icons["with-icon"] != want || icons["no-icon"] != "" || icons["svg-only"] != "" {
		t.Fatalf("icons = %v", icons)
	}
}

func TestHostItemsGetTheirIconsWhenTheFilesExist(t *testing.T) {
	config, _ := testConfig(t, nil)
	writeFileIn(t, config.HostDir, "data/update.json", `{"repo":"o/r"}`)
	writeFile(t, filepath.Join(config.HostDir, "icons", "refresh.png"))
	model, err := config.BuildModel()
	if err != nil {
		t.Fatal(err)
	}
	items := model.Sections[1].Items
	if len(items) != 2 || items[0].Icon != filepath.Join(config.HostDir, "icons", "refresh.png") || items[1].Icon != "" {
		t.Fatalf("host items = %+v", items)
	}
	writeFile(t, filepath.Join(config.HostDir, "icons", "updates.png"))
	model, _ = config.BuildModel()
	if got := model.Sections[1].Items[1].Icon; got != filepath.Join(config.HostDir, "icons", "updates.png") {
		t.Fatalf("updates icon = %q", got)
	}
}

func writeHiddenApp(t *testing.T, appsDir string) {
	t.Helper()
	writeStatusApp(t, appsDir, "postponed")
	manifestPath := filepath.Join(appsDir, "postponed", "app.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	hidden := strings.Replace(string(data), `"version":"1.0.0",`, `"version":"1.0.0","hidden":true,`, 1)
	if err := os.WriteFile(manifestPath, []byte(hidden), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestModelOmitsHiddenAppsAndNeverRunsTheirStatus(t *testing.T) {
	var statusRuns atomic.Int32
	runner := func(ctx context.Context, script string) ScriptOutput {
		statusRuns.Add(1)
		return ScriptOutput{Stdout: "ok\n"}
	}
	config, _ := testConfig(t, runner)
	writeStatusApp(t, config.AppsDir, "shown")
	writeHiddenApp(t, config.AppsDir)
	model, err := config.BuildModel()
	if err != nil {
		t.Fatal(err)
	}
	items := model.Sections[0].Items
	if len(items) != 1 || items[0].ID != "shown" || statusRuns.Load() != 1 {
		t.Fatalf("items=%+v statusRuns=%d", items, statusRuns.Load())
	}
}

func TestHiddenAppStillRunsById(t *testing.T) {
	var runs atomic.Int32
	config, _ := testConfig(t, func(ctx context.Context, script string) ScriptOutput {
		runs.Add(1)
		return ScriptOutput{}
	})
	writeHiddenApp(t, config.AppsDir)
	result, err := config.Run("postponed", "status")
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK || runs.Load() != 1 {
		t.Fatalf("result = %+v, runs = %d", result, runs.Load())
	}
}

func TestModelActionsCarryMenuOptInAndConfirm(t *testing.T) {
	yes, no := true, false
	actions := modelActions([]manifest.Action{
		{ID: "go", Label: "Go", Output: manifest.OutputShow},
		{ID: "wipe", Label: "Wipe", Output: manifest.OutputShow, Menu: &yes, Confirm: "Wipe everything?"},
		{ID: "quiet", Label: "Quiet", Output: manifest.OutputShow, Menu: &no},
	})
	if actions[0].Menu || !actions[1].Menu || actions[2].Menu {
		t.Errorf("menu flags = %v %v %v; want false true false (screen-only actions stay out)",
			actions[0].Menu, actions[1].Menu, actions[2].Menu)
	}
	if actions[1].Confirm != "Wipe everything?" || actions[0].Confirm != "" {
		t.Errorf("confirm = %q / %q", actions[1].Confirm, actions[0].Confirm)
	}
	encoded, _ := json.Marshal(actions[1])
	if !strings.Contains(string(encoded), `"confirm":"Wipe everything?"`) || !strings.Contains(string(encoded), `"menu":true`) {
		t.Errorf("json = %s", encoded)
	}
}
