#ifndef COPPERUI_EMPTY_STATE_H
#define COPPERUI_EMPTY_STATE_H

#include "copperui/geometry.h"
#include "copperui/tokens.h"

// The empty state (docs/design-system.md section 4.18): centred, an optional 96 px icon, the serif
// heading, a muted caption and at most one primary button. The caller measures the text and says
// how many lines each part takes; this unit only stacks the parts.
//
// Rules (the Go preview mirrors them):
//  1. The stack starts EMPTY_SPACER below top.
//  2. Every part that exists is followed by RHYTHM, except the last one.
//  3. Icon: ICON_GRID square, centred in the box. Heading: headingLineCount x TYPE_HEADING_LINE.
//     Caption: captionLineCount x TYPE_CAPTION_LINE. Both span the box width.
//  4. Button: the box width, BUTTON_HEIGHT for one label line, BUTTON_TALL_HEIGHT for two or more.
//  5. A part that does not exist has a zero-size rectangle at the position where it would start.
//  6. height is the distance from top to the bottom of the last part.

namespace copperui {
namespace v1 {

struct EmptyStateContent {
    bool hasIcon;
    int headingLineCount;
    int captionLineCount; // 0: no caption
    bool hasButton;
    int buttonLineCount;
};

struct EmptyStateGeometry {
    Rect icon;
    Rect heading;
    Rect caption;
    Rect button;
    int height;
};

EmptyStateGeometry emptyStateGeometry(const EmptyStateContent &content, int top, int boxLeft, int boxWidth);

} // namespace v1
} // namespace copperui

#endif
