package uikit

import (
	"strings"
	"testing"

	"koboapps/internal/i18n"
)

func testEnv() Env {
	return Env{
		ActionIDs: []string{"start", "stop"},
		Catalogs: map[string]i18n.Catalog{
			"en": {"ui.go": "Go", "ui.stop": "Stop"},
			"vi": {"ui.go": "Đi"},
		},
	}
}

// screenJSON wraps body components into a valid document; the footer is optional.
func screenJSON(body, footer string) string {
	if footer == "" {
		footer = "[]"
	}
	return `{"kit":1,"start":"main","vars":{"pin":"","items":[],"on":false,"status":""},
	"screens":{"main":{"title":"Main","body":` + body + `,"footer":` + footer + `}}}`
}

func validate(t *testing.T, document string) Report {
	t.Helper()
	_, report := ParseAndValidate([]byte(document), nil, testEnv())
	return report
}

func TestValidDocumentsPass(t *testing.T) {
	cases := map[string]string{
		"empty screen":               screenJSON(`[]`, ""),
		"text with style":            screenJSON(`[{"text":"Hello","style":"caption","align":"center"}]`, ""),
		"locale map":                 screenJSON(`[{"text":{"en":"Hello","vi":"Xin chào"}}]`, ""),
		"row pushing a screen":       `{"kit":1,"start":"main","screens":{"main":{"title":"M","body":[{"row":"Options","push":"more"}]},"more":{"title":"More"}}}`,
		"locale key":                 screenJSON(`[{"notice":"@ui.go","title":"@ui.go"}]`, ""),
		"template":                   screenJSON(`[{"text":"PIN $pin, cost $$5"}]`, ""),
		"kv and bigcode":             screenJSON(`[{"kv":[["IP","$status"]]},{"bigcode":"$pin","digits":4}]`, ""),
		"list":                       screenJSON(`[{"list":"$items","max":10,"tap":"stop","menu":[{"id":"stop","label":"Remove","confirm":true}]}]`, ""),
		"primary button":             screenJSON(`[]`, `[{"button":"Go","action":"start","style":"primary","enabled":"!$on"}]`),
		"buttons enabled from state": screenJSON(`[]`, `[{"button":"Start","action":"start","enabled":"!$on"},{"button":"Stop","action":"stop","style":"primary","enabled":"$on"}]`),
		"field with an enabled flag": screenJSON(`[{"field":"Name","bind":"on","enabled":"!$on"}]`, ""),
		"comment key":                `{"_note":"x","kit":1,"start":"main","screens":{"main":{"title":"Main"}}}`,
	}
	for name, document := range cases {
		t.Run(name, func(t *testing.T) {
			if report := validate(t, document); !report.OK() {
				t.Fatalf("unexpected errors: %v", report.Errors)
			}
		})
	}
}

func TestRulesRejectWithCode(t *testing.T) {
	cases := []struct {
		name     string
		document string
		code     string
	}{
		{"not json", `{`, CodeNotJSON},
		{"wrong kit", strings.Replace(screenJSON(`[]`, ""), `"kit":1`, `"kit":2`, 1), CodeBadKit},
		{"unknown top-level key", strings.Replace(screenJSON(`[]`, ""), `"kit":1`, `"kit":1,"extra":1`, 1), CodeUnknownProp},
		{"unknown start screen", strings.Replace(screenJSON(`[]`, ""), `"start":"main"`, `"start":"nope"`, 1), CodeUnknownScreen},
		{"unknown component", screenJSON(`[{"carousel":"x"}]`, ""), CodeUnknownComponent},
		{"two components in one object", screenJSON(`[{"text":"a","toggle":"b","bind":"on"}]`, ""), CodeAmbiguous},
		{"unknown prop", screenJSON(`[{"text":"a","colour":"red"}]`, ""), CodeUnknownProp},
		{"missing required prop", screenJSON(`[{"toggle":"Wi-Fi"}]`, ""), CodeMissingProp},
		{"missing button action", screenJSON(`[{"button":"Go"}]`, ""), CodeMissingProp},
		{"bad enum", screenJSON(`[{"text":"a","style":"huge"}]`, ""), CodeBadEnum},
		{"wrong type", screenJSON(`[{"text":5}]`, ""), CodeWrongType},
		{"bare dollar", screenJSON(`[{"text":"costs $5"}]`, ""), CodeBadBinding},
		{"index binding", screenJSON(`[{"text":"$items[0]"}]`, ""), CodeBadBinding},
		{"bad flag", screenJSON(`[{"toggle":"x","bind":"on","enabled":"yes"}]`, ""), CodeBadBinding},
		{"field flag without $", screenJSON(`[{"field":"Name","bind":"on","enabled":"yes"}]`, ""), CodeBadBinding},
		{"field flag on undeclared var", screenJSON(`[{"field":"Name","bind":"on","enabled":"$ghost"}]`, ""), CodeUnknownVar},
		{"button flag without $", screenJSON(`[]`, `[{"button":"Go","action":"start","enabled":"on"}]`), CodeBadBinding},
		{"button flag on undeclared var", screenJSON(`[]`, `[{"button":"Go","action":"start","enabled":"$ghost"}]`), CodeUnknownVar},
		{"button flag not a flag", screenJSON(`[]`, `[{"button":"Go","action":"start","enabled":1}]`), CodeWrongType},
		{"list needs pure binding", screenJSON(`[{"list":"items"}]`, ""), CodeBadBinding},
		{"undeclared var", screenJSON(`[{"text":"$ghost"}]`, ""), CodeUnknownVar},
		{"undeclared bind", screenJSON(`[{"toggle":"x","bind":"ghost"}]`, ""), CodeUnknownVar},
		{"list var not array", screenJSON(`[{"list":"$pin"}]`, ""), CodeListNotArray},
		{"unknown action", screenJSON(`[]`, `[{"button":"Go","action":"fly"}]`), CodeUnknownAction},
		{"unknown tap action", screenJSON(`[{"list":"$items","tap":"fly"}]`, ""), CodeUnknownAction},
		{"two primary buttons", screenJSON(`[]`, `[{"button":"A","action":"start","style":"primary"},{"button":"B","action":"stop","style":"primary"}]`), CodeMultiplePrimary},
		{"empty state counts as primary", screenJSON(`[{"empty":"None","button":"Go","action":"start"}]`, `[{"button":"B","action":"stop","style":"primary"}]`), CodeMultiplePrimary},
		{"empty button without action", screenJSON(`[{"empty":"None","button":"Go"}]`, ""), CodeNeedsTogether},
		{"missing english", screenJSON(`[{"text":{"vi":"Chào"}}]`, ""), CodeMissingEnglish},
		{"missing locale key", screenJSON(`[{"text":"@ui.nope"}]`, ""), CodeMissingLocaleKey},
		{"bad locale key shape", screenJSON(`[{"text":"@Nope"}]`, ""), CodeBadLocaleKey},
		{"too long for e-ink", screenJSON(`[{"section":"`+strings.Repeat("x", 41)+`"}]`, ""), CodeTextTooLong},
		{"colon in button label", screenJSON(`[]`, `[{"button":"Step 1: go","action":"start"}]`), CodeTextHasColon},
		{"colon in menu label", screenJSON(`[{"list":"$items","menu":[{"id":"a","label":"A: b"}]}]`, ""), CodeTextHasColon},
		{"too many radio options", screenJSON(`[{"radio":"L","bind":"pin","options":[["a","A"],["b","B"],["c","C"],["d","D"],["e","E"],["f","F"],["g","G"],["h","H"],["i","I"]]}]`, ""), CodeTooManyOptions},
		{"duplicate radio value", screenJSON(`[{"radio":"L","bind":"pin","options":[["a","A"],["a","B"]]}]`, ""), CodeDuplicate},
		{"digits out of range", screenJSON(`[{"bigcode":"$pin","digits":40}]`, ""), CodeOutOfRange},
		{"row with action and push", screenJSON(`[{"row":"R","action":"start","push":"main"}]`, ""), CodeNeedsOne},
		{"header action needs one", `{"kit":1,"start":"main","screens":{"main":{"title":"M","action":{"label":"Go"}}}}`, CodeNeedsOne},
		{"push to unknown screen", `{"kit":1,"start":"main","screens":{"main":{"title":"M","action":{"label":"Go","push":"nope"}}}}`, CodeUnknownScreen},
		{"dialog action unknown", `{"kit":1,"start":"main","screens":{"main":{"title":"M","dialogs":{"d":{"title":"T","text":"x","ok":"Remove","action":"fly"}}}}}`, CodeUnknownAction},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			report := validate(t, tc.document)
			if !report.HasCode(tc.code) {
				t.Fatalf("want error %s, got %v", tc.code, report.Errors)
			}
		})
	}
}

func TestIssueMessagesNameThePathAndTheFix(t *testing.T) {
	report := validate(t, screenJSON(`[{"toggle":"Wi-Fi"}]`, ""))
	issue := report.Errors[0]
	if issue.Path != "screens.main.body[0]" || !strings.Contains(issue.Message, `"bind"`) {
		t.Fatalf("unhelpful issue: %s", issue)
	}
}

func TestTranslationWarnings(t *testing.T) {
	longVi := screenJSON(`[{"text":{"en":"Keep awake","vi":"Giữ cho thiết bị luôn luôn thức nhé"}}]`, "")
	report := validate(t, longVi)
	if len(report.Warnings) == 0 || report.Warnings[0].Code != CodeTranslationLong {
		t.Fatalf("want a translation-length warning, got %v", report.Warnings)
	}
	untranslated := validate(t, screenJSON(`[{"text":"@ui.stop"}]`, ""))
	if len(untranslated.Warnings) == 0 || untranslated.Warnings[0].Code != CodeLocaleMissingVi {
		t.Fatalf("want a missing-translation warning, got %v", untranslated.Warnings)
	}
}

func TestSecretTokenFieldWarns(t *testing.T) {
	document := `{"kit":1,"start":"main","vars":{"apiToken":""},"screens":{"main":{"title":"M","body":[{"field":"Token","bind":"apiToken","secret":true}]}}}`
	report := validate(t, document)
	if !report.OK() || len(report.Warnings) != 1 || report.Warnings[0].Code != CodeSecretField {
		t.Fatalf("want one secret-field warning, got errors=%v warnings=%v", report.Errors, report.Warnings)
	}
}

func TestExtraScreenFilesAreMerged(t *testing.T) {
	main := `{"kit":1,"start":"main","screens":{"main":{"title":"M","action":{"label":"More","push":"more"}}}}`
	extra := map[string][]byte{"more": []byte(`{"title":"More"}`)}
	document, report := ParseAndValidate([]byte(main), extra, testEnv())
	if !report.OK() || document.Screens["more"] == nil {
		t.Fatalf("screens/more.json not merged: %v", report.Errors)
	}
	_, duplicate := ParseAndValidate([]byte(main), map[string][]byte{"main": []byte(`{"title":"X"}`)}, testEnv())
	if !duplicate.HasCode(CodeDuplicate) {
		t.Fatalf("want a duplicate error, got %v", duplicate.Errors)
	}
}

func TestDocumentTextFallsBackToEnglish(t *testing.T) {
	document, report := ParseAndValidate([]byte(screenJSON(`[{"text":"@ui.stop"}]`, "")), nil, testEnv())
	if !report.OK() {
		t.Fatal(report.Errors)
	}
	if got := document.Text("@ui.stop", "vi"); got != "Stop" {
		t.Errorf("fallback = %q, want Stop", got)
	}
	if got := document.Text("@ui.go", "vi"); got != "Đi" {
		t.Errorf("vi = %q, want Đi", got)
	}
	if got := document.Text(map[string]any{"en": "A", "vi": "B"}, "vi"); got != "B" {
		t.Errorf("map = %q, want B", got)
	}
}

func TestTemplates(t *testing.T) {
	refs, err := ParseTemplate("a $x and $device.name, $$ done")
	if err != nil || len(refs) != 2 || refs[0] != "x" || refs[1] != "device.name" {
		t.Fatalf("refs=%v err=%v", refs, err)
	}
	vars := map[string]any{"x": 3.0, "device": map[string]any{"name": "Libra"}}
	if got := Substitute("a $x and $device.name, $$ done", vars); got != "a 3 and Libra, $ done" {
		t.Errorf("Substitute = %q", got)
	}
	if !Truthy("!$off", map[string]any{"off": false}) || Truthy("$items", map[string]any{"items": []any{}}) {
		t.Error("Truthy wrong")
	}
}
