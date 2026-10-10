package drawer

import (
	"koboapps/internal/uikit"
)

const (
	sampleLookSelected = "selected"
	sampleLookDisabled = "disabled"
	defaultSampleLook  = "normal"
)

// samplesComponent resolves a `samples` block: every cell is the resolved inner component with
// its look and caption. The on state and the disabled state are decided here, so the mod draws
// what the data says and only adds the pressed inversion and the generic dimming.
func (s screenContext) samplesComponent(c uikit.Component) ScreenComponent {
	items, _ := c.Main().([]any)
	columns := 1
	if declared, isNumber := c.Prop("columns").(float64); isNumber {
		columns = int(declared)
	}
	cells := make([]ScreenComponent, 0, len(items))
	for _, item := range items {
		if object, isObject := item.(map[string]any); isObject {
			cells = append(cells, s.sampleCell(object))
		}
	}
	return ScreenComponent{Columns: columns, Cells: cells}
}

func (s screenContext) sampleCell(object map[string]any) ScreenComponent {
	inner := map[string]any{}
	for key, value := range object {
		if key != "look" && key != "caption" {
			inner[key] = value
		}
	}
	kind := sampleKind(inner)
	cell := s.component(uikit.Component{Kind: kind, Props: inner})
	cell.Look = stringProp(object, "look")
	if cell.Look == "" {
		cell.Look = defaultSampleLook
	}
	cell.Caption = s.text(object["caption"])
	s.applySampleLook(&cell, inner)
	return cell
}

// sampleKind is the component key of a validated cell: the one key that is not a property of
// the others.
func sampleKind(inner map[string]any) string {
	for _, key := range uikit.SampleableKeys() {
		if _, found := inner[key]; found && key != "text" {
			return key
		}
	}
	return "text"
}

// applySampleLook sets the data a look implies. Inner props never bind variables, so a cell's
// value is decided by its look alone.
func (s screenContext) applySampleLook(cell *ScreenComponent, inner map[string]any) {
	isEnabled := cell.Look != sampleLookDisabled
	isSelected := cell.Look == sampleLookSelected
	switch cell.Type {
	case "toggle":
		cell.Value, cell.Enabled = isSelected, &isEnabled
	case "radio":
		cell.Value = ""
		if isSelected && len(cell.Options) > 0 {
			cell.Value = cell.Options[0][0]
		}
		cell.Enabled = &isEnabled
	case "field":
		cell.Value = s.text(inner["value"])
		cell.Enabled = &isEnabled
	case "button":
		cell.Enabled = &isEnabled
	}
}
