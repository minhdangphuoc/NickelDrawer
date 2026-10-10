package uikit

import (
	"strings"
	"testing"
)

const galleryMain = `{"gallery":"$items","columns":4,"rows":3,"on_tap":{"push":"viewer"},
"on_long_press":[{"label":"Set","action":"start"},{"label":"View","push":"viewer"},{"label":"Info","info":true},
{"label":"Remove","action":"stop","confirm":{"text":"Sure?","ok":"Remove"}}],
"pending":"$status","empty":{"icon":"picture","title":"None","text":"Copy some","button":"Import","action":"start"}}`

const viewerMain = `{"viewer":"$items","selected":"status","current_status":"In use",
"primary":{"label":"Set","label_current":"In use","action":"start"},
"secondary":{"label":"Remove","action":"stop","confirm":{"text":"Sure?","ok":"Remove"}}}`

// galleryDocument has main (gallery) and viewer (body as given) screens plus optional extras on main.
func galleryDocument(mainExtra, viewerBody, viewerExtra string) string {
	return `{"kit":1,"start":"main","vars":{"items":[],"status":""},"screens":{
"main":{"title":"Main"` + mainExtra + `,"body":[` + galleryMain + `]},
"viewer":{"title":"Viewer"` + viewerExtra + `,"body":[` + viewerBody + `]}}}`
}

func TestGalleryAndViewerValidate(t *testing.T) {
	cases := map[string]string{
		"gallery and viewer":      galleryDocument("", viewerMain, ""),
		"header icon actions":     galleryDocument(`,"actions":[{"icon":"import","label":"Import","action":"start"},{"icon":"settings","label":"Options","push":"viewer"}]`, viewerMain, ""),
		"radio with sub-line":     screenJSON(`[{"radio":"L","bind":"pin","options":[["a","A","About A"],["b","B"]]}]`, ""),
		"minimal gallery":         `{"kit":1,"start":"main","vars":{"items":[]},"screens":{"main":{"title":"M","body":[{"gallery":"$items"}]}}}`,
		"gallery empty no button": `{"kit":1,"start":"main","vars":{"items":[]},"screens":{"main":{"title":"M","body":[{"gallery":"$items","empty":{"title":"None"}}]}}}`,
	}
	for name, document := range cases {
		t.Run(name, func(t *testing.T) {
			if report := validate(t, document); !report.OK() {
				t.Fatalf("unexpected errors: %v", report.Errors)
			}
		})
	}
}

func TestGalleryAndViewerRulesRejectWithCode(t *testing.T) {
	icon := `{"icon":"import","label":"Import","action":"start"}`
	cases := []struct {
		name     string
		document string
		code     string
	}{
		{"actions with action link", galleryDocument(`,"action":{"label":"Go","push":"viewer"},"actions":[`+icon+`]`, viewerMain, ""), CodeNeedsOne},
		{"icon action with both", galleryDocument(`,"actions":[{"icon":"import","label":"I","action":"start","push":"viewer"}]`, viewerMain, ""), CodeNeedsOne},
		{"icon action with neither", galleryDocument(`,"actions":[{"icon":"import","label":"I"}]`, viewerMain, ""), CodeNeedsOne},
		{"unknown icon", galleryDocument(`,"actions":[{"icon":"star","label":"I","action":"start"}]`, viewerMain, ""), CodeBadEnum},
		{"three icon actions", galleryDocument(`,"actions":[`+icon+`,`+icon+`,`+icon+`]`, viewerMain, ""), CodeOutOfRange},
		{"gallery not an array", strings.Replace(galleryDocument("", viewerMain, ""), `"items":[]`, `"items":""`, 1), CodeListNotArray},
		{"columns out of range", screenJSON(`[{"gallery":"$items","columns":7}]`, ""), CodeOutOfRange},
		{"tap with both", screenJSON(`[{"gallery":"$items","on_tap":{"push":"main","action":"start"}}]`, ""), CodeNeedsOne},
		{"tap with neither", screenJSON(`[{"gallery":"$items","on_tap":{}}]`, ""), CodeNeedsOne},
		{"menu item with two targets", screenJSON(`[{"gallery":"$items","on_long_press":[{"label":"A","action":"start","info":true}]}]`, ""), CodeNeedsOne},
		{"menu item with none", screenJSON(`[{"gallery":"$items","on_long_press":[{"label":"A"}]}]`, ""), CodeNeedsOne},
		{"seven menu items", screenJSON(`[{"gallery":"$items","on_long_press":[`+strings.TrimSuffix(strings.Repeat(`{"label":"A","info":true},`, 7), ",")+`]}]`, ""), CodeOutOfRange},
		{"empty button without action", screenJSON(`[{"gallery":"$items","empty":{"title":"None","button":"Go"}}]`, ""), CodeNeedsTogether},
		{"empty without title", screenJSON(`[{"gallery":"$items","empty":{"text":"x"}}]`, ""), CodeMissingProp},
		{"two galleries", screenJSON(`[{"gallery":"$items"},{"gallery":"$items"}]`, ""), CodeDuplicate},
		{"gallery empty button plus primary", screenJSON(`[{"gallery":"$items","empty":{"title":"None","button":"Go","action":"start"}}]`, `[{"button":"B","action":"stop","style":"primary"}]`), CodeMultiplePrimary},
		{"viewer undeclared selected", galleryDocument("", strings.Replace(viewerMain, `"selected":"status"`, `"selected":"ghost"`, 1), ""), CodeUnknownVar},
		{"viewer without primary", galleryDocument("", `{"viewer":"$items","selected":"status"}`, ""), CodeMissingProp},
		{"viewer with footer", strings.Replace(galleryDocument("", viewerMain, `,"footer":[{"button":"X","action":"start"}]`), "", "", 0), CodeViewerNotAlone},
		{"viewer beside text", galleryDocument("", viewerMain+`,{"text":"hi"}`, ""), CodeViewerNotAlone},
		{"viewer plus primary button", galleryDocument("", viewerMain+`,{"button":"B","action":"stop","style":"primary"}`, ""), CodeMultiplePrimary},
		{"two viewers", galleryDocument("", viewerMain+`,`+viewerMain, ""), CodeDuplicate},
		{"radio with four entries", screenJSON(`[{"radio":"L","bind":"pin","options":[["a","A","x","y"]]}]`, ""), CodeWrongType},
		{"radio sub-line too long", screenJSON(`[{"radio":"L","bind":"pin","options":[["a","A","`+strings.Repeat("x", 61)+`"]]}]`, ""), CodeTextTooLong},
		{"radio with one entry", screenJSON(`[{"radio":"L","bind":"pin","options":[["a"]]}]`, ""), CodeWrongType},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if report := validate(t, tc.document); !report.HasCode(tc.code) {
				t.Fatalf("want error %s, got %v", tc.code, report.Errors)
			}
		})
	}
}

func TestFlowFollowsGalleryAndHeaderIconPushes(t *testing.T) {
	screens := map[string]any{
		"main": map[string]any{
			"actions": []any{map[string]any{"icon": "settings", "push": "options"}},
			"body": []any{map[string]any{
				"gallery":       "$items",
				"on_tap":        map[string]any{"push": "viewer"},
				"on_long_press": []any{map[string]any{"label": "View", "push": "detail"}, map[string]any{"label": "Info", "info": true}},
			}},
		},
		"options": map[string]any{}, "viewer": map[string]any{}, "detail": map[string]any{},
	}
	got := flowOf(screens).Edges["main"]
	want := []string{"options", "viewer", "detail"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("edges of main = %v, want %v", got, want)
	}
}

func TestFlowCyclesWithSeveralEdgesPerScreen(t *testing.T) {
	flow := Flow{Root: "main", Edges: map[string][]string{
		"main": {"a", "b"}, "a": {"b", "main"}, "b": {"a"},
	}}
	cycles := flow.Cycles()
	keys := map[string]bool{}
	for _, cycle := range cycles {
		keys[strings.Join(cycle, ">")] = true
	}
	for _, want := range []string{"a>b", "a>main", "a>main>b"} {
		if !keys[want] {
			t.Errorf("cycle %s not found in %v", want, cycles)
		}
	}
	if len(cycles) != 3 {
		t.Errorf("%d cycles, want 3: %v", len(cycles), cycles)
	}
}

func TestFlowRejectsCycleThroughGalleryPush(t *testing.T) {
	document := galleryDocument("", viewerMain, "")
	document = strings.Replace(document, `"title":"Viewer"`, `"title":"Viewer","action":{"label":"Back","push":"main"}`, 1)
	if report := validate(t, document); !report.HasCode(CodeFlowCycle) {
		t.Fatalf("want a flow cycle, got %v", report.Errors)
	}
}
