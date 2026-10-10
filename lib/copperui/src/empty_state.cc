#include "copperui/empty_state.h"

namespace copperui {
namespace v1 {

namespace {

Rect rectAt(int x, int y, int width, int height)
{
    Rect rect;
    rect.x = x;
    rect.y = y;
    rect.width = width;
    rect.height = height;
    return rect;
}

// Places a part of the given size at *y and moves *y below it; an absent part (size 0) takes no room.
Rect stackPart(int x, int width, int partHeight, int *y)
{
    Rect rect = rectAt(x, *y, width, partHeight);
    if (partHeight > 0) {
        *y += partHeight + RHYTHM;
    }
    return rect;
}

} // namespace

EmptyStateGeometry emptyStateGeometry(const EmptyStateContent &content, int top, int boxLeft, int boxWidth)
{
    EmptyStateGeometry geometry;
    int y = top + EMPTY_SPACER;
    geometry.icon = stackPart(boxLeft + (boxWidth - ICON_GRID) / 2, ICON_GRID, content.hasIcon ? ICON_GRID : 0, &y);
    geometry.heading = stackPart(boxLeft, boxWidth, content.headingLineCount * TYPE_HEADING_LINE, &y);
    geometry.caption = stackPart(boxLeft, boxWidth, content.captionLineCount * TYPE_CAPTION_LINE, &y);
    int buttonHeight = content.buttonLineCount >= 2 ? BUTTON_TALL_HEIGHT : BUTTON_HEIGHT;
    geometry.button = stackPart(boxLeft, boxWidth, content.hasButton ? buttonHeight : 0, &y);
    geometry.height = y - RHYTHM - top;
    return geometry;
}

} // namespace v1
} // namespace copperui
