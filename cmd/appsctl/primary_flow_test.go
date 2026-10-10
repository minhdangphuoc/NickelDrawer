package main

import (
	"path/filepath"
	"testing"

	"koboapps/internal/drawer"
)

// installWithState copies a shipped app and replaces its state script by one printing state.
func installWithState(t *testing.T, root drawerRoot, appID, state string) drawer.Config {
	t.Helper()
	copyTree(t, filepath.Join("..", "..", "apps", appID), filepath.Join(root.apps, appID))
	writeExecutable(t, filepath.Join(root.apps, appID, "bin", "state.sh"), []byte("#!/bin/sh\necho '"+state+"'\n"))
	return drawer.Config{StatusTimeout: exampleStateBudget, AppsDir: root.apps, HostDir: root.host, Locale: "en", Bundle: fallbackBundle()}
}

// footerButtons returns the footer buttons of a screen as the drawer resolves them.
func footerButtons(t *testing.T, config drawer.Config, appID, screenID string) []drawer.ScreenComponent {
	t.Helper()
	reply, err := config.Screen(appID, screenID, nil)
	if err != nil {
		t.Fatal(err)
	}
	return reply.Footer
}

// A8: whatever the state, a screen shows exactly one primary and it is the action that is possible.
func TestMainActionFollowsTheStateOnEveryScreenWithAStateDependentPrimary(t *testing.T) {
	cases := []struct {
		name, app, screen, state, wantText, wantAction, wantPush string
	}{
		{"localsend idle", "localsend", "main", `{"running":false}`, "Start receiving", "start", ""},
		{"localsend receiving", "localsend", "main", `{"running":true}`, "Stop", "stop", ""},
		{"ghsync unpaired", "ghsync", "main", `{"configured":false,"can_sync":false}`, "Pair with your phone", "", "pair"},
		{"ghsync paired", "ghsync", "main", `{"configured":true,"can_sync":true}`, "Sync now", "sync", ""},
		{"ghsync pairing idle", "ghsync", "pair", `{"pairing":false}`, "Start pairing", "pair", ""},
		{"ghsync pairing open", "ghsync", "pair", `{"pairing":true}`, "Stop pairing", "stop-pairing", ""},
		{"library idle", "library", "main", `{"running":false}`, "Scan", "scan", ""},
		{"library scanning", "library", "main", `{"running":true,"can_stop":true}`, "Stop", "cancel", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			config := installWithState(t, newDrawerRoot(t), tc.app, tc.state)
			footer := footerButtons(t, config, tc.app, tc.screen)
			if len(footer) != 1 {
				t.Fatalf("want exactly one footer button, got %+v", footer)
			}
			button := footer[0]
			if button.Style != "primary" || button.Text != tc.wantText || button.Action != tc.wantAction || button.Push != tc.wantPush {
				t.Errorf("button = %+v, want %q action %q push %q", button, tc.wantText, tc.wantAction, tc.wantPush)
			}
		})
	}
}

// Library's actions are rows that appear only when they apply; the destructive ones ask first.
func TestLibraryActionsAreRowsShownOnlyWhenTheyApply(t *testing.T) {
	config := installWithState(t, newDrawerRoot(t), "library", `{"can_fix":true,"can_undo":false,"has_selection":true,"can_restore":false}`)
	reply, err := config.Screen("library", "main", nil)
	if err != nil {
		t.Fatal(err)
	}
	rows := map[string]drawer.ScreenComponent{}
	for _, component := range reply.Body {
		if component.Type == "row" {
			rows[component.Text] = component
		}
	}
	if len(rows) != 2 {
		t.Fatalf("only Fix language and Move to trash apply, got %v", rows)
	}
	for _, name := range []string{"Fix language", "Move to trash"} {
		row, found := rows[name]
		if !found || row.Action == "" || row.Confirm == nil || row.Confirm.OK == "" {
			t.Errorf("%s must be a row with an action and a confirm: %+v", name, row)
		}
	}
}
