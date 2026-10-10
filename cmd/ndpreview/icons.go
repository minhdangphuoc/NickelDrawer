package main

// Header icon actions and the missing-picture glyph, drawn from plain rectangles and ellipses
// so the preview needs no icon font (the mod paints the same shapes with QPainter).

func solidRect(x, y, w, h, level int) Item {
	return Item{Shape: "rect", X: x, Y: y, W: w, H: h, Fill: level}
}

func ringItem(x, y, size, thickness, level int) Item {
	return Item{Shape: "ellipse", X: x, Y: y, W: size, H: size, Fill: noFill, Border: thickness, Ink: level}
}

// headerIconItems draws the icon centred in a headerIconTarget square at (x, y) and the tap
// target over it.
func headerIconItems(icon string, x, y int, event, label string) []Item {
	glyphX, glyphY := x+(headerIconTarget-headerIconGlyph)/2, y+(headerIconTarget-headerIconGlyph)/2
	items := iconShapes(icon, glyphX, glyphY)
	return append(items, Item{Shape: "rect", X: x, Y: y, W: headerIconTarget, H: headerIconTarget, Fill: noFill, Event: event, Label: label})
}

func iconShapes(icon string, x, y int) []Item {
	switch icon {
	case "import":
		return trayDownShapes(x, y)
	case "settings":
		return gearShapes(x, y)
	case "refresh":
		return []Item{ringItem(x+6, y+6, 52, 6, inkBlack), solidRect(x+36, y, 24, 6, inkBlack), solidRect(x+54, y, 6, 24, inkBlack)}
	case "add":
		return []Item{solidRect(x+29, y+4, 6, 56, inkBlack), solidRect(x+4, y+29, 56, 6, inkBlack)}
	}
	return nil
}

// trayDownShapes is an arrow pointing down into an open tray.
func trayDownShapes(x, y int) []Item {
	items := []Item{
		solidRect(x, y+38, 6, 26, inkBlack), solidRect(x+58, y+38, 6, 26, inkBlack), solidRect(x, y+58, 64, 6, inkBlack),
		solidRect(x+29, y, 6, 34, inkBlack),
	}
	for step, width := range []int{34, 26, 18, 10} { // arrow head, widest at the top
		items = append(items, solidRect(x+32-width/2, y+24+step*5, width, 5, inkBlack))
	}
	return items
}

// gearShapes is a ring with eight square teeth (four straight, four diagonal) and a hollow hub.
func gearShapes(x, y int) []Item {
	const tooth, center = 12, 32
	items := []Item{ringItem(x+12, y+12, 40, 10, inkBlack)}
	straight := [][2]int{{0, -26}, {0, 26}, {-26, 0}, {26, 0}}
	diagonal := [][2]int{{-18, -18}, {18, -18}, {-18, 18}, {18, 18}}
	for _, offset := range append(straight, diagonal...) {
		items = append(items, solidRect(x+center+offset[0]-tooth/2, y+center+offset[1]-tooth/2, tooth, tooth, inkBlack))
	}
	return items
}

// pictureGlyphItems is a small outline picture (frame, sun, hill) centred in the box.
func pictureGlyphItems(centerX, centerY, level int) []Item {
	const width, height = 96, 72
	x, y := centerX-width/2, centerY-height/2
	return []Item{
		{Shape: "rect", X: x, Y: y, W: width, H: height, Fill: noFill, Border: 4, Ink: level},
		{Shape: "ellipse", X: x + 16, Y: y + 14, W: 16, H: 16, Fill: level, Ink: level},
		solidRect(x+14, y+height-24, width-28, 6, level),
		solidRect(x+34, y+height-36, 30, 6, level),
	}
}
