#include "copperui/pager.h"

namespace copperui {
namespace v1 {

bool pagerIsShown(int pageCount)
{
    return pageCount > 1;
}

PagerGeometry pagerGeometryIn(int boxLeft, int boxWidth, int top)
{
    int centre = boxLeft + boxWidth / 2;
    int half = boxWidth / 2;
    int arrowTop = top + (PAGER_HEIGHT - ICON_CHEVRON) / 2;
    PagerGeometry geometry;
    geometry.label.x = centre - PAGER_LABEL_WIDTH / 2;
    geometry.label.y = top;
    geometry.label.width = PAGER_LABEL_WIDTH;
    geometry.label.height = PAGER_HEIGHT;
    geometry.previousArrow.x = geometry.label.x - PAGER_ARROW_GAP - ICON_CHEVRON;
    geometry.previousArrow.y = arrowTop;
    geometry.previousArrow.width = geometry.previousArrow.height = ICON_CHEVRON;
    geometry.nextArrow.x = geometry.label.x + PAGER_LABEL_WIDTH + PAGER_ARROW_GAP;
    geometry.nextArrow.y = arrowTop;
    geometry.nextArrow.width = geometry.nextArrow.height = ICON_CHEVRON;
    geometry.previousTarget.x = boxLeft;
    geometry.previousTarget.y = top;
    geometry.previousTarget.width = half;
    geometry.previousTarget.height = PAGER_TARGET_HEIGHT;
    geometry.nextTarget = geometry.previousTarget;
    geometry.nextTarget.x = boxLeft + half;
    geometry.nextTarget.width = boxWidth - half;
    return geometry;
}

PagerGeometry pagerGeometry(int top)
{
    return pagerGeometryIn(GUTTER, CONTENT_WIDTH, top);
}

} // namespace v1
} // namespace copperui
