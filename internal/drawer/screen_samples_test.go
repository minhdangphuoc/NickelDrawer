package drawer

import (
	"os"
	"path/filepath"
	"testing"
)

const batch3UI = `{"kit":1,"start":"main",
 "vars":{"running":false,"configured":true},
 "screens":{
  "main":{"title":"Demo",
   "body":[
    {"samples":[
      {"button":"Sync","style":"primary","look":"pressed","caption":"Pressed"},
      {"button":"Sync","style":"primary","look":"disabled","caption":"Disabled"}],"columns":2},
    {"samples":[
      {"toggle":"Wi-Fi","look":"selected","caption":"On"},
      {"toggle":"Wi-Fi","look":"disabled"},
      {"radio":"Mode","options":[["a","Fast"]],"look":"selected"},
      {"radio":"Mode","options":[["a","Fast"]]},
      {"field":"Search","value":"Truyện Kiều"},
      {"row":"Refresh","detail":"Balanced","action":"go"},
      {"kv":[["IP","10.0.0.2"]]}]},
    {"row":"Fix language","action":"go","confirm":{"text":"Sure?","ok":"Fix"},"show":"$configured"},
    {"row":"Stop","action":"go","show":"$running"}],
   "footer":[
    {"button":"Pair with phone","push":"pair","style":"primary","show":"!$configured"},
    {"button":"Sync now","action":"go","style":"primary","show":"$configured"}]},
  "pair":{"title":"Pair"}}}`

func writeBatch3App(t *testing.T, appsDir string) {
	t.Helper()
	dir := filepath.Join(appsDir, "demo")
	for path, content := range map[string]string{
		"app.json": screenManifest, "ui.json": batch3UI, "bin/go.sh": "#!/bin/sh\n", "bin/state.sh": "#!/bin/sh\n",
		"locales/en.json": `{}`,
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

func TestScreenResolvesSamplesAsStaticCells(t *testing.T) {
	config, _ := testConfig(t, stateRunner(`{}`, ""))
	writeBatch3App(t, config.AppsDir)
	reply, err := config.Screen("demo", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	buttons := findComponent(t, reply.Body, "samples", 0)
	if buttons.Columns != 2 || len(buttons.Cells) != 2 {
		t.Fatalf("buttons: columns=%d cells=%d", buttons.Columns, len(buttons.Cells))
	}
	pressed, disabled := buttons.Cells[0], buttons.Cells[1]
	if pressed.Type != "button" || pressed.Look != "pressed" || pressed.Caption != "Pressed" || pressed.Style != "primary" || !*pressed.Enabled {
		t.Errorf("pressed cell = %+v", pressed)
	}
	if disabled.Look != "disabled" || *disabled.Enabled {
		t.Errorf("disabled cell = %+v", disabled)
	}

	cells := findComponent(t, reply.Body, "samples", 1)
	if len(cells.Cells) != 7 || cells.Columns != 1 {
		t.Fatalf("cells=%d columns=%d", len(cells.Cells), cells.Columns)
	}
	on, off := cells.Cells[0], cells.Cells[1]
	if on.Value != true || !*on.Enabled || off.Value != false || *off.Enabled || off.Look != "disabled" {
		t.Errorf("toggle cells: on=%+v off=%+v", on, off)
	}
	chosen, plain := cells.Cells[2], cells.Cells[3]
	if chosen.Value != "a" || plain.Value != "" || len(chosen.Options) != 1 {
		t.Errorf("radio cells: chosen=%+v plain=%+v", chosen, plain)
	}
	if field := cells.Cells[4]; field.Type != "field" || field.Value != "Truyện Kiều" {
		t.Errorf("field cell = %+v", field)
	}
	if row := cells.Cells[5]; row.Type != "row" || row.Detail != "Balanced" || row.Look != "normal" {
		t.Errorf("row cell = %+v", row)
	}
	if kv := cells.Cells[6]; kv.Type != "kv" || kv.Rows == nil {
		t.Errorf("kv cell = %+v", kv)
	}
}

func TestScreenLeavesOutRowsAndButtonsWhoseShowFlagIsFalse(t *testing.T) {
	config, _ := testConfig(t, stateRunner(`{"configured":false,"running":true}`, ""))
	writeBatch3App(t, config.AppsDir)
	reply, err := config.Screen("demo", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	var rows []string
	for _, component := range reply.Body {
		if component.Type == "row" {
			rows = append(rows, component.Text)
		}
	}
	if len(rows) != 1 || rows[0] != "Stop" {
		t.Errorf("visible rows = %v, want only Stop", rows)
	}
	if len(reply.Footer) != 1 || reply.Footer[0].Text != "Pair with phone" || reply.Footer[0].Push != "pair" || reply.Footer[0].Action != "" {
		t.Errorf("footer = %+v, want the pushing primary only", reply.Footer)
	}
}

func TestScreenRowCarriesItsConfirm(t *testing.T) {
	config, _ := testConfig(t, stateRunner(`{}`, ""))
	writeBatch3App(t, config.AppsDir)
	reply, err := config.Screen("demo", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	row := findComponent(t, reply.Body, "row", 0)
	if row.Text != "Fix language" || row.Confirm == nil || row.Confirm.Text != "Sure?" || row.Confirm.OK != "Fix" {
		t.Errorf("row = %+v", row)
	}
	if len(reply.Footer) != 1 || reply.Footer[0].Text != "Sync now" || reply.Footer[0].Action != "go" {
		t.Errorf("footer = %+v, want the sync primary", reply.Footer)
	}
}
