package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"koboapps/internal/uikit"
)

func eventLine(fields map[string]any) string {
	data, err := json.Marshal(fields)
	if err != nil {
		return ""
	}
	return string(data)
}

func actionEvent(name string) string {
	return eventLine(map[string]any{"event": "action", "name": name})
}

func (rc renderContext) blocksFor(components []uikit.Component) []block {
	var blocks []block
	for _, component := range components {
		blocks = append(blocks, rc.componentBlocks(component)...)
	}
	return blocks
}

func (rc renderContext) componentBlocks(c uikit.Component) []block {
	switch c.Kind {
	case "text":
		return rc.textBlocks(c)
	case "section":
		return []block{rc.sectionBlock(rc.text(c.Main()))}
	case "kv":
		return rc.kvBlocks(c)
	case "row":
		return []block{rc.rowBlock(c)}
	case "toggle":
		return []block{rc.toggleBlock(c)}
	case "radio":
		return rc.radioBlocks(c)
	case "field":
		return rc.fieldBlocks(c)
	case "bigcode":
		return []block{rc.bigCodeBlock(rc.text(c.Main()), c)}
	case "qr":
		return rc.qrBlocks(rc.text(c.Main()), rc.text(c.Prop("caption")))
	case "pair":
		return rc.pairBlocks(c)
	case "steps":
		return rc.stepsBlocks(c)
	case "notice":
		if rc.text(c.Main()) == "" {
			return nil // a notice bound to an empty variable is hidden
		}
		return []block{rc.noticeBlock(c)}
	case "button":
		return []block{rc.buttonBlock(c)}
	case "list":
		return rc.listBlocks(c)
	case "empty":
		return rc.emptyBlocks(c)
	}
	return nil
}

func textStyle(style any) (size, lineH int, bold bool) {
	switch style {
	case "caption":
		return captionSize, captionLineH, false
	case "heading":
		return headingSize, headingLineH, true
	}
	return bodySize, bodyLineH, false
}

func (rc renderContext) textBlocks(c uikit.Component) []block {
	size, lineH, bold := textStyle(c.Prop("style"))
	align, _ := c.Prop("align").(string)
	ink := inkBlack
	if c.Prop("style") == "caption" {
		ink = inkMuted
	}
	return []block{paragraphBlock(rc.text(c.Main()), size, lineH, familySans, bold, align, ink)}
}

func paragraphBlock(text string, size, lineH int, family string, bold bool, align string, ink int) block {
	lines := wrapText(text, contentWidth, size, family)
	height := len(lines) * lineH
	return block{height: height, gap: rhythm, draw: func(y int) []Item {
		return []Item{{Shape: "text", X: gutter, Y: y, W: contentWidth, H: height, Lines: lines, Size: size, LineH: lineH,
			Family: family, Bold: bold, Align: align, Ink: ink, Fill: noFill}}
	}}
}

func (rc renderContext) sectionBlock(label string) block {
	return block{height: sectionHeight, gap: 0, draw: func(y int) []Item {
		item := textItem(gutter, y+16, contentWidth, 40, strings.ToUpper(elide(label, contentWidth, 28, familySans, false)), 28, familySans, "start")
		item.Ink = inkMuted
		item.Bold = true
		return []Item{item, separator(y + sectionHeight - separatorPx)}
	}}
}

// valueRow draws "key ... value" in a row of rowHeight with a separator under it.
func valueRow(key, value, event, label string, isFileName bool) block {
	return block{height: rowHeight, gap: 0, draw: func(y int) []Item {
		keyWidth, valueWidth := 440, contentWidth-440-rhythm
		keyItem := textItem(gutter, y+24, keyWidth, 56, elide(key, keyWidth, rowTextSize, familySans, false), rowTextSize, familySans, "start")
		valueItem := textItem(gutter+keyWidth+rhythm, y+24, valueWidth, 56, elide(value, valueWidth, rowTextSize, familySans, isFileName), rowTextSize, familySans, "end")
		valueItem.Bold = true
		hit := Item{Shape: "rect", X: gutter, Y: y, W: contentWidth, H: rowHeight, Fill: noFill, Event: event, Label: label}
		items := []Item{keyItem, valueItem, separator(y + rowHeight - separatorPx)}
		if event != "" {
			items = append(items, hit)
		}
		return items
	}}
}

func (rc renderContext) kvBlocks(c uikit.Component) []block {
	rows, _ := c.Main().([]any)
	var blocks []block
	for _, raw := range rows {
		pair, _ := raw.([]any)
		if len(pair) == 2 {
			blocks = append(blocks, valueRow(rc.text(pair[0]), rc.text(pair[1]), "", "", looksLikeFileName(rc.text(pair[1]))))
		}
	}
	return withTrailingGap(blocks)
}

func looksLikeFileName(text string) bool {
	return strings.Contains(text, ".") && !strings.Contains(text, " ") && !strings.Contains(text, ":")
}

func withTrailingGap(blocks []block) []block {
	if len(blocks) > 0 {
		blocks[len(blocks)-1].gap = rhythm
	}
	return blocks
}

func (rc renderContext) rowBlock(c uikit.Component) block {
	label := rc.text(c.Main())
	detail := rc.text(c.Prop("detail"))
	action, _ := c.Prop("action").(string)
	event := ""
	if action != "" {
		event = actionEvent(action)
	}
	return block{height: tapRowHeight, gap: rhythm, draw: func(y int) []Item {
		textWidth := contentWidth - 360
		items := []Item{
			textItem(gutter, y+36, textWidth, 56, elide(label, textWidth, rowTextSize, familySans, false), rowTextSize, familySans, "start"),
			separator(y + tapRowHeight - separatorPx),
		}
		if detail != "" {
			item := textItem(screenWidth-gutter-320, y+36, 280, 56, elide(detail, 280, rowTextSize, familySans, false), rowTextSize, familySans, "end")
			item.Ink = inkMuted
			items = append(items, item)
		}
		if action != "" {
			items = append(items, textItem(screenWidth-gutter-32, y+36, 32, 56, "›", rowTextSize, familySans, "end"))
			items = append(items, Item{Shape: "rect", X: gutter, Y: y, W: contentWidth, H: tapRowHeight, Fill: noFill, Event: event, Label: label})
		}
		return items
	}}
}

func (rc renderContext) toggleBlock(c uikit.Component) block {
	label := rc.text(c.Main())
	bind, _ := c.Prop("bind").(string)
	isOn := uikit.Truthy("$"+bind, rc.vars)
	isEnabled := uikit.Truthy(c.Prop("enabled"), rc.vars)
	event := ""
	if isEnabled {
		event = eventLine(map[string]any{"event": "toggle", "bind": bind, "value": !isOn})
	}
	return block{height: rowHeight, gap: rhythm, draw: func(y int) []Item {
		word := rc.host("off")
		if isOn {
			word = rc.host("on")
		}
		textWidth := contentWidth - 360
		wordItem := textItem(screenWidth-gutter-140, y+24, 140, 56, word, rowTextSize, familySans, "end")
		wordItem.Bold = isOn
		items := []Item{
			textItem(gutter, y+24, textWidth, 56, elide(label, textWidth, rowTextSize, familySans, false), rowTextSize, familySans, "start"),
			wordItem,
			separator(y + rowHeight - separatorPx),
		}
		items = append(items, switchItems(screenWidth-gutter-140-rhythm-96, y+26, isOn)...)
		return append(items, Item{Shape: "rect", X: gutter, Y: y, W: contentWidth, H: rowHeight, Fill: noFill, Event: event, Label: label})
	}}
}

// switchItems draws a 96 x 52 switch: outline with the knob left (off) or right and filled (on).
func switchItems(x, y int, isOn bool) []Item {
	track := Item{Shape: "rect", X: x, Y: y, W: 96, H: 52, Fill: noFill, Border: 3}
	knobX, knobFill := x+6, paperFull
	if isOn {
		track.Fill = inkBlack
		knobX = x + 96 - 6 - 40
	}
	knob := Item{Shape: "ellipse", X: knobX, Y: y + 6, W: 40, H: 40, Fill: knobFill, Border: 3}
	return []Item{track, knob}
}

func (rc renderContext) radioBlocks(c uikit.Component) []block {
	bind, _ := c.Prop("bind").(string)
	selected := uikit.FormatValue(lookupVar(rc.vars, bind))
	blocks := []block{rc.sectionBlock(rc.text(c.Main()))}
	options, _ := c.Prop("options").([]any)
	for _, raw := range options {
		pair, _ := raw.([]any)
		if len(pair) != 2 {
			continue
		}
		value, _ := pair[0].(string)
		blocks = append(blocks, rc.radioOption(bind, value, rc.text(pair[1]), value == selected))
	}
	return withTrailingGap(blocks)
}

func lookupVar(vars map[string]any, path string) any {
	value, _ := uikit.Lookup(vars, path)
	return value
}

func (rc renderContext) radioOption(bind, value, label string, isSelected bool) block {
	return block{height: rowHeight, gap: 0, draw: func(y int) []Item {
		ring := Item{Shape: "ellipse", X: gutter, Y: y + 26, W: 52, H: 52, Fill: noFill, Border: 4}
		items := []Item{ring}
		if isSelected {
			items = append(items, Item{Shape: "ellipse", X: gutter + 14, Y: y + 40, W: 24, H: 24, Fill: inkBlack})
		}
		textWidth := contentWidth - 52 - rhythm
		text := textItem(gutter+52+rhythm, y+24, textWidth, 56, elide(label, textWidth, rowTextSize, familySans, false), rowTextSize, familySans, "start")
		text.Bold = isSelected
		items = append(items, text, separator(y+rowHeight-separatorPx))
		event := eventLine(map[string]any{"event": "input", "bind": bind, "value": value})
		return append(items, Item{Shape: "rect", X: gutter, Y: y, W: contentWidth, H: rowHeight, Fill: noFill, Event: event, Label: label})
	}}
}

func (rc renderContext) fieldBlocks(c uikit.Component) []block {
	label := rc.text(c.Main())
	bind, _ := c.Prop("bind").(string)
	value := uikit.FormatValue(lookupVar(rc.vars, bind))
	ink := inkBlack
	if value == "" {
		value, ink = rc.text(c.Prop("placeholder")), inkDim
	} else if c.Prop("secret") == true {
		value = strings.Repeat("•", len([]rune(value)))
	}
	caption := paragraphBlock(label, captionSize, captionLineH, familySans, false, "start", inkMuted)
	caption.gap = 8
	input := block{height: fieldHeight, gap: rhythm, draw: func(y int) []Item {
		box := Item{Shape: "rect", X: gutter, Y: y, W: contentWidth, H: fieldHeight, Fill: noFill, Border: 2}
		text := textItem(gutter+rhythm, y+30, contentWidth-2*rhythm, 60, elide(value, contentWidth-2*rhythm, rowTextSize, familySans, false), rowTextSize, familySans, "start")
		text.Ink = ink
		return []Item{box, text}
	}}
	return []block{caption, input}
}

func (rc renderContext) bigCodeBlock(code string, c uikit.Component) block {
	if code == "" {
		digits := 4
		if declared, isNumber := c.Prop("digits").(float64); isNumber {
			digits = int(declared)
		}
		code = strings.Repeat("-", digits)
	}
	return block{height: bigCodeHeight, gap: rhythm, draw: func(y int) []Item {
		item := textItem(gutter, y, contentWidth, bigCodeHeight, code, bigCodeSize, familyMono, "center")
		item.Bold = true
		return []Item{item}
	}}
}

// qrBlocks draws a placeholder with the three finder squares; it does not encode the address.
func (rc renderContext) qrBlocks(address, caption string) []block {
	blocks := []block{{height: qrSide, gap: rhythm, draw: func(y int) []Item {
		return qrPlaceholder((screenWidth-qrSide)/2, y)
	}}}
	if address != "" {
		blocks = append(blocks, singleLine(address, captionSize, familyMono, "center", inkBlack, captionLineH))
	}
	if caption != "" {
		blocks = append(blocks, paragraphBlock(caption, captionSize, captionLineH, familySans, false, "center", inkMuted))
	}
	return blocks
}

func singleLine(text string, size int, family, align string, ink, lineH int) block {
	return block{height: lineH, gap: rhythm, draw: func(y int) []Item {
		item := textItem(gutter, y, contentWidth, lineH, elide(text, contentWidth, size, family, false), size, family, align)
		item.Ink = ink
		return []Item{item}
	}}
}

func qrPlaceholder(x, y int) []Item {
	frame := Item{Shape: "rect", X: x, Y: y, W: qrSide, H: qrSide, Fill: noFill, Border: 2}
	const finder = 120
	var items = []Item{frame}
	for _, corner := range [][2]int{{x + 24, y + 24}, {x + qrSide - 24 - finder, y + 24}, {x + 24, y + qrSide - 24 - finder}} {
		items = append(items,
			Item{Shape: "rect", X: corner[0], Y: corner[1], W: finder, H: finder, Fill: inkBlack},
			Item{Shape: "rect", X: corner[0] + 20, Y: corner[1] + 20, W: finder - 40, H: finder - 40, Fill: paperFull},
			Item{Shape: "rect", X: corner[0] + 40, Y: corner[1] + 40, W: finder - 80, H: finder - 80, Fill: inkBlack})
	}
	for row := 0; row < 6; row++ {
		for col := 0; col < 6; col++ {
			if (row*7+col*3)%4 < 2 {
				items = append(items, Item{Shape: "rect", X: x + 168 + col*24*2, Y: y + 168 + row*24*2, W: 24, H: 24, Fill: inkBlack})
			}
		}
	}
	return items
}

func (rc renderContext) pairBlocks(c uikit.Component) []block {
	fields, _ := c.Main().(map[string]any)
	var blocks []block
	if title := rc.text(fields["title"]); title != "" {
		blocks = append(blocks, paragraphBlock(title, headingSize, headingLineH, familySans, true, "center", inkBlack))
	}
	blocks = append(blocks, rc.qrBlocks(rc.text(fields["url"]), "")...)
	blocks = append(blocks, rc.bigCodeBlock(rc.text(fields["code"]), uikit.Component{Props: map[string]any{}}))
	for _, caption := range []string{rc.text(fields["hint"]), rc.text(fields["expires"])} {
		if caption != "" {
			blocks = append(blocks, paragraphBlock(caption, captionSize, captionLineH, familySans, false, "center", inkMuted))
		}
	}
	return blocks
}

func (rc renderContext) stepsBlocks(c uikit.Component) []block {
	fields, _ := c.Main().(map[string]any)
	headline := fmt.Sprintf("%s %s / %s", rc.text(fields["label"]), rc.text(fields["done"]), rc.text(fields["total"]))
	blocks := []block{singleLine(headline, bodySize, familySans, "start", inkBlack, bodyLineH)}
	blocks[0].gap = 0
	detail := rc.text(fields["detail"])
	if detail != "" {
		blocks = append(blocks, block{height: captionLineH, gap: rhythm, draw: func(y int) []Item {
			item := textItem(gutter, y, contentWidth, captionLineH, elide(detail, contentWidth, captionSize, familySans, true), captionSize, familySans, "start")
			item.Ink = inkMuted
			return []Item{item}
		}})
	} else {
		blocks[0].gap = rhythm
	}
	return blocks
}

func (rc renderContext) noticeBlock(c uikit.Component) block {
	title := rc.text(c.Prop("title"))
	inner := contentWidth - 2*noticePadding - 2*noticeBorder
	lines := wrapText(rc.text(c.Main()), inner, bodySize, familySans)
	height := 2*noticePadding + len(lines)*bodyLineH
	if title != "" {
		height += bodyLineH
	}
	return block{height: height, gap: rhythm, draw: func(y int) []Item {
		items := []Item{{Shape: "rect", X: gutter, Y: y, W: contentWidth, H: height, Fill: paperGrey, Border: noticeBorder}}
		top := y + noticePadding
		if title != "" {
			heading := textItem(gutter+noticePadding, top, inner, bodyLineH, elide(title, inner, bodySize, familySans, false), bodySize, familySans, "start")
			heading.Bold = true
			items = append(items, heading)
			top += bodyLineH
		}
		return append(items, Item{Shape: "text", X: gutter + noticePadding, Y: top, W: inner, H: len(lines) * bodyLineH, Lines: lines,
			Size: bodySize, LineH: bodyLineH, Family: familySans, Ink: inkBlack, Fill: noFill})
	}}
}

func (rc renderContext) buttonBlock(c uikit.Component) block {
	label := rc.text(c.Main())
	action, _ := c.Prop("action").(string)
	primary := c.Prop("style") == "primary"
	enabled := uikit.Truthy(c.Prop("enabled"), rc.vars)
	return rc.button(label, action, primary, enabled, c.Path, c.Prop("confirm") != nil)
}

func (rc renderContext) button(label, action string, primary, enabled bool, path string, hasConfirm bool) block {
	lines := wrapText(label, contentWidth-2*rhythm, bodySize, familySans)
	height := buttonHeight
	if len(lines) > 1 {
		height = buttonTallHt
	}
	return block{height: height, gap: rhythm, draw: func(y int) []Item {
		box := Item{Shape: "rect", X: gutter, Y: y, W: contentWidth, H: height, Fill: noFill, Border: 3, Label: label}
		ink := inkBlack
		if primary {
			box.Fill, ink = inkBlack, paperFull
		}
		if !enabled {
			// Disabled: outline and label dimmed, no event (no press, taps ignored).
			box.Fill, box.Ink, ink = noFill, inkDim, inkDim
		} else if hasConfirm {
			box.Confirm = path
		} else if action != "" {
			box.Event = actionEvent(action)
		}
		text := Item{Shape: "text", X: gutter + rhythm, Y: y + (height-len(lines)*bodyLineH)/2, W: contentWidth - 2*rhythm, H: len(lines) * bodyLineH,
			Lines: lines, Size: bodySize, LineH: bodyLineH, Family: familySans, Bold: true, Align: "center", Ink: ink, Fill: noFill}
		return []Item{box, text}
	}}
}

func (rc renderContext) listBlocks(c uikit.Component) []block {
	text, _ := c.Main().(string)
	varName := strings.TrimPrefix(text, "$")
	items, _ := lookupVar(rc.vars, varName).([]any)
	if limit, isNumber := c.Prop("max").(float64); isNumber && len(items) > int(limit) {
		items = items[len(items)-int(limit):]
	}
	var blocks []block
	if title := rc.text(c.Prop("title")); title != "" {
		blocks = append(blocks, rc.sectionBlock(title))
	}
	if len(items) == 0 {
		return append(blocks, singleLineCentered(rc.text(c.Prop("empty_text")), rowHeight))
	}
	tap, _ := c.Prop("tap").(string)
	for index, item := range items {
		primary, secondary := listTexts(item)
		row := valueRow(primary, secondary, "", primary, looksLikeFileName(primary))
		if tap != "" {
			row = tappableRow(row, eventLine(map[string]any{"event": "tap", "list": varName, "index": index, "item": item}), tap)
		}
		blocks = append(blocks, row)
	}
	return withTrailingGap(blocks)
}

// listTexts reads a list item: a plain string or [primary, secondary].
// tappableRow adds a tap target over a drawn row.
func tappableRow(row block, event, action string) block {
	draw := row.draw
	row.draw = func(y int) []Item {
		hit := Item{Shape: "rect", X: gutter, Y: y, W: contentWidth, H: rowHeight, Fill: noFill, Event: event, Action: action}
		return append(draw(y), hit)
	}
	return row
}

func listTexts(item any) (primary, secondary string) {
	if pair, isPair := item.([]any); isPair {
		if len(pair) > 0 {
			primary = uikit.FormatValue(pair[0])
		}
		if len(pair) > 1 {
			secondary = uikit.FormatValue(pair[1])
		}
		return primary, secondary
	}
	return uikit.FormatValue(item), ""
}

func singleLineCentered(text string, height int) block {
	return block{height: height, gap: rhythm, draw: func(y int) []Item {
		item := textItem(gutter, y+24, contentWidth, 56, elide(text, contentWidth, rowTextSize, familySans, false), rowTextSize, familySans, "center")
		item.Ink = inkMuted
		return []Item{item}
	}}
}

func (rc renderContext) emptyBlocks(c uikit.Component) []block {
	spacer := block{height: 240, gap: 0, draw: func(int) []Item { return nil }}
	blocks := []block{spacer, paragraphBlock(rc.text(c.Main()), headingSize, headingLineH, familySans, true, "center", inkBlack)}
	if caption := rc.text(c.Prop("text")); caption != "" {
		blocks = append(blocks, paragraphBlock(caption, captionSize, captionLineH, familySans, false, "center", inkMuted))
	}
	if label := rc.text(c.Prop("button")); label != "" {
		action, _ := c.Prop("action").(string)
		blocks = append(blocks, rc.button(label, action, true, true, c.Path, false))
	}
	return blocks
}
