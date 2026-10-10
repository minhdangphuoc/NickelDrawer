#ifndef COPPERUI_PAGER_H
#define COPPERUI_PAGER_H

#include "copperui/geometry.h"
#include "copperui/tokens.h"

// The pager (docs/design-system.md section 4.19): a "1 / 3" line with a chevron on each side,
// 24 px above the footer, on every screen that has more than one page. The two halves of the
// line are the touch targets. The chevrons are Tabler icons (icon.chevron), never text glyphs.

namespace copperui {
namespace v1 {

// A single page shows no pager at all, on app screens and in the drawer.
bool pagerIsShown(int pageCount);

struct PagerGeometry {
    Rect previousTarget; // left half, PAGER_TARGET_WIDTH x PAGER_TARGET_HEIGHT
    Rect nextTarget;     // right half
    Rect previousArrow;  // ICON_CHEVRON square, drawn 1:1
    Rect nextArrow;
    Rect label;          // the "1 / 3" text, centred between the arrows
};

// The pager whose line starts at top, centred on the page (content area from GUTTER).
PagerGeometry pagerGeometry(int top);

// Same for a pager that sits in a box of boxWidth px whose left edge is boxLeft: the drawer's
// footer, where the targets are the arrow buttons only.
PagerGeometry pagerGeometryIn(int boxLeft, int boxWidth, int top);

} // namespace v1
} // namespace copperui

#endif
