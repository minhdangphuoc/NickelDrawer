package main

import (
	"koboapps/internal/uikit"
)

// Overlay is what is drawn above a screen: a toast or a modal dialog.
type Overlay struct {
	Toast  string
	Dialog *DialogSpec
}

// DialogSpec is a resolved confirm dialog. OkEvent is sent when the confirm button is tapped.
type DialogSpec struct {
	Title, Text, Ok, Cancel string
	OkEvent                 string
}

func (rc renderContext) overlayItems(screen *uikit.Screen, overlay Overlay) []Item {
	var items []Item
	if overlay.Dialog != nil {
		items = append(items, rc.dialogItems(*overlay.Dialog)...)
	}
	if overlay.Toast != "" {
		items = append(items, toastItems(overlay.Toast)...)
	}
	return items
}

// toastItems draws Nickel's toast: a black box with white text near the bottom.
func toastItems(message string) []Item {
	const width, height = 900, 112
	x, y := (screenWidth-width)/2, screenHeight-toastBottom-height
	box := Item{Shape: "rect", X: x, Y: y, W: width, H: height, Fill: inkBlack}
	text := textItem(x+rhythm, y+28, width-2*rhythm, 56, elide(message, width-2*rhythm, regular(rowTextSize, familySans), false), rowTextSize, familySans, "center")
	text.Ink = paperFull
	return []Item{box, text}
}

// dialogItems draws a modal on the dialog tokens (docs/design-system.md 4.9): a 1040 px box with
// a 3 px black border and 48 px padding, serif title (42 / 60), 36 px text, and two 120 px buttons
// with the cancel button LEFT and the confirm button RIGHT. It is centred on the whole screen.
func (rc renderContext) dialogItems(dialog DialogSpec) []Item {
	innerW := dialogWidth - 2*dialogPadding
	textLines := wrapText(dialog.Text, innerW, regular(rowTextSize, familySans))
	height := 2*dialogPadding + buttonHeight
	if dialog.Title != "" {
		height += dialogTitleLineH + rhythm
	}
	if len(textLines) > 0 {
		height += len(textLines)*rowLabelHeight + rhythm
	}
	x, y := (screenWidth-dialogWidth)/2, (screenHeight-height)/2
	innerX := x + dialogPadding
	items := []Item{{Shape: "rect", X: x, Y: y, W: dialogWidth, H: height, Fill: paperFull, Border: dialogBorder, Ink: inkBlack}}
	top := y + dialogPadding
	if dialog.Title != "" {
		title := textItem(innerX, top, innerW, dialogTitleLineH, elide(dialog.Title, innerW, bold(dialogTitleSize, familySerif), false), dialogTitleSize, familySerif, "start")
		title.Bold = true
		items = append(items, title)
		top += dialogTitleLineH + rhythm
	}
	if len(textLines) > 0 {
		items = append(items, Item{Shape: "text", X: innerX, Y: top, W: innerW, H: len(textLines) * rowLabelHeight, Lines: textLines,
			Size: rowTextSize, LineH: rowLabelHeight, Family: familySans, Ink: inkBlack, Fill: noFill})
	}
	buttonsY := y + height - dialogPadding - buttonHeight
	half := (innerW - rhythm) / 2
	items = append(items, dialogButton(innerX, buttonsY, half, dialog.Cancel, false, "", true)...)
	return append(items, dialogButton(innerX+half+rhythm, buttonsY, half, dialog.Ok, true, dialog.OkEvent, true)...)
}

func dialogButton(x, y, width int, label string, isFilled bool, event string, closes bool) []Item {
	box := Item{Shape: "rect", X: x, Y: y, W: width, H: buttonHeight, Fill: noFill, Border: buttonBorder, Ink: inkBlack, Event: event, Close: closes, Label: label}
	ink := inkBlack
	if isFilled {
		box.Fill, ink = inkBlack, paperFull
	}
	text := textItem(x, y+(buttonHeight-rowLabelHeight)/2, width, rowLabelHeight, elide(label, width-2*rhythm, bold(rowTextSize, familySans), false), rowTextSize, familySans, "center")
	text.Bold, text.Ink = true, ink
	return []Item{box, text}
}

// menuItems draws Nickel's popup menu look (A11): 760 px wide, 3 px black border, a bold title
// row, 120 px rows of 34 px text separated by 1 px dividers in the row line grey. The box is
// centred on the point (centreX, centreY).
func menuItems(title string, rows []string, centreX, centreY int) []Item {
	height := 2*dialogBorder + targetMin*(1+len(rows)) + hairline*len(rows)
	x, y := centreX-menuWidth/2, centreY-height/2
	items := []Item{{Shape: "rect", X: x, Y: y, W: menuWidth, H: height, Fill: paperFull, Border: dialogBorder, Ink: inkBlack}}
	innerX, innerW := x+dialogBorder+menuSide, menuWidth-2*dialogBorder-2*menuSide
	rowTop := y + dialogBorder
	label := func(text string, isBold bool) Item {
		item := textItem(innerX, rowTop+(targetMin-copperuiMenuLine)/2, innerW, copperuiMenuLine,
			elide(text, innerW, textFace{size: menuTextSize, family: familySans, isBold: isBold}, false), menuTextSize, familySans, "start")
		item.Bold = isBold
		return item
	}
	items = append(items, label(title, true))
	rowTop += targetMin
	for _, row := range rows {
		items = append(items, Item{Shape: "rect", X: x + dialogBorder, Y: rowTop, W: menuWidth - 2*dialogBorder, H: hairline, Fill: lineRow})
		rowTop += hairline
		items = append(items, label(row, false))
		rowTop += targetMin
	}
	return items
}
