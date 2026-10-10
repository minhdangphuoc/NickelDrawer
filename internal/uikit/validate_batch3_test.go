package uikit

import (
	"strings"
	"testing"
)

// samplesJSON puts one samples component in the body of a screen with a declared flag and action.
func samplesJSON(samples, columns string) string {
	extra := ""
	if columns != "" {
		extra = `,"columns":` + columns
	}
	return screenJSON(`[{"samples":`+samples+extra+`}]`, "")
}

func TestSamplesAcceptStaticCells(t *testing.T) {
	cases := map[string]string{
		"buttons in three columns": samplesJSON(`[
			{"button":"Sync","style":"primary","look":"normal","caption":"Normal"},
			{"button":"Sync","style":"primary","look":"pressed","caption":"Pressed"},
			{"button":"Sync","style":"primary","look":"disabled","caption":"Disabled"}]`, "3"),
		"toggle on and off":  samplesJSON(`[{"toggle":"Wi-Fi","look":"normal"},{"toggle":"Wi-Fi","look":"selected","caption":"On"}]`, ""),
		"radio with one":     samplesJSON(`[{"radio":"Mode","options":[["a","Fast"]],"look":"selected"}]`, ""),
		"row with chevron":   samplesJSON(`[{"row":"Refresh","detail":"Balanced","action":"start"}]`, ""),
		"kv, notice, steps":  samplesJSON(`[{"kv":[["IP","10.0.0.2"]]},{"notice":"Tip","title":"Hint"},{"steps":{"label":"Downloading","done":"2","total":"3"}}]`, ""),
		"field and text":     samplesJSON(`[{"field":"Name","placeholder":"owner/name"},{"text":"Reading list","style":"heading"}]`, ""),
		"sample is no input": samplesJSON(`[{"toggle":"Wi-Fi","enabled":false,"look":"disabled"}]`, ""),
	}
	for name, document := range cases {
		t.Run(name, func(t *testing.T) {
			if report := validate(t, document); !report.OK() {
				t.Fatalf("unexpected errors: %v", report.Errors)
			}
		})
	}
}

func TestSamplesRejectWithCode(t *testing.T) {
	cases := []struct {
		name, document, code string
	}{
		{"list cannot be sampled", samplesJSON(`[{"list":"$items"}]`, ""), CodeSampleNotStatic},
		{"gallery cannot be sampled", samplesJSON(`[{"gallery":"$items"}]`, ""), CodeSampleNotStatic},
		{"qr cannot be sampled", samplesJSON(`[{"qr":"x"}]`, ""), CodeSampleNotStatic},
		{"nested samples", samplesJSON(`[{"samples":[{"button":"x"}]}]`, ""), CodeSampleNotStatic},
		{"push is navigation", samplesJSON(`[{"row":"R","push":"main"}]`, ""), CodeSampleNotStatic},
		{"bind is live", samplesJSON(`[{"toggle":"T","bind":"on"}]`, ""), CodeSampleNotStatic},
		{"selected needs an on state", samplesJSON(`[{"button":"B","look":"selected"}]`, ""), CodeBadEnum},
		{"unknown look", samplesJSON(`[{"button":"B","look":"hover"}]`, ""), CodeBadEnum},
		{"binding in text", samplesJSON(`[{"text":"Hello $pin"}]`, ""), CodeBadBinding},
		{"binding in flag", samplesJSON(`[{"button":"B","enabled":"$on"}]`, ""), CodeBadBinding},
		{"too many radio options", samplesJSON(`[{"radio":"R","options":[["a","A"],["b","B"]]}]`, ""), CodeOutOfRange},
		{"radio without an option", samplesJSON(`[{"radio":"R"}]`, ""), CodeOutOfRange},
		{"more than one kv row", samplesJSON(`[{"kv":[["A","1"],["B","2"]]}]`, ""), CodeOutOfRange},
		{"columns need buttons", samplesJSON(`[{"row":"R"}]`, "2"), CodeOutOfRange},
		{"columns above three", samplesJSON(`[{"button":"B"}]`, "4"), CodeOutOfRange},
		{"empty", samplesJSON(`[]`, ""), CodeOutOfRange},
		{"caption too long", samplesJSON(`[{"button":"B","caption":"`+strings.Repeat("x", 25)+`"}]`, ""), CodeTextTooLong},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if report := validate(t, tc.document); !report.HasCode(tc.code) {
				t.Fatalf("want error %s, got %v", tc.code, report.Errors)
			}
		})
	}
}

func TestSamplePrimaryDoesNotCount(t *testing.T) {
	document := screenJSON(`[{"samples":[{"button":"Sync","style":"primary"}]}]`, `[{"button":"Go","action":"start","style":"primary"}]`)
	if report := validate(t, document); !report.OK() {
		t.Fatalf("a primary picture counted as a primary action: %v", report.Errors)
	}
}

func TestButtonPushesOrRuns(t *testing.T) {
	good := `{"kit":1,"start":"main","screens":{"main":{"title":"M","footer":[{"button":"Pair","push":"pair","style":"primary"}]},"pair":{"title":"P"}}}`
	if report := validate(t, good); !report.OK() {
		t.Fatalf("a button may push a screen: %v", report.Errors)
	}
	both := strings.Replace(good, `"push":"pair"`, `"push":"pair","action":"start"`, 1)
	if report := validate(t, both); !report.HasCode(CodeNeedsOne) {
		t.Fatalf("action and push together must fail: %v", report.Errors)
	}
	confirmed := strings.Replace(good, `"push":"pair"`, `"push":"pair","confirm":{"text":"Sure?","ok":"Yes"}`, 1)
	if report := validate(t, confirmed); !report.HasCode(CodeNeedsTogether) {
		t.Fatalf("a pushing button cannot confirm: %v", report.Errors)
	}
	orphan := strings.Replace(good, `"push":"pair"`, `"action":"start"`, 1)
	if report := validate(t, orphan); !report.HasCode(CodeFlowUnreachable) {
		t.Fatalf("a button's push is part of the flow, so an unpushed screen stays unreachable: %v", report.Errors)
	}
}

func TestRowConfirmNeedsAnAction(t *testing.T) {
	if report := validate(t, screenJSON(`[{"row":"Fix","action":"start","confirm":{"text":"Sure?","ok":"Fix"},"show":"$on"}]`, "")); !report.OK() {
		t.Fatalf("row with confirm and show: %v", report.Errors)
	}
	pushing := `{"kit":1,"start":"main","screens":{"main":{"title":"M","body":[{"row":"R","push":"more","confirm":{"text":"x","ok":"Go"}}]},"more":{"title":"More"}}}`
	if report := validate(t, pushing); !report.HasCode(CodeNeedsTogether) {
		t.Fatalf("confirm on a pushing row must fail: %v", report.Errors)
	}
}

func TestSeveralPrimariesNeedOppositeShowFlags(t *testing.T) {
	two := func(first, second string) string {
		return screenJSON(`[]`, `[{"button":"Start","action":"start","style":"primary"`+first+`},{"button":"Stop","action":"stop","style":"primary"`+second+`}]`)
	}
	if report := validate(t, two(`,"show":"!$on"`, `,"show":"$on"`)); !report.OK() {
		t.Fatalf("opposite show flags: %v", report.Errors)
	}
	for name, document := range map[string]string{
		"no flags":       two("", ""),
		"one flag only":  two(`,"show":"$on"`, ""),
		"the same flag":  two(`,"show":"$on"`, `,"show":"$on"`),
		"flag and shown": two("", `,"show":"$on"`),
	} {
		if report := validate(t, document); !report.HasCode(CodeMultiplePrimary) {
			t.Errorf("%s: want ui.multiple_primary, got %v", name, report.Errors)
		}
	}
}

func warningCodes(report Report) []string {
	var codes []string
	for _, warning := range report.Warnings {
		codes = append(codes, warning.Code)
	}
	return codes
}

func hasWarning(report Report, code string) bool {
	for _, warning := range report.Warnings {
		if warning.Code == code {
			return true
		}
	}
	return false
}

func TestPrimaryDisabledWhileOtherEnabledWarns(t *testing.T) {
	bad := screenJSON(`[]`, `[{"button":"Start","action":"start","enabled":"!$on"},{"button":"Stop","action":"stop","style":"primary","enabled":"$on"}]`)
	if report := validate(t, bad); !report.OK() || !hasWarning(report, CodePrimaryDisabled) {
		t.Fatalf("want the A8 warning, got errors=%v warnings=%v", report.Errors, warningCodes(report))
	}
	fixed := screenJSON(`[]`, `[{"button":"Start","action":"start","style":"primary","show":"!$on"},{"button":"Stop","action":"stop","style":"primary","show":"$on"}]`)
	if report := validate(t, fixed); !report.OK() || hasWarning(report, CodePrimaryDisabled) {
		t.Fatalf("show flags promote the possible action: errors=%v warnings=%v", report.Errors, warningCodes(report))
	}
	independent := screenJSON(`[{"button":"Secondary","action":"stop"}]`, `[{"button":"Primary","action":"start","style":"primary","enabled":"$on"}]`)
	if report := validate(t, independent); hasWarning(report, CodePrimaryDisabled) {
		t.Fatalf("an unrelated secondary is no alternative: %v", warningCodes(report))
	}
}

func TestStyleWarnings(t *testing.T) {
	cases := []struct{ name, document, code string }{
		{"typed capitals", screenJSON(`[{"section":"INSTALLED"},{"kv":[["A","1"]]}]`, ""), CodeUppercaseText},
		{"vietnamese capitals", screenJSON(`[{"section":{"en":"Problems","vi":"SỰ CỐ"}},{"kv":[["A","1"]]}]`, ""), CodeUppercaseText},
		{"caption with a full stop", screenJSON(`[{"text":"Takes effect after a restart.","style":"caption"}]`, ""), CodeCaptionPeriod},
		{"qr caption with a full stop", screenJSON(`[{"qr":"https://example.org","caption":"Scan with your phone."}]`, ""), CodeCaptionPeriod},
		{"section over one paragraph", screenJSON(`[{"section":"About"},{"text":"A sentence."}]`, ""), CodeSectionOverParagraph},
		{"section over a notice", screenJSON(`[{"section":"Tip"},{"notice":"Look here."}]`, ""), CodeSectionOverParagraph},
		{"section over nothing", screenJSON(`[{"section":"One"},{"section":"Two"},{"kv":[["A","1"]]}]`, ""), CodeSectionEmpty},
		{"status wider than its slot", `{"kit":1,"start":"main","screens":{"main":{"title":"M","status":"Đang ở màn hình chờ: 999 hình ảnh nhé"}}}`, CodeStatusTooWide},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			report := validate(t, tc.document)
			if !report.OK() || !hasWarning(report, tc.code) {
				t.Fatalf("want warning %s, got errors=%v warnings=%v", tc.code, report.Errors, warningCodes(report))
			}
		})
	}
}

func TestStyleRulesLetGoodContentThrough(t *testing.T) {
	cases := map[string]string{
		"acronyms stay":                 screenJSON(`[{"text":"Scan the QR code or type the PIN into the URL bar"}]`, ""),
		"two sentences keep the period": screenJSON(`[{"text":"Hold a row. It opens a menu.","style":"caption"}]`, ""),
		"ellipsis is not a full stop":   screenJSON(`[{"text":"Working…","style":"caption"}]`, ""),
		"body text keeps its full stop": screenJSON(`[{"text":"One sentence."}]`, ""),
		"section over rows":             screenJSON(`[{"section":"Device"},{"kv":[["A","1"]]},{"text":"Note."}]`, ""),
		"status that fits":              `{"kit":1,"start":"main","screens":{"main":{"title":"M","status":"Kit 1 · 18"}}}`,
		"bound status is not measured":  `{"kit":1,"start":"main","vars":{"s":""},"screens":{"main":{"title":"M","status":"$s"}}}`,
		"heading before rows is a text": screenJSON(`[{"text":"Title","style":"heading"},{"section":"Rows"},{"kv":[["A","1"]]}]`, ""),
	}
	for name, document := range cases {
		t.Run(name, func(t *testing.T) {
			report := validate(t, document)
			if !report.OK() || len(report.Warnings) != 0 {
				t.Fatalf("errors=%v warnings=%v", report.Errors, report.Warnings)
			}
		})
	}
}

func TestLocaleStatusKeysAreMeasured(t *testing.T) {
	env := testEnv()
	env.Catalogs["en"]["ui.status_shown"] = "On the sleep screen: {count} pictures in rotation"
	env.Catalogs["en"]["ui.status_ok"] = "Ready"
	_, report := ParseAndValidate([]byte(screenJSON(`[]`, "")), nil, env)
	if !hasWarning(report, CodeStatusTooWide) {
		t.Fatalf("a long status key must warn: %v", warningCodes(report))
	}
	for _, warning := range report.Warnings {
		if strings.Contains(warning.Path, "ui.status_ok") {
			t.Errorf("a short status must pass: %s", warning)
		}
	}
	if !IsStatusKey("msg.state_status_pairing") || !IsStatusKey("ui.pictures_status") || IsStatusKey("msg.err_http_status") {
		t.Error("IsStatusKey picks the wrong keys")
	}
}

func TestUppercaseWords(t *testing.T) {
	cases := map[string][]string{
		"INSTALLED":                    {"INSTALLED"},
		"Installed":                    nil,
		"SỰ CỐ":                        {"SỰ", "CỐ"},
		"Scan the QR code or PIN":      nil,
		"USB, NFC, EPUB, IP":           nil,
		"Fix names (NFC)":              nil,
		"System":                       nil,
		"OK":                           nil,
		"Turn on Wi-Fi":                nil,
		"Update NickelDrawer and Apps": nil,
	}
	for text, want := range cases {
		got := UppercaseWords(text)
		if len(got) != len(want) {
			t.Errorf("UppercaseWords(%q) = %v, want %v", text, got, want)
		}
	}
}

func TestEndsWithFullStop(t *testing.T) {
	cases := map[string]bool{
		"Takes effect after a restart.": true,
		"Two. Sentences.":               false,
		"Is it? Yes.":                   false,
		"Wait…":                         false,
		"Wait...":                       false,
		"No stop":                       false,
		"v1.2 build":                    false,
	}
	for text, want := range cases {
		if got := EndsWithFullStop(text); got != want {
			t.Errorf("EndsWithFullStop(%q) = %v, want %v", text, got, want)
		}
	}
}

func TestEstimatedWidthIsInTheRightRange(t *testing.T) {
	// Real Avenir-class text runs about 0.5 em per character at 32 px (the audit measured
	// "Luân phiên: Ngẫu nhiên" at 352 px).
	if width := EstimateWidth("Luân phiên: Ngẫu nhiên", 32, false); width < 300 || width > 400 {
		t.Errorf("width = %d", width)
	}
	if narrow, wide := EstimateWidth("illi", 32, false), EstimateWidth("MWMW", 32, false); narrow >= wide {
		t.Errorf("narrow letters (%d) must be thinner than wide ones (%d)", narrow, wide)
	}
	if regular, bold := EstimateWidth("Status", 32, false), EstimateWidth("Status", 32, true); bold <= regular {
		t.Errorf("bold (%d) must be wider than regular (%d)", bold, regular)
	}
}
