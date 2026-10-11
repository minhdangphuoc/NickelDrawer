#include "icon_metrics.h"

namespace IconMetrics {

static const int QT_ALIGN_TOP = 0x20;
static const int QT_ALIGN_BOTTOM = 0x40;

VerticalAlignment verticalAlignmentOf(int qtAlignmentFlags)
{
    if (qtAlignmentFlags & QT_ALIGN_BOTTOM) {
        return ALIGN_BOTTOM;
    }
    if (qtAlignmentFlags & QT_ALIGN_TOP) {
        return ALIGN_TOP;
    }
    return ALIGN_CENTRE;
}

int pixmapTop(int labelHeight, int pixmapHeight, VerticalAlignment alignment)
{
    switch (alignment) {
    case ALIGN_TOP:
        return 0;
    case ALIGN_BOTTOM:
        return labelHeight - pixmapHeight;
    case ALIGN_CENTRE:
        break;
    }
    return (labelHeight - pixmapHeight) / 2;
}

int canvasSide(int nickelSide, int imageSide)
{
    return nickelSide > 0 ? nickelSide : imageSide;
}

} // namespace IconMetrics
