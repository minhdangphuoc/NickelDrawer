#ifndef STATUSBARSHOT_ICON_PIXMAPS_H
#define STATUSBARSHOT_ICON_PIXMAPS_H

#include <QPixmap>

// The button's three images, all the same square size so the row never shifts:
//   normal   the Tabler "screenshot" outline (black, transparent background);
//   pressed  Nickel's "_hit" look: the glyph in white on a black disc;
//   blank    fully transparent, shown for the moment of the capture.
// The glyph is the installed Tabler PNG (cmd/iconsgen renders it; never drawn here), the disc is
// plain geometry, like PressArea::HIGHLIGHT_CIRCLE in the drawer.
namespace IconPixmaps {

struct Set {
    QPixmap normal;
    QPixmap pressed;
    QPixmap blank;

    bool isUsable() const { return !normal.isNull(); }
};

// Loads <helper folder>/icons/screenshot-64.png and builds the set at size x size pixels. An unusable
// (empty) set when the PNG is missing or unreadable (logged).
Set load(int size);

} // namespace IconPixmaps

#endif
