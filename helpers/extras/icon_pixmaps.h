#ifndef EXTRAS_ICON_PIXMAPS_H
#define EXTRAS_ICON_PIXMAPS_H

#include <QPixmap>

// The button's three images, all the same square size so the row never shifts:
//   normal   the Tabler "screenshot" outline (black strokes, transparent background);
//   pressed  Nickel's "_hit" look: the glyph in white on a black disc as wide as the image;
//   blank    fully transparent, shown for the moment of the capture.
// The glyph is the installed Tabler PNG, rendered by cmd/iconsgen on the exact canvas, stroke and
// margins of Nickel's brightness icon, so it is drawn 1:1 here and never scaled when the sizes agree.
// The disc is plain geometry, like PressArea::HIGHLIGHT_CIRCLE in the drawer.
namespace IconPixmaps {

struct Set {
    QPixmap normal;
    QPixmap pressed;
    QPixmap blank;

    bool isUsable() const { return !normal.isNull(); }
};

// Loads <helper folder>/icons/screenshot.png. nickelSide is the side of Nickel's own icon (0: unknown);
// the images are made that big, scaling the PNG only when its side differs. An unusable (empty) set when
// the PNG is missing or unreadable (logged).
Set load(int nickelSide);

} // namespace IconPixmaps

#endif
