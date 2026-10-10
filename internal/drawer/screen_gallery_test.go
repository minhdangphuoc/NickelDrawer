package drawer

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const galleryManifest = `{"schema":1,"id":"pics","name":"Pics","version":"1.0.0","actions":[
 {"id":"import","label":"Import","run":"bin/run.sh","output":"show","menu":false},
 {"id":"set-current","label":"Set","run":"bin/run.sh","output":"show","menu":false},
 {"id":"remove","label":"Remove","run":"bin/run.sh","output":"show","menu":false},
 {"id":"state","label":"State","run":"bin/state.sh","output":"show","menu":false}]}`

const galleryUI = `{"kit":1,"start":"main",
 "vars":{"pictures":[],"pending":0,"selected":"","summary":"","mode":"fill"},
 "screens":{
  "main":{"title":"Pics","actions":[{"icon":"import","label":"Import","action":"import"},{"icon":"settings","label":"Options","push":"options"}],
   "body":[{"text":"$summary"},
    {"gallery":"$pictures","on_tap":{"push":"viewer"},
     "on_long_press":[{"label":"Set","action":"set-current"},{"label":"Info","info":true},
       {"label":"Remove","action":"remove","confirm":{"text":"Sure?","ok":"Remove","cancel":"No"}}],
     "pending":"$pending",
     "empty":{"icon":"picture","title":"None","text":"Copy","button":"Import","action":"import"}}]},
  "viewer":{"title":"Viewer","body":[{"viewer":"$pictures","selected":"selected","current_status":"In use",
     "primary":{"label":"Set","label_current":"Now","action":"set-current"},
     "secondary":{"label":"Remove","action":"remove","confirm":{"text":"Sure?","ok":"Remove"}}}]},
  "options":{"title":"Options","body":[{"radio":"Mode","bind":"mode","options":[["fill","Fill","Crop edges"],["fit","Fit"]]}]}}}`

func writeGalleryApp(t *testing.T, appsDir string) string {
	t.Helper()
	dir := filepath.Join(appsDir, "pics")
	for path, content := range map[string]string{
		"app.json": galleryManifest, "ui.json": galleryUI, "bin/run.sh": "#!/bin/sh\n", "bin/state.sh": "#!/bin/sh\n",
	} {
		full := filepath.Join(dir, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func pictureState(t *testing.T, dir string, pictures []map[string]any) string {
	t.Helper()
	data, err := json.Marshal(map[string]any{"pictures": pictures, "pending": 2, "summary": ""})
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestScreenResolvesGalleryTiles(t *testing.T) {
	config, _ := testConfig(t, nil)
	dir := writeGalleryApp(t, config.AppsDir)
	data := filepath.Join(dir, "data")
	config.RunScript = stateRunner(pictureState(t, dir, []map[string]any{
		{"id": "0001", "thumb": filepath.Join(data, "thumbs", "0001.png"), "label": "one", "info": "a\nb", "badge": "current"},
		{"id": "0002", "thumb": filepath.Join(data, "thumbs", "0002.png"), "label": "two"},
		{"id": "bad id!", "thumb": filepath.Join(data, "thumbs", "x.png"), "label": "dropped"},
	}), "")
	reply, err := config.Screen("pics", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(reply)
	text := string(raw)
	for _, want := range []string{
		`"actions":[{"icon":"import","label":"Import","action":"import"},{"icon":"settings","label":"Options","push":"options"}]`,
		`"type":"gallery"`, `"columns":4`, `"rows":3`, `"tap":{"push":"viewer"}`, `"pending":2`,
		`"menu":[{"label":"Set","action":"set-current"},{"label":"Info","info":true},{"label":"Remove","action":"remove","confirm":{"title":"","text":"Sure?","ok":"Remove","cancel":"No"}}]`,
		`"empty":{"icon":"picture","title":"None","text":"Copy","button":"Import","action":"import"}`,
		`"current":true`,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("reply lacks %s\n%s", want, text)
		}
	}
	gallery := findComponent(t, reply.Body, "gallery", 0)
	tiles := gallery.Tiles.([]GalleryTile)
	if len(tiles) != 2 || tiles[0].ID != "0001" || !tiles[0].Current || tiles[1].Current {
		t.Errorf("tiles: %+v", tiles)
	}
	if len(reply.Body) != 1 {
		t.Errorf("a text bound to an empty variable must be hidden: %+v", reply.Body)
	}
}

func TestScreenRejectsUnsafePicturePaths(t *testing.T) {
	config, _ := testConfig(t, nil)
	dir := writeGalleryApp(t, config.AppsDir)
	data := filepath.Join(dir, "data")
	unsafe := map[string]string{
		"parent dir":    data + "/thumbs/../../secret.png",
		"outside data":  filepath.Join(dir, "other", "0001.png"),
		"not png":       filepath.Join(data, "thumbs", "0001.jpg"),
		"relative":      "data/thumbs/0001.png",
		"sibling name":  data + "-evil/0001.png",
		"empty":         "",
		"data dir only": data + ".png",
	}
	for name, path := range unsafe {
		t.Run(name, func(t *testing.T) {
			config.RunScript = stateRunner(pictureState(t, dir, []map[string]any{{"id": "0001", "thumb": path, "label": "x"}}), "")
			reply, err := config.Screen("pics", "", nil)
			if err != nil {
				t.Fatal(err)
			}
			tiles := findComponent(t, reply.Body, "gallery", 0).Tiles.([]GalleryTile)
			if len(tiles) != 1 || tiles[0].Thumb != "" {
				t.Errorf("unsafe path %q was sent: %+v", path, tiles)
			}
		})
	}
}

func TestScreenCapsTilesAtFiveHundred(t *testing.T) {
	config, _ := testConfig(t, nil)
	dir := writeGalleryApp(t, config.AppsDir)
	var pictures []map[string]any
	for i := 0; i < 600; i++ {
		pictures = append(pictures, map[string]any{"id": "p" + strings.Repeat("0", 3) + string(rune('a'+i%26)) + strings.Repeat("x", i/26), "label": "x"})
	}
	config.RunScript = stateRunner(pictureState(t, dir, pictures), "")
	reply, _ := config.Screen("pics", "", nil)
	if got := len(findComponent(t, reply.Body, "gallery", 0).Tiles.([]GalleryTile)); got != maxGalleryTiles {
		t.Errorf("%d tiles, want %d", got, maxGalleryTiles)
	}
}

func TestScreenValueSelectsViewerPictureAndReachesState(t *testing.T) {
	config, _ := testConfig(t, nil)
	dir := writeGalleryApp(t, config.AppsDir)
	images := filepath.Join(dir, "data", "images")
	state := pictureState(t, dir, []map[string]any{
		{"id": "0001", "image": filepath.Join(images, "0001.png"), "label": "one"},
		{"id": "0002", "image": filepath.Join(images, "0002.png"), "label": "two", "badge": "current"},
	})
	config.RunScript = stateRunner(state, "")
	for name, test := range map[string]struct {
		value *string
		want  string
	}{"names a tile": {ptr("0002"), "0002"}, "unknown id": {ptr("9999"), "0001"}, "no value": {nil, "0001"}} {
		t.Run(name, func(t *testing.T) {
			reply, err := config.Screen("pics", "viewer", test.value)
			if err != nil {
				t.Fatal(err)
			}
			viewer := findComponent(t, reply.Body, "viewer", 0)
			if viewer.Value != test.want || viewer.CurrentStatus != "In use" || viewer.Primary.LabelCurrent != "Now" ||
				viewer.Secondary.Confirm == nil || viewer.Secondary.Confirm.OK != "Remove" {
				t.Errorf("viewer: %+v", viewer)
			}
			if (test.value == nil) != (reply.Value == nil) {
				t.Errorf("value echo: %v", reply.Value)
			}
		})
	}
	reply, _ := config.Screen("pics", "viewer", ptr("0002"))
	if raw, _ := json.Marshal(reply); !strings.Contains(string(raw), `"value":"0002"`) {
		t.Errorf("the reply must echo the value: %s", raw)
	}
}

func TestScreenValueSetsSelectedAndEnv(t *testing.T) {
	dir := t.TempDir()
	config := Config{AppsDir: dir, HostDir: t.TempDir(), CacheDir: t.TempDir(), Locale: "en"}
	appDir := writeGalleryApp(t, dir)
	script := `#!/bin/sh
echo "{\"summary\":\"v=$NICKELDRAWER_VALUE d=$NICKELDRAWER\"}"
`
	if err := os.WriteFile(filepath.Join(appDir, "bin", "state.sh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	config.StatusTimeout = 5e9
	reply, err := config.Screen("pics", "main", ptr("0003"))
	if err != nil {
		t.Fatal(err)
	}
	if reply.Stale || reply.Value == nil || *reply.Value != "0003" {
		t.Fatalf("reply: %+v", reply)
	}
	if text := findComponent(t, reply.Body, "text", 0).Text; text != "v=0003 d=1" {
		t.Errorf("state script saw %q", text)
	}
}

func TestScreenMarksFailedStateAsStale(t *testing.T) {
	cases := map[string]ScriptRunner{
		"bad json":  stateRunner("not json", ""),
		"exit code": func(context.Context, string) ScriptOutput { return ScriptOutput{ExitCode: 1, Stdout: `{}`} },
		"timeout":   func(context.Context, string) ScriptOutput { return ScriptOutput{TimedOut: true} },
		"not found": func(context.Context, string) ScriptOutput { return ScriptOutput{Err: os.ErrNotExist} },
	}
	for name, runner := range cases {
		t.Run(name, func(t *testing.T) {
			config, _ := testConfig(t, runner)
			writeGalleryApp(t, config.AppsDir)
			reply, err := config.Screen("pics", "", nil)
			if err != nil || !reply.Stale || reply.StaleReason == "" {
				t.Errorf("want stale with a reason, got %+v %v", reply, err)
			}
		})
	}
	config, _ := testConfig(t, stateRunner(`{"summary":"ok"}`, ""))
	writeGalleryApp(t, config.AppsDir)
	if reply, _ := config.Screen("pics", "", nil); reply.Stale || reply.StaleReason != "" {
		t.Error("a good state is not stale")
	}
	writeScreenApp(t, config.AppsDir)
	failing := func(context.Context, string) ScriptOutput { return ScriptOutput{ExitCode: 1} }
	config.RunScript = failing
	if reply, _ := config.Screen("demo", "", nil); !reply.Stale {
		t.Error("demo has a state action that failed")
	}
}

func TestStaleReasonNamesTimeoutExitAndBadOutput(t *testing.T) {
	cases := []struct {
		runner ScriptRunner
		want   string
	}{
		{func(context.Context, string) ScriptOutput { return ScriptOutput{TimedOut: true, ExitCode: -1} }, "took longer than"},
		{func(context.Context, string) ScriptOutput { return ScriptOutput{ExitCode: 3, Stderr: "boom\nmore"} }, "exited with 3: boom"},
		{stateRunner("not json", ""), "did not print one JSON object"},
	}
	for _, tc := range cases {
		config, _ := testConfig(t, tc.runner)
		writeGalleryApp(t, config.AppsDir)
		reply, err := config.Screen("pics", "", nil)
		if err != nil || !strings.Contains(reply.StaleReason, tc.want) {
			t.Errorf("stale reason %q, want it to contain %q (%v)", reply.StaleReason, tc.want, err)
		}
	}
}

func TestScreenRadioOptionDetails(t *testing.T) {
	config, _ := testConfig(t, stateRunner(`{"mode":"fit"}`, ""))
	writeGalleryApp(t, config.AppsDir)
	reply, err := config.Screen("pics", "options", nil)
	if err != nil {
		t.Fatal(err)
	}
	radio := findComponent(t, reply.Body, "radio", 0)
	if len(radio.Options) != 2 || radio.Options[0] != [2]string{"fill", "Fill"} || strings.Join(radio.OptionDetails, "|") != "Crop edges|" {
		t.Errorf("radio: %+v", radio)
	}
	// Without any sub-line the field is left out.
	config.RunScript = stateRunner(`{}`, "")
	writeScreenApp(t, config.AppsDir)
	plain, _ := config.Screen("demo", "", nil)
	if raw, _ := json.Marshal(plain); strings.Contains(string(raw), "option_details") {
		t.Errorf("option_details must be omitted: %s", raw)
	}
}

func TestEveryScriptSeesNickelDrawerEnv(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "env.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho \"[$NICKELDRAWER]\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if output := (Config{}).runScript()(context.Background(), script); output.Stdout != "[1]\n" {
		t.Errorf("got %q", output.Stdout)
	}
}

func ptr(text string) *string { return &text }
