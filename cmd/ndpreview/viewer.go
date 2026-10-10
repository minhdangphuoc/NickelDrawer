package main

import (
	"fmt"
	"image"

	"koboapps/internal/uikit"
)

const (
	chromeShown  = "shown"
	chromeHidden = "hidden"
)

// soleViewer returns the viewer of a screen whose only body component is a viewer.
func soleViewer(screen *uikit.Screen) (uikit.Component, bool) {
	if len(screen.Body) == 1 && screen.Body[0].Kind == "viewer" {
		return screen.Body[0], true
	}
	return uikit.Component{}, false
}

// shownIndex is the tile the viewer shows: the screen value when it names a tile, else the first.
func (rc renderContext) shownIndex(tiles []pictureTile) int {
	if rc.value != nil {
		for index, tile := range tiles {
			if tile.id == *rc.value {
				return index
			}
		}
	}
	return 0
}

// layoutViewer draws the picture viewer. With the chrome shown: header ("n / N", status when the
// picture is current), the picture between header and bottom bar, and the bar with the secondary
// and primary buttons. Hidden (full view): the picture filling the whole area and the back button
// that shows the chrome again.
func (rc renderContext) layoutViewer(screen *uikit.Screen, viewer uikit.Component, overlay Overlay) Page {
	tiles := rc.pictureTiles(viewer)
	shownIndex := rc.shownIndex(tiles)
	var items []Item
	if rc.chrome == chromeHidden {
		items = rc.viewerPicture(tiles, shownIndex, image.Rect(0, 0, screenWidth, viewHeight))
		items = append(items, rc.viewerBackButton()...)
	} else {
		items = rc.viewerChrome(screen, viewer, tiles, shownIndex)
	}
	return Page{ScreenID: screen.ID, Items: finishPage(items, rc.overlayItems(screen, overlay)), Index: 0, Count: 1}
}

func (rc renderContext) viewerPicture(tiles []pictureTile, shownIndex int, area image.Rectangle) []Item {
	if len(tiles) == 0 {
		return nil
	}
	if picture, loaded := pictureItem(tiles[shownIndex].imagePath, area); loaded {
		return []Item{picture}
	}
	return pictureGlyphItems(area.Min.X+area.Dx()/2, area.Min.Y+area.Dy()/2, inkDim)
}

// viewerBackButton is the full view's way back (mod/drawer screen_layout.cc drawViewerBackButton):
// a white disc with a thin black outline and the header's "←", where the header's arrow was.
func (rc renderContext) viewerBackButton() []Item {
	x, y := gutter+headerIconWidth/2-viewerBackTarget/2, (headerRowHeight-viewerBackTarget)/2
	inset := (viewerBackTarget - viewerBackDisc) / 2
	disc := Item{Shape: "ellipse", X: x + inset, Y: y + inset, W: viewerBackDisc, H: viewerBackDisc,
		Fill: paperFull, Border: viewerBackOutline, Ink: inkBlack}
	hit := Item{Shape: "rect", X: x, Y: y, W: viewerBackTarget, H: viewerBackTarget, Fill: noFill, Label: rc.host("back")}
	items := []Item{disc, hit}
	return append(items, iconCentred("back", headerIconWidth, x, y, viewerBackTarget, viewerBackTarget, inkBlack)...)
}

func (rc renderContext) viewerChrome(screen *uikit.Screen, viewer uikit.Component, tiles []pictureTile, shownIndex int) []Item {
	shown := *screen
	isCurrent := len(tiles) > 0 && tiles[shownIndex].isCurrent
	if len(tiles) > 0 {
		shown.Title = fmt.Sprintf("%d / %d", shownIndex+1, len(tiles))
	}
	shown.Status = nil
	if isCurrent {
		shown.Status = viewer.Prop("current_status")
	}
	barTop := viewHeight - bottomMargin - viewerBarHeight
	area := image.Rect(0, headerHeight, screenWidth, barTop-rhythm)
	items := rc.headerItems(&shown)
	items = append(items, rc.viewerPicture(tiles, shownIndex, area)...)
	id := ""
	if len(tiles) > 0 {
		id = tiles[shownIndex].id
	}
	return append(items, rc.viewerBar(viewer, id, isCurrent, barTop)...)
}

// viewerBar is the bottom row: secondary (fixed width) at the left, primary filling the rest.
func (rc renderContext) viewerBar(viewer uikit.Component, id string, isCurrent bool, y int) []Item {
	var items []Item
	primaryX := gutter
	if secondary, hasSecondary := viewer.Prop("secondary").(map[string]any); hasSecondary {
		spec := buttonSpec{label: rc.text(secondary["label"]), action: stringField(secondary, "action"), value: id,
			isEnabled: id != "", confirmPath: ""}
		if secondary["confirm"] != nil {
			spec.confirmPath = viewer.Path + ".secondary"
		}
		items = append(items, barButton(gutter, y, viewerSecondaryWide, spec)...)
		primaryX += viewerSecondaryWide + viewerBarGap
	}
	if primary, hasPrimary := viewer.Prop("primary").(map[string]any); hasPrimary {
		spec := buttonSpec{label: rc.text(primary["label"]), action: stringField(primary, "action"), value: id,
			isPrimary: true, isEnabled: id != "" && !isCurrent}
		if isCurrent {
			spec.label = firstNonEmpty(rc.text(primary["label_current"]), spec.label)
		}
		items = append(items, barButton(primaryX, y, screenWidth-gutter-primaryX, spec)...)
	}
	return items
}

// buttonSpec describes one button of the viewer's bottom bar.
type buttonSpec struct {
	label, action, value, confirmPath string
	isPrimary, isEnabled              bool
}

func barButton(x, y, width int, spec buttonSpec) []Item {
	// The bar is one button high: a long label is elided, as in the mod.
	lines := []string{elide(spec.label, width-2*rhythm, bold(bodySize, familySans), false)}
	event := ""
	if spec.action != "" && spec.confirmPath == "" {
		event = eventLine(map[string]any{"event": "action", "name": spec.action, "value": spec.value})
	}
	return buttonItems(buttonLayout{x: x, y: y, width: width, height: viewerBarHeight}, lines, spec.isPrimary, spec.isEnabled, event, spec.confirmPath, spec.label)
}
