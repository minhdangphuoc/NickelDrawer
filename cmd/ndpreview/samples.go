package main

import (
	"koboapps/internal/uikit"
)

// The `samples` component (docs/uikit-plan.md 5.1): a states matrix drawn by the same code as the
// live components, but as pictures. Rules, mirrored by mod/drawer/screen_layout.cc:
//
//  1. Cell width is (contentWidth - (columns-1) x rhythm) / columns; only button cells may share
//     a row, so every other cell is contentWidth wide.
//  2. A cell is the inner component's blocks (as the live component lays them out) with its caption
//     under it: an 8 px gap and one caption line (muted, centred), reserved even when empty. Cell
//     rows are as high as their tallest cell and sit rhythm apart.
//  3. A button cell is a fixed-size control: one centred line, elided, buttonHeight high.
//  4. Looks: "pressed" paints the cell rectangle black and inverts every grey of the inner items
//     (255 - level; icons turn white); "disabled" raises every ink and fill darker than ink.dim to
//     ink.dim; "selected" is the on state of a toggle or radio; "normal" is as is.
//  5. A cell has no event, no confirm and no target.
//  6. The whole block never splits across pages and has a rhythm gap after it.
const sampleCaptionGap = 8

type sampleCell struct {
	inner    []block
	width    int
	height   int // the inner blocks, caption excluded
	look     string
	caption  string
	isButton bool
	label    string
	isOn     bool
	primary  bool
	enabled  bool
}

func (rc renderContext) samplesBlocks(c uikit.Component) []block {
	items, _ := c.Main().([]any)
	columns := 1
	if declared, isNumber := c.Prop("columns").(float64); isNumber {
		columns = max(1, int(declared))
	}
	cellWidth := (contentWidth - (columns-1)*rhythm) / columns
	var cells []sampleCell
	for _, raw := range items {
		if object, isObject := raw.(map[string]any); isObject {
			cells = append(cells, rc.sampleCell(object, cellWidth))
		}
	}
	rows := sampleRows(cells, columns)
	height := 0
	for index, row := range rows {
		height += sampleRowHeight(row)
		if index < len(rows)-1 {
			height += rhythm
		}
	}
	return []block{{height: height, gap: rhythm, draw: func(y int) []Item {
		var drawn []Item
		for _, row := range rows {
			for column, cell := range row {
				drawn = append(drawn, cell.items(gutter+column*(cellWidth+rhythm), y)...)
			}
			y += sampleRowHeight(row) + rhythm
		}
		return drawn
	}}}
}

func sampleRows(cells []sampleCell, columns int) [][]sampleCell {
	var rows [][]sampleCell
	for start := 0; start < len(cells); start += columns {
		rows = append(rows, cells[start:min(start+columns, len(cells))])
	}
	return rows
}

func sampleRowHeight(row []sampleCell) int {
	tallest := 0
	for _, cell := range row {
		tallest = max(tallest, cell.totalHeight())
	}
	return tallest
}

// totalHeight always reserves the caption line, so a matrix keeps its rhythm whatever is captioned.
func (cell sampleCell) totalHeight() int {
	return cell.height + sampleCaptionGap + captionLineH
}

// sampleCell resolves one cell. The inner component is laid out by the live code with the state
// its look implies, so a sample can never drift from the real control.
func (rc renderContext) sampleCell(object map[string]any, width int) sampleCell {
	look, _ := object["look"].(string)
	if look == "" {
		look = "normal"
	}
	inner := map[string]any{}
	for key, value := range object {
		if key != "look" && key != "caption" {
			inner[key] = value
		}
	}
	cell := sampleCell{width: width, look: look, caption: rc.text(object["caption"]), enabled: look != "disabled"}
	cell.isOn = look == "selected"
	kind := sampleComponentKind(inner)
	if kind == "button" {
		cell.isButton, cell.label = true, rc.text(inner["button"])
		cell.primary = inner["style"] == "primary"
		cell.height = buttonHeight
		return cell
	}
	sampleRC, props := rc.sampleContext(kind, inner, cell)
	cell.inner = sampleRC.componentBlocks(uikit.Component{Kind: kind, Props: props})
	cell.height = blocksHeightNoTail(cell.inner)
	return cell
}

func sampleComponentKind(inner map[string]any) string {
	for _, key := range uikit.SampleableKeys() {
		if _, found := inner[key]; found && key != "text" {
			return key
		}
	}
	return "text"
}

// sampleContext binds the inner component to throwaway variables that carry its look, so the live
// toggle / radio / field code draws the on and disabled states.
func (rc renderContext) sampleContext(kind string, inner map[string]any, cell sampleCell) (renderContext, map[string]any) {
	props := map[string]any{}
	for key, value := range inner {
		props[key] = value
	}
	vars := map[string]any{"sampleEnabled": cell.enabled, "sampleOn": cell.isOn, "sampleChoice": "", "sampleValue": ""}
	switch kind {
	case "toggle":
		props["bind"], props["enabled"] = "sampleOn", "$sampleEnabled"
	case "radio":
		props["bind"], props["enabled"] = "sampleChoice", "$sampleEnabled"
		if options, _ := inner["options"].([]any); cell.isOn && len(options) > 0 {
			first, _ := options[0].([]any)
			vars["sampleChoice"] = first[0]
		}
	case "row":
		if props["action"] == nil && props["push"] == nil {
			props["action"] = "sample" // a row cell is drawn with its chevron; withoutEvents makes it inert
		}
	case "field":
		props["bind"], props["enabled"] = "sampleValue", "$sampleEnabled"
		vars["sampleValue"] = rc.text(inner["value"])
		delete(props, "value")
	}
	derived := rc
	derived.vars = vars
	return derived, props
}

func blocksHeightNoTail(blocks []block) int { return blocksHeight(blocks) }

// items draws the cell with its top-left corner at (x, y).
func (cell sampleCell) items(x, y int) []Item {
	var drawn []Item
	if cell.isButton {
		drawn = cell.buttonItems(x, y)
	} else {
		drawn = drawBlocks(cell.inner, y)
	}
	drawn = withoutEvents(drawn)
	switch cell.look {
	case "pressed":
		drawn = invertedCell(drawn, x, y, cell.width, cell.height)
	case "disabled":
		drawn = dimmedItems(drawn)
	}
	if cell.caption != "" {
		caption := textItem(x, y+cell.height+sampleCaptionGap, cell.width, captionLineH,
			elide(cell.caption, cell.width, regular(captionSize, familySans), false), captionSize, familySans, "center")
		caption.Ink = inkMuted
		drawn = append(drawn, caption)
	}
	return drawn
}

func (cell sampleCell) buttonItems(x, y int) []Item {
	lines := []string{elide(cell.label, cell.width-2*rhythm, bold(bodySize, familySans), false)}
	return buttonItems(buttonLayout{x: x, y: y, width: cell.width, height: buttonHeight}, lines, cell.primary, cell.enabled, "", "", cell.label)
}

// withoutEvents makes the items inert: a sample is a picture.
func withoutEvents(items []Item) []Item {
	for index := range items {
		items[index].Event, items[index].Action, items[index].Confirm, items[index].Close = "", "", "", false
	}
	return items
}

// invertedCell is the pressed look: PressArea's HIGHLIGHT_INVERT on the cell rectangle.
func invertedCell(items []Item, x, y, width, height int) []Item {
	inverted := []Item{{Shape: "rect", X: x, Y: y, W: width, H: height, Fill: inkBlack}}
	for _, item := range items {
		inverted = append(inverted, invertItem(item))
	}
	return inverted
}

func invertItem(item Item) Item {
	switch item.Shape {
	case "rect", "ellipse":
		if item.Fill != noFill {
			item.Fill = paperFull - item.Fill
		}
		item.Ink = paperFull - item.Ink
	case "text":
		item.Ink = paperFull - item.Ink
	case "icon":
		if item.Icon != nil {
			item.Icon, item.Ink = whiteIcon(item.Icon), inkBlack
		}
	}
	return item
}

// dimmedItems is the disabled look: every ink darker than ink.dim becomes ink.dim.
func dimmedItems(items []Item) []Item {
	for index := range items {
		item := &items[index]
		switch item.Shape {
		case "rect", "ellipse":
			if item.Fill != noFill && item.Fill < inkDim {
				item.Fill = inkDim
			}
			if item.Border > 0 && item.Ink < inkDim {
				item.Ink = inkDim
			}
		case "text", "icon":
			if item.Ink < inkDim {
				item.Ink = inkDim
			}
		}
	}
	return items
}
