package main

import (
	"image"
	"strconv"
	"strings"

	"koboapps/internal/uikit"
)

// galleryIndex is the position of the gallery in a body, -1 when there is none.
func galleryIndex(body []uikit.Component) int {
	for index, component := range body {
		if component.Kind == "gallery" {
			return index
		}
	}
	return -1
}

// gridGeometry is the tile size and the pages of a gallery grid.
type gridGeometry struct {
	columns, rows int
	tileW, tileH  int
	left          int // x of the first column
	pages         int
}

func (g gridGeometry) perPage() int { return g.columns * g.rows }

// newGridGeometry sizes the tiles to fill the content width (same gutters as the header) at the screen's 3:4 aspect and
// reduces rows until the grid (and the pager, when there is more than one page) fits.
func newGridGeometry(columns, wantedRows, cellCount, available int) gridGeometry {
	tileW := (contentWidth - (columns-1)*galleryGutter) / columns
	leftover := contentWidth - (columns*tileW + (columns-1)*galleryGutter)
	g := gridGeometry{columns: columns, tileW: tileW, tileH: tileW * 4 / 3, left: gutter + leftover/2}
	g.rows = fittingRows(wantedRows, g.tileH, available)
	g.pages = pageCount(cellCount, g.perPage())
	if g.pages > 1 {
		g.rows = fittingRows(wantedRows, g.tileH, available-galleryPagerHeight-rhythm)
		g.pages = pageCount(cellCount, g.perPage())
	}
	return g
}

func fittingRows(wanted, tileHeight, available int) int {
	return max(1, min(wanted, (available+galleryGutter)/(tileHeight+galleryGutter)))
}

func pageCount(cells, perPage int) int { return max(1, (cells+perPage-1)/perPage) }

// galleryView is a gallery component with its state resolved.
type galleryView struct {
	component uikit.Component
	tiles     []pictureTile
	pending   int
	columns   int
	rows      int
}

func (rc renderContext) galleryView(c uikit.Component) galleryView {
	pending, _ := strconv.Atoi(strings.TrimSpace(rc.text(c.Prop("pending"))))
	return galleryView{component: c, tiles: rc.pictureTiles(c), pending: max(0, pending),
		columns: intProp(c, "columns", defaultGalleryCols), rows: intProp(c, "rows", defaultGalleryRows)}
}

func intProp(c uikit.Component, name string, fallback int) int {
	if declared, isNumber := c.Prop(name).(float64); isNumber {
		return int(declared)
	}
	return fallback
}

func (g galleryView) isEmpty() bool { return len(g.tiles) == 0 && g.pending == 0 }

// layoutGalleryScreen lays out a screen whose body holds a gallery: the components before the
// grid repeat on every grid page, the pager sits at the bottom.
func (rc renderContext) layoutGalleryScreen(screen *uikit.Screen, index, pageIndex int, overlay Overlay) Page {
	footer := rc.blocksFor(screen.Footer)
	footerHeight := blocksHeight(footer)
	bodyTop, bodyBottom := bodyBounds(footerHeight)
	before := rc.blocksFor(screen.Body[:index])
	gridTop := bodyTop + blocksHeight(before)
	if len(before) > 0 {
		gridTop += before[len(before)-1].gap
	}
	gallery := rc.galleryView(screen.Body[index])
	items := append(rc.headerItems(screen), drawBlocks(before, bodyTop)...)
	page := Page{ScreenID: screen.ID, Index: 0, Count: 1}
	if gallery.isEmpty() {
		items = append(items, drawBlocks(rc.galleryEmptyBlocks(gallery), gridTop)...)
	} else {
		var gridItems []Item
		gridItems, page.Index, page.Count = rc.gridItems(gallery, pageIndex, gridTop, bodyBottom)
		items = append(items, gridItems...)
	}
	items = append(items, drawBlocks(footer, viewHeight-bottomMargin-footerHeight)...)
	page.Items = finishPage(items, rc.overlayItems(screen, overlay))
	return page
}

// gridItems draws one page of tiles (then placeholders) and the pager; it returns the page
// shown and the number of pages.
func (rc renderContext) gridItems(gallery galleryView, pageIndex, top, bottom int) ([]Item, int, int) {
	cellCount := len(gallery.tiles) + gallery.pending
	geometry := newGridGeometry(gallery.columns, gallery.rows, cellCount, bottom-top)
	pageIndex = clamp(pageIndex, 0, geometry.pages-1)
	first := pageIndex * geometry.perPage()
	var items []Item
	for cell := first; cell < min(cellCount, first+geometry.perPage()); cell++ {
		position := cell - first
		x := geometry.left + (position%geometry.columns)*(geometry.tileW+galleryGutter)
		y := top + (position/geometry.columns)*(geometry.tileH+galleryGutter)
		box := image.Rect(x, y, x+geometry.tileW, y+geometry.tileH)
		if cell < len(gallery.tiles) {
			items = append(items, rc.tileItems(gallery, gallery.tiles[cell], box)...)
		} else {
			items = append(items, placeholderItem(box))
		}
	}
	if geometry.pages > 1 {
		items = append(items, pageBar(bottom-galleryPagerHeight, pageIndex+1, geometry.pages)...)
	}
	return items, pageIndex, geometry.pages
}

// placeholderItem is the dashed tile of a picture that is being imported.
func placeholderItem(box image.Rectangle) Item {
	return Item{Shape: "rect", X: box.Min.X, Y: box.Min.Y, W: box.Dx(), H: box.Dy(), Fill: noFill, Border: galleryDashWidth, Ink: inkDim, Dashed: true}
}

// tileItems draws one tile: border, picture (or the missing glyph), the current badge and the
// tap target.
func (rc renderContext) tileItems(gallery galleryView, tile pictureTile, box image.Rectangle) []Item {
	border := galleryBorder
	if tile.isCurrent {
		border = galleryCurrentBdr
	}
	frame := Item{Shape: "rect", X: box.Min.X, Y: box.Min.Y, W: box.Dx(), H: box.Dy(), Fill: paperFull, Border: border, Ink: inkBlack,
		Label: tile.label}
	frame.Event, frame.Action = rc.tapEvent(gallery.component, tile.id)
	items := []Item{frame}
	inner := box.Inset(galleryCurrentBdr)
	if picture, loaded := pictureItem(tile.thumbPath, inner); loaded {
		items = append(items, picture)
	} else {
		items = append(items, pictureGlyphItems(box.Min.X+box.Dx()/2, box.Min.Y+box.Dy()/2, inkDim)...)
	}
	if tile.isCurrent {
		items = append(items, badgeItems(box.Max.X-galleryBadgeInset-galleryBadgeSize, box.Min.Y+galleryBadgeInset)...)
	}
	return items
}

// badgeItems is the black circle with a white check mark of the current picture.
func badgeItems(x, y int) []Item {
	circle := Item{Shape: "ellipse", X: x, Y: y, W: galleryBadgeSize, H: galleryBadgeSize, Fill: inkBlack, Ink: inkBlack}
	items := []Item{circle}
	icon, loaded := loadIcon("check", chevronIconSize, true) // white strokes on the black disc
	if loaded {
		items = append(items, Item{Shape: "icon", X: x + (galleryBadgeSize-chevronIconSize)/2, Y: y + (galleryBadgeSize-chevronIconSize)/2,
			W: chevronIconSize, H: chevronIconSize, Fill: noFill, Icon: icon, Label: "current"})
	}
	return items
}

// tapEvent is the protocol line (and the app action) of a tap on a tile; both are empty when
// the gallery has no on_tap.
func (rc renderContext) tapEvent(c uikit.Component, id string) (event, action string) {
	tap, isObject := c.Prop("on_tap").(map[string]any)
	if !isObject {
		return "", ""
	}
	if name, isString := tap["action"].(string); isString {
		return eventLine(map[string]any{"event": "action", "name": name, "value": id}), name
	}
	target, _ := tap["push"].(string)
	return eventLine(map[string]any{"event": "screen", "name": target, "value": id}), ""
}

// galleryEmptyBlocks is the gallery's own empty state: icon, heading, text and primary button.
func (rc renderContext) galleryEmptyBlocks(gallery galleryView) []block {
	empty, _ := gallery.component.Prop("empty").(map[string]any)
	if empty == nil {
		return nil
	}
	blocks := []block{{height: galleryEmptySpacer, gap: 0, draw: func(int) []Item { return nil }},
		{height: gridIconBox, gap: rhythm, draw: func(y int) []Item {
			return pictureGlyphItems(screenWidth/2, y+gridIconBox/2, inkDim)
		}}}
	blocks = append(blocks, paragraphBlock(rc.text(empty["title"]), headingSize, headingLineH, familySerif, true, "center", inkBlack))
	if caption := rc.text(empty["text"]); caption != "" {
		blocks = append(blocks, paragraphBlock(caption, captionSize, captionLineH, familySans, false, "center", inkMuted))
	}
	if label := rc.text(empty["button"]); label != "" {
		action, _ := empty["action"].(string)
		blocks = append(blocks, rc.button(label, action, true, true, gallery.component.Path, false))
	}
	return blocks
}
