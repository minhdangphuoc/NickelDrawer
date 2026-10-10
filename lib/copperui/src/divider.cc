#include "copperui/divider.h"

namespace copperui {
namespace v1 {

Divider divider(DividerKind kind, int y)
{
    Divider line;
    line.rect.x = GUTTER;
    line.rect.y = y;
    line.rect.width = CONTENT_WIDTH;
    line.rect.height = STROKE_RULE;
    line.ink = kind == DIVIDER_SECTION ? LINE_SECTION : LINE_ROW;
    return line;
}

} // namespace v1
} // namespace copperui
