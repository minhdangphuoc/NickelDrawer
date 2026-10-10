package main

import (
	"fmt"
	"image"
	"strings"

	"koboapps/internal/uikit"
)

// Item is one drawing primitive in device pixels. Both the HTML and the PNG preview draw
// the same items, so they cannot disagree about the layout.
type Item struct {
	Shape     string // "rect", "ellipse", "text", "image" or "icon"
	X, Y      int
	W, H      int
	Fill      int // grey 0..255, noFill for none
	Border    int // outline width in px, 0 for none
	Lines     []string
	Size      int
	LineH     int
	Family    string
	Bold      bool
	Underline bool
	Align     string       // start, center, end
	Ink       int          // text colour, and the border of rects and ellipses
	Event     string       // protocol line sent to the backend when tapped
	Action    string       // app.json action a tap also runs (list rows with "tap")
	Confirm   string       // component path whose confirm dialog opens first
	Close     bool         // tap closes the open dialog
	Label     string       // what the item is, for tooltips and the tap log
	Dashed    bool         // rect border drawn as dashes
	Picture   *image.Gray  // "image": already scaled to W x H
	Icon      *image.NRGBA // "icon": a Tabler PNG, composited 1:1 (black strokes, faded by Ink)
}

// block is a vertical slice of a screen. Blocks never split across pages; a list is many blocks.
type block struct {
	height int
	gap    int
	// keepWithNext keeps the block on the page of the next one (section label and its first row,
	// field caption and box, pair title and QR, radio title and options): see packPages.
	keepWithNext bool
	isTapRow     bool // a tappable "row": a run of them is joined (joinRowRuns)
	isSection    bool // a section label
	draw         func(y int) []Item
}

// Page is one laid-out page of a screen.
type Page struct {
	ScreenID string
	Items    []Item
	Index    int
	Count    int
}

// separator is a row divider: 2 px in the row line grey.
func separator(y int) Item {
	return Item{Shape: "rect", X: gutter, Y: y, W: contentWidth, H: separatorPx, Fill: lineRow}
}

// sectionRule is the black 2 px rule under a section label.
func sectionRule(y int) Item {
	return Item{Shape: "rect", X: gutter, Y: y, W: contentWidth, H: separatorPx, Fill: lineSection}
}

func textItem(x, y, w, h int, text string, size int, family, align string) Item {
	return Item{Shape: "text", X: x, Y: y, W: w, H: h, Lines: []string{text}, Size: size, LineH: h, Family: family, Align: align, Ink: inkBlack, Fill: noFill}
}

// packPageIndexes gives the page of every block, in a body of pages available px high. It is the
// port of lib/copperui/src/pack.cc packPages and is pinned by lib/copperui/tests/pack-vectors.json.
//
// A run of blocks whose all but last member keepWithNext is one group. A group goes whole onto
// the current page when it fits, else onto a fresh page, so a section label is never alone at the
// bottom of a page. A group taller than a page is packed block by block. keepWithNext on the last
// block of the body is ignored.
func packPageIndexes(blocks []block, available int) []int {
	pages := make([]int, len(blocks))
	page, used, onPage := 0, 0, 0
	for index := 0; index < len(blocks); {
		last := index
		for last < len(blocks)-1 && blocks[last].keepWithNext {
			last++
		}
		groupHeight := blocksHeight(blocks[index : last+1])
		if onPage > 0 && used+blocks[index-1].gap+groupHeight > available {
			page, used, onPage = page+1, 0, 0
		}
		isTooTall := groupHeight > available
		for position := index; position <= last; position++ {
			needed := blocks[position].height
			if onPage > 0 {
				needed += blocks[position-1].gap
			}
			if isTooTall && onPage > 0 && used+needed > available {
				page, used, onPage = page+1, 0, 0
				needed = blocks[position].height
			}
			pages[position] = page
			used += needed
			onPage++
		}
		index = last + 1
	}
	return pages
}

// packPages splits blocks into pages that each fit in available pixels.
func packPages(blocks []block, available int) [][]block {
	pageOf := packPageIndexes(blocks, available)
	pages := make([][]block, 1)
	for index, item := range blocks {
		for len(pages) <= pageOf[index] {
			pages = append(pages, nil)
		}
		pages[pageOf[index]] = append(pages[pageOf[index]], item)
	}
	return pages
}

func drawBlocks(blocks []block, top int) []Item {
	var items []Item
	y := top
	for _, item := range blocks {
		items = append(items, item.draw(y)...)
		y += item.height + item.gap
	}
	return items
}

func blocksHeight(blocks []block) int {
	total := 0
	for i, item := range blocks {
		total += item.height
		if i < len(blocks)-1 {
			total += item.gap
		}
	}
	return total
}

// bodyBounds are the first and last pixel rows of the body for a footer of footerHeight.
func bodyBounds(footerHeight int) (top, bottom int) {
	top = headerHeight + rhythm
	bottom = viewHeight - bottomMargin
	if footerHeight > 0 {
		bottom -= footerHeight + rhythm
	}
	return top, bottom
}

// layoutScreen lays out one page of a screen: header, body (paged), footer and overlay.
func layoutScreen(rc renderContext, screenID string, pageIndex int, overlay Overlay) (Page, error) {
	screen, found := rc.doc.Screens[screenID]
	if !found {
		return Page{}, fmt.Errorf("screen %q does not exist; screens: %s", screenID, strings.Join(rc.doc.ScreenIDs(), ", "))
	}
	if viewer, isViewer := soleViewer(screen); isViewer {
		return rc.layoutViewer(screen, viewer, overlay), nil
	}
	if index := galleryIndex(screen.Body); index >= 0 {
		return rc.layoutGalleryScreen(screen, index, pageIndex, overlay), nil
	}
	footer := rc.blocksFor(screen.Footer)
	footerHeight := blocksHeight(footer)
	bodyTop, bodyBottom := bodyBounds(footerHeight)
	body := rc.blocksFor(screen.Body)
	pages := packPages(body, bodyBottom-bodyTop)
	if len(pages) > 1 {
		available := bodyBottom - bodyTop - pageBarHeight - rhythm
		pages = packPages(keepSectionsTogether(body, available), available)
	}
	pageIndex = clamp(pageIndex, 0, len(pages)-1)

	items := rc.headerItems(screen)
	items = append(items, drawBlocks(pages[pageIndex], bodyTop)...)
	if len(pages) > 1 {
		items = append(items, pageBar(bodyBottom-pageBarHeight, pageIndex+1, len(pages))...)
	}
	items = append(items, drawBlocks(footer, viewHeight-bottomMargin-footerHeight)...)
	return Page{ScreenID: screenID, Items: finishPage(items, rc.overlayItems(screen, overlay)), Index: pageIndex, Count: len(pages)}, nil
}

// finishPage turns items laid out in the page area (y from the top of the view below Nickel's
// status bar, as the device lays out) into screen coordinates: everything moves down by the
// status-bar inset, a plain strip stands for the status bar above and the tab bar below, and the
// overlay (a toast or dialog, positioned on the whole screen) comes last. The strips carry no
// icons; they only show how much room the screen really has.
func finishPage(items, overlay []Item) []Item {
	result := nickelChromeItems()
	for _, item := range items {
		item.Y += insetStatusBar
		result = append(result, item)
	}
	return append(result, overlay...)
}

// nickelChromeItems are the status bar (0..insetStatusBar) and tab bar strips, each a 1 px
// dim line and a small muted caption.
func nickelChromeItems() []Item {
	tabTop := insetStatusBar + viewHeight
	caption := func(y int, text string) Item {
		item := textItem(gutter, y, contentWidth, 40, text, 28, familySans, "center")
		item.Ink = inkDim
		return item
	}
	return []Item{
		caption((insetStatusBar-40)/2, "Nickel status bar"),
		{Shape: "rect", X: 0, Y: insetStatusBar - 1, W: screenWidth, H: 1, Fill: inkDim},
		{Shape: "rect", X: 0, Y: tabTop, W: screenWidth, H: 1, Fill: inkDim},
		caption(tabTop+(tabBarHeight-40)/2, "Nickel tab bar"),
	}
}

// pageBar is the pager (lib/copperui/src/pager.cc pagerGeometry): "n / N" muted and centred with
// a Tabler chevron on each side. Callers draw it only when there is more than one page.
func pageBar(y, current, total int) []Item {
	centre := gutter + contentWidth/2
	labelX := centre - pagerLabelWidth/2
	label := textItem(labelX, y, pagerLabelWidth, pageBarHeight, fmt.Sprintf("%d / %d", current, total), captionSize, familySans, "center")
	label.Ink = inkMuted
	arrowY := y + (pageBarHeight-chevronIconSize)/2
	items := []Item{label}
	items = append(items, iconAt("chevron-left", chevronIconSize, labelX-pagerArrowGap-chevronIconSize, arrowY, inkBlack)...)
	return append(items, iconAt("chevron-right", chevronIconSize, labelX+pagerLabelWidth+pagerArrowGap, arrowY, inkBlack)...)
}

func clamp(value, low, high int) int {
	return max(low, min(value, high))
}

// renderContext carries what resolving texts needs.
type renderContext struct {
	doc     *uikit.Document
	locale  string
	vars    map[string]any
	baseDir string  // relative picture paths in the state resolve against this folder
	value   *string // the screen value (--value), e.g. the picture a viewer opens on
	chrome  string  // viewer: chromeShown (default) or chromeHidden
}

func (rc renderContext) text(raw any) string {
	return uikit.Substitute(rc.doc.Text(raw, rc.locale), rc.vars)
}

func (rc renderContext) host(key string) string { return hostText(rc.locale, key) }

// headerItems draws the header as the device shows it (mod/drawer/screen_view.cc): Nickel's
// "←", the title left-aligned next to it in the sans, the status text and the optional action
// at the right of the same row, and the thin divider under the whole row.
func (rc renderContext) headerItems(screen *uikit.Screen) []Item {
	back := Item{Shape: "rect", X: gutter, Y: 0, W: headerIconWidth, H: headerRowHeight, Fill: noFill, Label: rc.host("back")}
	icons := rc.headerIconActionItems(screen)
	rightEdge := screenWidth - gutter - len(screen.Actions)*headerIconTarget
	right := rc.headerRight(screen, rightEdge)
	titleX := gutter + headerIconWidth + rhythm
	titleWidth := rightEdge - titleX - len(right)*(headerSideWidth+rhythm)
	title := textItem(titleX, 0, titleWidth, headerRowHeight, elide(rc.text(screen.Title), titleWidth, regular(headerTitleSz, familySans), false), headerTitleSz, familySans, "start")
	divider := Item{Shape: "rect", X: gutter, Y: headerRowHeight, W: contentWidth, H: headerDivider, Fill: headerDividerInk}
	items := []Item{back}
	items = append(items, iconCentred("back", headerIconWidth, gutter, 0, headerIconWidth, headerRowHeight, inkBlack)...)
	items = append(items, title)
	items = append(items, right...)
	items = append(items, icons...)
	return append(items, divider)
}

// headerRight draws the status text and the optional action at the right of the header row,
// the action last (rightmost), each in a column of at most headerSideWidth.
func (rc renderContext) headerRight(screen *uikit.Screen, rightEdge int) []Item {
	var items []Item
	if screen.Status != nil {
		if status := rc.text(screen.Status); status != "" {
			item := textItem(0, 0, headerSideWidth, headerRowHeight, elide(status, headerSideWidth, regular(headerStatusSz, familySans), false), headerStatusSz, familySans, "end")
			item.Ink = inkMuted
			items = append(items, item)
		}
	}
	if screen.Action != nil {
		label := rc.text(screen.Action["label"])
		item := textItem(0, 0, headerSideWidth, headerRowHeight, elide(label, headerSideWidth, regular(headerStatusSz, familySans), false), headerStatusSz, familySans, "end")
		item.Underline = true
		item.Event = actionEventFor(screen.Action)
		item.Label = label
		items = append(items, item)
	}
	x := rightEdge - len(items)*headerSideWidth - (len(items)-1)*rhythm
	for index := range items {
		items[index].X = x
		x += headerSideWidth + rhythm
	}
	return items
}

// headerIconActionItems draws the icon buttons of the header at its right edge, the last one
// rightmost.
func (rc renderContext) headerIconActionItems(screen *uikit.Screen) []Item {
	var items []Item
	x := screenWidth - gutter - len(screen.Actions)*headerIconTarget
	for _, action := range screen.Actions {
		icon, _ := action["icon"].(string)
		label := rc.text(action["label"])
		item := headerIconItems(icon, x, 0, actionEventFor(action), label)
		items = append(items, item...)
		x += headerIconTarget
	}
	return items
}

func actionEventFor(action map[string]any) string {
	if name, isString := action["action"].(string); isString {
		return actionEvent(name)
	}
	if target, isString := action["push"].(string); isString {
		return eventLine(map[string]any{"event": "screen", "name": target})
	}
	return ""
}
