package uikit

import "testing"

func checklistDocument(items string) string {
	return `{"kit":1,"start":"main","vars":{"tasks":` + items + `},
	"screens":{"main":{"title":"Tasks","body":[{"list":"$tasks","style":"checklist"}]}}}`
}

func TestChecklistAcceptsTextsAndDoneMarks(t *testing.T) {
	report := validate(t, checklistDocument(`["Buy milk",["Call Anna","✓"],["Pay rent",""]]`))
	if len(report.Errors) != 0 {
		t.Fatalf("unexpected errors: %v", report.Errors)
	}
}

func TestChecklistAcceptsAnEmptyList(t *testing.T) {
	if report := validate(t, checklistDocument(`[]`)); len(report.Errors) != 0 {
		t.Fatalf("unexpected errors: %v", report.Errors)
	}
}

func TestChecklistRejectsItemsThatCannotCarryADoneMark(t *testing.T) {
	for name, items := range map[string]string{
		"object":       `[{"title":"Buy milk"}]`,
		"empty pair":   `[[]]`,
		"triple":       `[["a","✓","c"]]`,
		"bare boolean": `[true]`,
	} {
		t.Run(name, func(t *testing.T) {
			if report := validate(t, checklistDocument(items)); !report.HasCode(CodeChecklistItems) {
				t.Fatalf("want %s, got %v", CodeChecklistItems, report.Errors)
			}
		})
	}
}

func TestPlainListStyleNeedsNoDoneMarks(t *testing.T) {
	document := `{"kit":1,"start":"main","vars":{"tasks":[{"title":"x"}]},
	"screens":{"main":{"title":"Tasks","body":[{"list":"$tasks","style":"plain"}]}}}`
	if report := validate(t, document); report.HasCode(CodeChecklistItems) {
		t.Fatalf("a plain list must not be held to the checklist rule: %v", report.Errors)
	}
}
