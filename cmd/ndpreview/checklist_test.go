package main

import "testing"

const checklistVars = `{"tasks":["Buy milk",["Call Anna","✓"],["Pay rent","5 min"]]}`

func iconsNamed(page Page, name string) []Item {
	var found []Item
	for _, item := range page.Items {
		if item.Shape == "icon" && item.Label == name {
			found = append(found, item)
		}
	}
	return found
}

func TestChecklistRowsHaveABoxAtTheLeft(t *testing.T) {
	page := layoutBody(t, checklistVars, `{"list":"$tasks","style":"checklist"}`, ``)
	open, done := iconsNamed(page, "square"), iconsNamed(page, "square-check")
	if len(open) != 2 || len(done) != 1 {
		t.Fatalf("want 2 empty boxes and 1 ticked box, got %d and %d", len(open), len(done))
	}
	for _, box := range append(open, done...) {
		if box.X != gutter || box.W != checkboxIconSize || box.H != checkboxIconSize {
			t.Errorf("box %+v, want %d px at the left gutter", box, checkboxIconSize)
		}
	}
}

func TestChecklistDoneRowHasNoCheckIconAtTheRight(t *testing.T) {
	page := layoutBody(t, checklistVars, `{"list":"$tasks","style":"checklist"}`, ``)
	if checks := iconsNamed(page, "check"); len(checks) != 0 {
		t.Errorf("a done item is the ticked box, not a second check icon: %+v", checks)
	}
	if shown := textItems(page, "✓"); len(shown) != 0 {
		t.Errorf("the done mark must not be drawn as a glyph: %+v", shown)
	}
}

func TestChecklistKeepsTheCaptionAndTheTextBaseline(t *testing.T) {
	plain := layoutBody(t, checklistVars, `{"list":"$tasks"}`, ``)
	checklist := layoutBody(t, checklistVars, `{"list":"$tasks","style":"checklist"}`, ``)
	before, after := textItems(plain, "Buy milk"), textItems(checklist, "Buy milk")
	if len(before) != 1 || len(after) != 1 {
		t.Fatalf("want one primary text each, got %d and %d", len(before), len(after))
	}
	if before[0].Y != after[0].Y || before[0].H != after[0].H {
		t.Errorf("the box moved the text: y %d -> %d", before[0].Y, after[0].Y)
	}
	if after[0].X != gutter+checkboxIconSize+rhythm {
		t.Errorf("text starts at %d, want right of the box at %d", after[0].X, gutter+checkboxIconSize+rhythm)
	}
	if len(textItems(checklist, "5 min")) != 1 {
		t.Error("a secondary text that is not the done mark stays a caption")
	}
}

func TestPlainListIgnoresTheChecklistRule(t *testing.T) {
	page := layoutBody(t, checklistVars, `{"list":"$tasks"}`, ``)
	if len(iconsNamed(page, "square"))+len(iconsNamed(page, "square-check")) != 0 {
		t.Error("a plain list draws no boxes")
	}
	if len(iconsNamed(page, "check")) != 1 {
		t.Error("a plain list keeps the check icon at the right")
	}
}
