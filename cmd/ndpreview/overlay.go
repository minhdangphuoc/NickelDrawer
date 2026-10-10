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

// dialogItems draws a modal: white box, 4 px border, title, text and two buttons (cancel left).
func (rc renderContext) dialogItems(dialog DialogSpec) []Item {
	textLines := wrapText(dialog.Text, dialogWidth-2*rhythm*2, regular(bodySize, familySans))
	height := 48 + headingLineH + rhythm + len(textLines)*bodyLineH + rhythm*2 + buttonHeight + 48
	x, y := (screenWidth-dialogWidth)/2, (screenHeight-height)/2
	innerX, innerW := x+2*rhythm, dialogWidth-4*rhythm
	title := textItem(innerX, y+48, innerW, headingLineH, elide(dialog.Title, innerW, bold(headingSize, familySans), false), headingSize, familySans, "start")
	title.Bold = true
	body := Item{Shape: "text", X: innerX, Y: y + 48 + headingLineH + rhythm, W: innerW, H: len(textLines) * bodyLineH, Lines: textLines,
		Size: bodySize, LineH: bodyLineH, Family: familySans, Ink: inkBlack, Fill: noFill}
	buttonsY := y + height - 48 - buttonHeight
	half := (innerW - rhythm) / 2
	items := []Item{{Shape: "rect", X: x, Y: y, W: dialogWidth, H: height, Fill: paperFull, Border: dialogBorder}, title, body}
	items = append(items, dialogButton(innerX, buttonsY, half, dialog.Cancel, false, "", true)...)
	return append(items, dialogButton(innerX+half+rhythm, buttonsY, half, dialog.Ok, true, dialog.OkEvent, true)...)
}

func dialogButton(x, y, width int, label string, isFilled bool, event string, closes bool) []Item {
	box := Item{Shape: "rect", X: x, Y: y, W: width, H: buttonHeight, Fill: noFill, Border: buttonBorder, Event: event, Close: closes, Label: label}
	ink := inkBlack
	if isFilled {
		box.Fill, ink = inkBlack, paperFull
	}
	text := textItem(x, y+32, width, 56, elide(label, width, bold(bodySize, familySans), false), bodySize, familySans, "center")
	text.Bold, text.Ink = true, ink
	return []Item{box, text}
}
