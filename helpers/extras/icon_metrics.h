#ifndef EXTRAS_ICON_METRICS_H
#define EXTRAS_ICON_METRICS_H

// The few numbers behind placing the status-bar button like Nickel's own icons. No Qt, no libnickel,
// no C++ standard library (host-tested).
//
// Nickel adds every status-bar icon with addWidget(widget, 0): no layout alignment. Each one is a
// QLabel whose pixmap is aligned by the label's own alignment (0x44 = bottom, horizontal centre) inside
// a label that the layout stretches to the bar's height. A button of a fixed, smaller size sits at the
// top of that cell instead, which is how the first version ended up higher than its neighbours.

namespace IconMetrics {

enum VerticalAlignment { ALIGN_TOP, ALIGN_CENTRE, ALIGN_BOTTOM };

// The vertical part of a Qt::Alignment value (Qt 5.2: AlignTop 0x20, AlignBottom 0x40, AlignVCenter
// 0x80). QLabel centres vertically when no vertical flag is set.
VerticalAlignment verticalAlignmentOf(int qtAlignmentFlags);

// Offset of the top of a pixmap drawn in a label of labelHeight, from the label's top edge. A pixmap
// taller than the label is cut at the top and bottom alike: the offset may be negative.
int pixmapTop(int labelHeight, int pixmapHeight, VerticalAlignment alignment);

// The side of the canvas the button's images are drawn at: Nickel's icon side when it is known (so
// the button is exactly as big as its neighbours), else the image's own side.
int canvasSide(int nickelSide, int imageSide);

} // namespace IconMetrics

#endif
