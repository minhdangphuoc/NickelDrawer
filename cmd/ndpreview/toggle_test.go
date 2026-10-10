package main

import (
	"testing"

	"koboapps/internal/uikit"
)

func toggleItems(t *testing.T, props, vars map[string]any) (block, []Item) {
	t.Helper()
	rc := renderContext{doc: &uikit.Document{}, locale: "en", vars: vars}
	toggle := rc.toggleBlock(uikit.Component{Kind: "toggle", Props: props})
	return toggle, toggle.draw(0)
}

func TestToggleMatchesNickelSwitchRow(t *testing.T) {
	toggle, items := toggleItems(t, map[string]any{"toggle": "Debug log", "bind": "debug", "enabled": true},
		map[string]any{"debug": true})
	if toggle.height != rowHeight {
		t.Errorf("height = %d, want %d", toggle.height, rowHeight)
	}
	track := firstItem(Page{Items: items}, func(item Item) bool { return item.Shape == "rect" && item.W == toggleSwitchWidth })
	if track.X+track.W != gutter+contentWidth || track.H != toggleSwitchHeight {
		t.Errorf("the switch must end at the content edge, 94 x 56: %+v", track)
	}
	state := firstItem(Page{Items: items}, func(item Item) bool { return len(item.Lines) == 1 && item.Lines[0] == "ON" })
	if state.Bold || state.Size != toggleStateSize || state.X+state.W != track.X-toggleStateGap {
		t.Errorf("the state word must be Nickel's: uppercase, regular, 25 px left of the switch: %+v", state)
	}
}

func TestToggleDetailIsInsideTheRow(t *testing.T) {
	toggle, items := toggleItems(t, map[string]any{"toggle": "postponed", "bind": "shown", "detail": "Shown in the drawer"},
		map[string]any{"shown": false})
	if toggle.height != radioDetailRow {
		t.Fatalf("height = %d, want %d", toggle.height, radioDetailRow)
	}
	detail := firstItem(Page{Items: items}, func(item Item) bool { return len(item.Lines) == 1 && item.Lines[0] == "Shown in the drawer" })
	if detail.Ink != inkMuted || detail.Y+detail.H > toggle.height-separatorPx {
		t.Errorf("the detail line must be muted and above the row divider: %+v", detail)
	}
}
