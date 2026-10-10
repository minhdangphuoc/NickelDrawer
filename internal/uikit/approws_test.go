package uikit

import "testing"

func markerRow() Component {
	return Component{Kind: "row", Props: map[string]any{"row": "x", "action": AppRowsMarkerAction}}
}

func TestExpandAppRowsReplacesTheMarkerAndGroupsInstalledVersions(t *testing.T) {
	vars := map[string]any{AppRowsVar: []any{
		map[string]any{"kind": "section", "text": "Updates"},
		map[string]any{"kind": "row", "text": "To-do", "detail": "1 → 2", "action": "app_update_todo",
			"confirm": map[string]any{"title": "t", "text": "x", "ok": "OK", "cancel": "No"}},
		map[string]any{"kind": "section", "text": "Installed"},
		map[string]any{"kind": "kv", "key": "A", "value": "1"},
		map[string]any{"kind": "kv", "key": "B", "value": "2"},
	}}
	tail := Component{Kind: "text", Props: map[string]any{"text": "after"}}
	got := ExpandAppRows([]Component{markerRow(), tail}, vars)
	kinds := ""
	for _, component := range got {
		kinds += component.Kind + " "
	}
	if kinds != "section row section kv text " {
		t.Fatalf("components = %s", kinds)
	}
	if pairs := got[3].Props["kv"].([]any); len(pairs) != 2 {
		t.Errorf("installed versions are one kv of %d pairs", len(pairs))
	}
	if got[1].Props["action"] != "app_update_todo" || got[1].Props["confirm"].(map[string]any)["ok"] != "OK" {
		t.Errorf("row = %+v", got[1].Props)
	}
}

func TestExpandAppRowsEscapesDollarSignsAndSplitsLongKvRuns(t *testing.T) {
	items := []any{map[string]any{"kind": "section", "text": "Cost $5"}}
	for i := 0; i < 13; i++ {
		items = append(items, map[string]any{"kind": "kv", "key": "k", "value": "v"})
	}
	got := ExpandAppRows([]Component{markerRow()}, map[string]any{AppRowsVar: items})
	if got[0].Props["section"] != "Cost $$5" {
		t.Errorf("section = %v", got[0].Props["section"])
	}
	if len(got) != 3 || len(got[1].Props["kv"].([]any)) != 12 || len(got[2].Props["kv"].([]any)) != 1 {
		t.Errorf("kv runs = %d components", len(got))
	}
}

func TestExpandAppRowsWithoutItemsRemovesTheMarker(t *testing.T) {
	if got := ExpandAppRows([]Component{markerRow()}, map[string]any{}); len(got) != 0 {
		t.Errorf("got %d components", len(got))
	}
	other := Component{Kind: "row", Props: map[string]any{"row": "x", "action": "other"}}
	if got := ExpandAppRows([]Component{other}, nil); len(got) != 1 {
		t.Error("an ordinary row was removed")
	}
}
