#ifndef COPPERUI_DIVIDER_H
#define COPPERUI_DIVIDER_H

#include "copperui/geometry.h"
#include "copperui/tokens.h"

// The two kinds of line between blocks (docs/design-system.md 2.4 and 2.5): a row divider is
// 2 px in line.row (Nickel's #666666) and never black, so lists do not read as tables; the rule
// under a section label is 2 px black and the only black separator.

namespace copperui {
namespace v1 {

enum DividerKind { DIVIDER_ROW, DIVIDER_SECTION };

struct Divider {
    Rect rect; // full content width, STROKE_RULE high
    int ink;   // grey level 0..255
};

// The divider whose top edge is at y (the last STROKE_RULE pixels of its row or label block).
Divider divider(DividerKind kind, int y);

} // namespace v1
} // namespace copperui

#endif
