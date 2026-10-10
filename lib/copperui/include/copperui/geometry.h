#ifndef COPPERUI_GEOMETRY_H
#define COPPERUI_GEOMETRY_H

namespace copperui {
namespace v1 {

// A rectangle in device pixels. Plain data so the pure layout code needs no Qt.
struct Rect {
    int x;
    int y;
    int width;
    int height;
};

} // namespace v1
} // namespace copperui

#endif
