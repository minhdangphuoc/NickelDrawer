package uikit

import (
	"encoding/json"
	"testing"
)

// flowDocument builds a ui.json whose screens push as given (screen id -> pushed id, "" for none).
func flowDocument(t *testing.T, start string, pushes map[string]string) string {
	t.Helper()
	screens := map[string]any{}
	for id, target := range pushes {
		screen := map[string]any{"title": "Screen"}
		if target != "" {
			screen["action"] = map[string]any{"label": "Open", "push": target}
		}
		screens[id] = screen
	}
	data, err := json.Marshal(map[string]any{"kit": 1, "start": start, "vars": map[string]any{}, "screens": screens})
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestFlowAcceptsValidGraphs(t *testing.T) {
	cases := map[string]map[string]string{
		"main only":            {"main": ""},
		"main and settings":    {"main": "settings", "settings": ""},
		"main, sub and detail": {"main": "settings", "settings": "token", "token": ""},
	}
	for name, pushes := range cases {
		t.Run(name, func(t *testing.T) {
			if report := validate(t, flowDocument(t, "main", pushes)); !report.OK() {
				t.Fatalf("unexpected errors: %v", report.Errors)
			}
		})
	}
}

func TestFlowRulesRejectWithCode(t *testing.T) {
	cases := []struct {
		name   string
		start  string
		pushes map[string]string
		code   string
	}{
		{"no main screen", "home", map[string]string{"home": ""}, CodeFlowNoMain},
		{"start is not main", "settings", map[string]string{"main": "settings", "settings": ""}, CodeFlowStartNotMain},
		{"self push", "main", map[string]string{"main": "main"}, CodeFlowCycle},
		{"cycle of two", "main", map[string]string{"main": "a", "a": "b", "b": "a"}, CodeFlowCycle},
		{"cycle back to main", "main", map[string]string{"main": "a", "a": "main"}, CodeFlowCycle},
		{"three below main", "main", map[string]string{"main": "a", "a": "b", "b": "c", "c": ""}, CodeFlowTooDeep},
		{"orphan screen", "main", map[string]string{"main": "", "lost": ""}, CodeFlowUnreachable},
		{"orphan pushing into the flow", "main", map[string]string{"main": "a", "a": "", "lost": "a"}, CodeFlowUnreachable},
		{"push to unknown screen", "main", map[string]string{"main": "ghost"}, CodeUnknownScreen},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			report := validate(t, flowDocument(t, c.start, c.pushes))
			if !report.HasCode(c.code) {
				t.Fatalf("want %s, got %v", c.code, report.Errors)
			}
		})
	}
}

func TestFlowCycleIsReportedOnce(t *testing.T) {
	report := validate(t, flowDocument(t, "main", map[string]string{"main": "a", "a": "b", "b": "a"}))
	count := 0
	for _, issue := range report.Errors {
		if issue.Code == CodeFlowCycle {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("%d cycle errors, want 1: %v", count, report.Errors)
	}
}

func TestFlowDepthsAreLongestChains(t *testing.T) {
	flow := Flow{Root: "main", Edges: map[string][]string{"main": {"a"}, "a": {"b"}, "b": nil, "lost": nil}}
	depths := flow.Depths()
	if depths["main"] != 0 || depths["a"] != 1 || depths["b"] != 2 {
		t.Fatalf("depths %v", depths)
	}
	if _, found := depths["lost"]; found {
		t.Fatal("an unreachable screen got a depth")
	}
}
