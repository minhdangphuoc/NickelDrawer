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
		if rc.text(c.Main()) == "" {
			return nil // a text bound to an empty variable is hidden
		}
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
	family := familySans
	if c.Prop("style") == "heading" {
		family = familySerif // in-page titles use Nickel's serif (design-system 2.2)
	}
	return []block{paragraphBlock(rc.text(c.Main()), size, lineH, family, bold, align, ink)}
}

func paragraphBlock(text string, size, lineH int, family string, bold bool, align string, ink int) block {
	lines := wrapText(text, contentWidth, textFace{size: size, family: family, isBold: bold})
	height := len(lines) * lineH
	return block{height: height, gap: rhythm, draw: func(y int) []Item {
		return []Item{{Shape: "text", X: gutter, Y: y, W: contentWidth, H: height, Lines: lines, Size: size, LineH: lineH,
			Family: family, Bold: bold, Align: align, Ink: ink, Fill: noFill}}
	}}
}

func (rc renderContext) sectionBlock(label string) block {
	return block{height: sectionHeight, gap: 0, keepWithNext: true, draw: func(y int) []Item {
		upper := strings.ToUpper(label) // uppercase first: capitals are wider than the lower case they replace
		item := textItem(gutter, y+sectionTopInset, contentWidth, sectionLineH, elide(upper, contentWidth, bold(sectionSize, familySans), false), sectionSize, familySans, "start")
		item.Ink = inkMuted
		item.Bold = true
		return []Item{item, sectionRule(y + sectionHeight - separatorPx)}
	}}
}

// valueRow draws "key ... value" in a row of rowHeight with a separator under it.
func valueRow(key, value, event, label string, isFileName bool) block {
	return block{height: rowHeight, gap: 0, draw: func(y int) []Item {
		keyWidth, valueWidth := keyColumn, contentWidth-keyColumn-rhythm
		keyItem := textItem(gutter, y+rowTextInset, keyWidth, rowLabelHeight, elide(key, keyWidth, regular(rowTextSize, familySans), false), rowTextSize, familySans, "start")
		valueItem := textItem(gutter+keyWidth+rhythm, y+rowTextInset, valueWidth, rowLabelHeight, elide(value, valueWidth, bold(rowTextSize, familySans), isFileName), rowTextSize, familySans, "end")
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
		if len(pair) != 2 {
			continue
		}
		key, value := rc.text(pair[0]), rc.text(pair[1])
		if key == "" && value == "" {
			continue // the drawer hides a row whose key and value are both empty
		}
		blocks = append(blocks, valueRow(key, value, "", "", looksLikeFileName(value)))
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
	event := actionEventFor(map[string]any{"action": c.Prop("action"), "push": c.Prop("push")})
	return block{height: tapRowHeight, gap: rhythm, draw: func(y int) []Item {
		textWidth := contentWidth - rowTapTextInset
		items := []Item{
			textItem(gutter, y+rowTextInsetTall, textWidth, rowLabelHeight, elide(label, textWidth, regular(rowTextSize, familySans), false), rowTextSize, familySans, "start"),
			separator(y + tapRowHeight - separatorPx),
		}
		if detail != "" {
			item := textItem(screenWidth-gutter-rowDetailRight, y+rowTextInsetTall, rowDetailWidth, rowLabelHeight, elide(detail, rowDetailWidth, regular(rowTextSize, familySans), false), rowTextSize, familySans, "end")
			item.Ink = inkMuted
			items = append(items, item)
		}
		if event != "" {
			items = append(items, iconCentred("chevron-right", chevronIconSize, screenWidth-gutter-chevronIconSize, y, chevronIconSize, tapRowHeight, inkBlack)...)
			items = append(items, Item{Shape: "rect", X: gutter, Y: y, W: contentWidth, H: tapRowHeight, Fill: noFill, Event: event, Label: label})
		}
		return items
	}}
}

func (rc renderContext) toggleBlock(c uikit.Component) block {
	label := rc.text(c.Main())
	detail := rc.text(c.Prop("detail"))
	bind, _ := c.Prop("bind").(string)
	isOn := uikit.Truthy("$"+bind, rc.vars)
	isEnabled := uikit.Truthy(c.Prop("enabled"), rc.vars)
	event := ""
	ink := inkBlack // a disabled toggle is dimmed as a whole and has no event
	if isEnabled {
		event = eventLine(map[string]any{"event": "toggle", "bind": bind, "value": !isOn})
	} else {
		ink = inkDim
	}
	height := rowHeight
	if detail != "" {
		height = radioDetailRow
	}
	return block{height: height, gap: rhythm, draw: func(y int) []Item {
		state := rc.host("off")
		if isOn {
			state = rc.host("on")
		}
		switchX := gutter + contentWidth - toggleSwitchWidth
		switchY := y + (height-toggleSwitchHeight)/2
		stateX := switchX - toggleStateGap - toggleStateWidth
		textWidth := stateX - rhythm - gutter
		labelY := y + rowTextInset
		if detail != "" {
			labelY = y + rowLabelWithDetail
		}
		labelItem := textItem(gutter, labelY, textWidth, rowLabelHeight, elide(label, textWidth, regular(rowTextSize, familySans), false), rowTextSize, familySans, "start")
		labelItem.Ink = ink
		stateItem := textItem(stateX, switchY, toggleStateWidth, toggleSwitchHeight, strings.ToUpper(state), toggleStateSize, familySans, "end")
		stateItem.Ink = ink
		items := []Item{labelItem, stateItem}
		if detail != "" {
			sub := textItem(gutter, y+rowDetailTop, textWidth, captionLineH, elide(detail, textWidth, regular(captionSize, familySans), false), captionSize, familySans, "start")
			sub.Ink = inkMuted
			if !isEnabled {
				sub.Ink = inkDim
			}
			items = append(items, sub)
		}
		items = append(items, switchItems(switchX, switchY, isOn, ink)...)
		items = append(items, separator(y+height-separatorPx))
		return append(items, Item{Shape: "rect", X: gutter, Y: y, W: contentWidth, H: height, Fill: noFill, Event: event, Label: label})
	}}
}

// switchItems approximates Nickel's switch image (94 x 56; the preview cannot ship Kobo
// images, the device draws Nickel's own switch_on / switch_off, faded when disabled): outline
// with the knob left (off), or filled with the knob right (on), in ink (dim when disabled).
func switchItems(x, y int, isOn bool, ink int) []Item {
	const border, knob, inset = buttonBorder, 44, 6
	track := Item{Shape: "rect", X: x, Y: y, W: toggleSwitchWidth, H: toggleSwitchHeight, Fill: noFill, Border: border, Ink: ink}
	knobX := x + inset
	if isOn {
		track.Fill = ink
		knobX = x + toggleSwitchWidth - inset - knob
	}
	knobItem := Item{Shape: "ellipse", X: knobX, Y: y + inset, W: knob, H: knob, Fill: paperFull, Border: border, Ink: ink}
	return []Item{track, knobItem}
}

func (rc renderContext) radioBlocks(c uikit.Component) []block {
	bind, _ := c.Prop("bind").(string)
	selected := uikit.FormatValue(lookupVar(rc.vars, bind))
	// The title is a caption above the options, not a section label (design-system 2.11).
	title := paragraphBlock(rc.text(c.Main()), captionSize, captionLineH, familySans, false, "start", inkMuted)
	title.gap, title.keepWithNext = fieldCaptionGap, true
	blocks := []block{title}
	isEnabled := uikit.Truthy(c.Prop("enabled"), rc.vars)
	options, _ := c.Prop("options").([]any)
	for _, raw := range options {
		pair, _ := raw.([]any)
		if len(pair) < 2 {
			continue
		}
		value, _ := pair[0].(string)
		detail := ""
		if len(pair) > 2 {
			detail = rc.text(pair[2])
		}
		blocks = append(blocks, rc.radioOption(bind, value, rc.text(pair[1]), detail, value == selected, isEnabled))
	}
	for index := 1; index < len(blocks)-1; index++ {
		blocks[index].keepWithNext = true // the options stay with their title; the last one ends the group
	}
	return withTrailingGap(blocks)
}

func lookupVar(vars map[string]any, path string) any {
	value, _ := uikit.Lookup(vars, path)
	return value
}

// radioOption is one row; an option with a sub-line is radioDetailRow high and shows the
// sub-line in caption style under the label.
func (rc renderContext) radioOption(bind, value, label, detail string, isSelected, isEnabled bool) block {
	height := rowHeight
	if detail != "" {
		height = radioDetailRow
	}
	return block{height: height, gap: 0, draw: func(y int) []Item {
		ink, detailInk := inkBlack, inkMuted
		if !isEnabled { // dimmed as a whole, no event
			ink, detailInk = inkDim, inkDim
		}
		ringY := y + (height-radioRingSide)/2
		items := []Item{{Shape: "ellipse", X: gutter, Y: ringY, W: radioRingSide, H: radioRingSide, Fill: noFill, Border: radioRingBorder, Ink: ink}}
		if isSelected {
			items = append(items, Item{Shape: "ellipse", X: gutter + radioDotInset, Y: ringY + radioDotInset, W: radioDotSide, H: radioDotSide, Fill: ink, Ink: ink})
		}
		textWidth := contentWidth - radioRingSide - rhythm
		textX := gutter + radioRingSide + rhythm
		labelY := y + rowTextInset
		if detail != "" {
			labelY = y + rowLabelWithDetail
		}
		labelFace := regular(rowTextSize, familySans)
		if isSelected {
			labelFace = bold(rowTextSize, familySans)
		}
		text := textItem(textX, labelY, textWidth, rowLabelHeight, elide(label, textWidth, labelFace, false), rowTextSize, familySans, "start")
		text.Bold, text.Ink = isSelected, ink
		items = append(items, text)
		if detail != "" {
			sub := textItem(textX, labelY+rowLabelHeight, textWidth, captionLineH, elide(detail, textWidth, regular(captionSize, familySans), false), captionSize, familySans, "start")
			sub.Ink = detailInk
			items = append(items, sub)
		}
		items = append(items, separator(y+height-separatorPx))
		event := ""
		if isEnabled {
			event = eventLine(map[string]any{"event": "input", "bind": bind, "value": value})
		}
		return append(items, Item{Shape: "rect", X: gutter, Y: y, W: contentWidth, H: height, Fill: noFill, Event: event, Label: label})
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
	isEnabled := uikit.Truthy(c.Prop("enabled"), rc.vars)
	if !isEnabled {
		ink = inkDim // a disabled field is dimmed as a whole and has no target
	}
	caption := paragraphBlock(label, captionSize, captionLineH, familySans, false, "start", inkMuted)
	caption.gap, caption.keepWithNext = fieldCaptionGap, true
	input := block{height: fieldHeight, gap: rhythm, draw: func(y int) []Item {
		box := Item{Shape: "rect", X: gutter, Y: y, W: contentWidth, H: fieldHeight, Fill: noFill, Border: fieldBorder, Ink: inkBlack}
		if !isEnabled {
			box.Ink = inkDim
		}
		text := textItem(gutter+rhythm, y+(fieldHeight-fieldTextHeight)/2, contentWidth-2*rhythm, fieldTextHeight, elide(value, contentWidth-2*rhythm, regular(rowTextSize, familySans), false), rowTextSize, familySans, "start")
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
		return []Item{textItem(gutter, y, contentWidth, bigCodeHeight, code, bigCodeSize, familyMono, "center")}
	}}
}

// qrBlocks draws a placeholder with the three finder squares; it does not encode the address.
func (rc renderContext) qrBlocks(address, caption string) []block {
	blocks := []block{{height: qrSide, gap: rhythm, draw: func(y int) []Item {
		return qrPlaceholder((screenWidth-qrSide)/2, y)
	}}}
	if address != "" {
		blocks[0].keepWithNext = true // the code stays with its address line
		blocks = append(blocks, singleLine(address, captionSize, familyMono, "center", inkBlack, captionLineH))
	}
	if caption != "" {
		blocks = append(blocks, paragraphBlock(caption, captionSize, captionLineH, familySans, false, "center", inkMuted))
	}
	return blocks
}

func singleLine(text string, size int, family, align string, ink, lineH int) block {
	return block{height: lineH, gap: rhythm, draw: func(y int) []Item {
		item := textItem(gutter, y, contentWidth, lineH, elide(text, contentWidth, regular(size, family), false), size, family, align)
		item.Ink = ink
		return []Item{item}
	}}
}

func qrPlaceholder(x, y int) []Item {
	frame := Item{Shape: "rect", X: x, Y: y, W: qrSide, H: qrSide, Fill: noFill, Border: qrFrameBorder}
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
		heading := paragraphBlock(title, headingSize, headingLineH, familySerif, true, "center", inkBlack)
		heading.keepWithNext = true // the title stays with the QR code
		blocks = append(blocks, heading)
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
		blocks[0].keepWithNext = true
		blocks = append(blocks, block{height: captionLineH, gap: rhythm, draw: func(y int) []Item {
			item := textItem(gutter, y, contentWidth, captionLineH, elide(detail, contentWidth, regular(captionSize, familySans), true), captionSize, familySans, "start")
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
	lines := wrapText(rc.text(c.Main()), inner, regular(bodySize, familySans))
	height := 2*noticePadding + len(lines)*bodyLineH
	if title != "" {
		height += bodyLineH
	}
	return block{height: height, gap: rhythm, draw: func(y int) []Item {
		items := []Item{{Shape: "rect", X: gutter, Y: y, W: contentWidth, H: height, Fill: paperGrey, Border: noticeBorder}}
		top := y + noticePadding
		if title != "" {
			heading := textItem(gutter+noticePadding, top, inner, bodyLineH, elide(title, inner, bold(bodySize, familySans), false), bodySize, familySans, "start")
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
	lines := wrapLimited(label, contentWidth-2*rhythm, bold(bodySize, familySans), buttonLines)
	height := buttonHeight
	if len(lines) > 1 {
		height = buttonTallHt
	}
	event, confirmPath := "", ""
	if hasConfirm {
		confirmPath = path
	} else if action != "" {
		event = actionEvent(action)
	}
	return block{height: height, gap: rhythm, draw: func(y int) []Item {
		return buttonItems(buttonLayout{x: gutter, y: y, width: contentWidth, height: height}, lines, primary, enabled, event, confirmPath, label)
	}}
}

// buttonLayout is the box of a button.
type buttonLayout struct{ x, y, width, height int }

// buttonItems draws a button: outline (filled black when primary) and the bold centred label.
// A disabled button is dimmed and has no event; a confirm path opens that dialog first.
func buttonItems(box buttonLayout, lines []string, primary, enabled bool, event, confirmPath, label string) []Item {
	outline := Item{Shape: "rect", X: box.x, Y: box.y, W: box.width, H: box.height, Fill: noFill, Border: buttonBorder, Label: label}
	ink := inkBlack
	if primary {
		outline.Fill, ink = inkBlack, paperFull
	}
	switch {
	case !enabled:
		outline.Fill, outline.Ink, ink = noFill, inkDim, inkDim
	case confirmPath != "":
		outline.Confirm = confirmPath
	default:
		outline.Event = event
	}
	text := Item{Shape: "text", X: box.x + rhythm, Y: box.y + (box.height-len(lines)*bodyLineH)/2, W: box.width - 2*rhythm, H: len(lines) * bodyLineH,
		Lines: lines, Size: bodySize, LineH: bodyLineH, Family: familySans, Bold: true, Align: "center", Ink: ink, Fill: noFill}
	return []Item{outline, text}
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
		event := ""
		if tap != "" {
			event = eventLine(map[string]any{"event": "tap", "list": varName, "index": index, "item": item})
		}
		blocks = append(blocks, listRowBlock(primary, secondary, event, tap))
	}
	return withTrailingGap(blocks)
}

const checkMark = "✓" // U+2713: a list row's secondary text that is drawn as the Tabler check icon

// listRowBlock is a row of a list (lib/copperui/src/list_row.cc): the primary text on the left
// takes the whole row less the secondary caption (a file name needs the room and is elided in the
// middle); the secondary text is a muted caption at the right edge, or the check icon.
func listRowBlock(primary, secondary, event, action string) block {
	return block{height: rowHeight, gap: 0, draw: func(y int) []Item {
		hasCheck := secondary == checkMark
		secondaryWidth := 0
		if secondary != "" && !hasCheck {
			secondaryWidth = min(textMeter.width(secondary, regular(captionSize, familySans)), listSecondaryMax)
		}
		reserved := 0
		switch {
		case hasCheck:
			reserved = chevronIconSize + rhythm
		case secondaryWidth > 0:
			reserved = secondaryWidth + rhythm
		}
		primaryWidth := contentWidth - reserved
		items := []Item{textItem(gutter, y+rowTextInset, primaryWidth, rowLabelHeight,
			elide(primary, primaryWidth, regular(rowTextSize, familySans), looksLikeFileName(primary)), rowTextSize, familySans, "start")}
		switch {
		case hasCheck:
			items = append(items, iconAt("check", chevronIconSize, gutter+contentWidth-chevronIconSize, y+(rowHeight-chevronIconSize)/2, inkBlack)...)
		case secondaryWidth > 0:
			caption := textItem(gutter+contentWidth-secondaryWidth, y+(rowHeight-captionLineH)/2, secondaryWidth, captionLineH,
				elide(secondary, secondaryWidth, regular(captionSize, familySans), false), captionSize, familySans, "end")
			caption.Ink = inkMuted
			items = append(items, caption)
		}
		items = append(items, separator(y+rowHeight-separatorPx))
		if event != "" {
			items = append(items, Item{Shape: "rect", X: gutter, Y: y, W: contentWidth, H: rowHeight, Fill: noFill, Event: event, Action: action, Label: primary})
		}
		return items
	}}
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
		item := textItem(gutter, y+24, contentWidth, 56, elide(text, contentWidth, regular(rowTextSize, familySans), false), rowTextSize, familySans, "center")
		item.Ink = inkMuted
		return []Item{item}
	}}
}

func (rc renderContext) emptyBlocks(c uikit.Component) []block {
	spacer := block{height: emptySpacerHeight, gap: 0, draw: func(int) []Item { return nil }}
	blocks := []block{spacer, paragraphBlock(rc.text(c.Main()), headingSize, headingLineH, familySerif, true, "center", inkBlack)}
	if caption := rc.text(c.Prop("text")); caption != "" {
		blocks = append(blocks, paragraphBlock(caption, captionSize, captionLineH, familySans, false, "center", inkMuted))
	}
	if label := rc.text(c.Prop("button")); label != "" {
		action, _ := c.Prop("action").(string)
		blocks = append(blocks, rc.button(label, action, true, true, c.Path, false))
	}
	return blocks
}
