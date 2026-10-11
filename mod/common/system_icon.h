#ifndef NICKELDRAWER_SYSTEM_ICON_H
#define NICKELDRAWER_SYSTEM_ICON_H

#include <QPixmap>

class QPainter;
class QRect;

// The icons of both mods. We never draw icons ourselves: each one is Nickel's
// own image from its Qt resources (re/notes/system-icons.md), else a Tabler
// Icons PNG (MIT) that the mod's installer puts in the icon directory the mod
// sets with setIconDirectory(), else none, and the control shows its text
// label or is hidden. Shared by every mod (each compiles its own copy).
namespace SystemIcon {

enum Id {
    SETTINGS,    // header "settings"
    IMPORT,      // header "import"
    REFRESH,     // header "refresh"
    ADD,         // header "add"
    BACK,        // page header back, full-view back
    CLOSE,       // dialog close ✕
    CHECK_BADGE, // gallery: the current picture's badge (white check on the black disc)
    VIEW_LIST,   // drawer header list toggle
    VIEW_GRID,   // drawer header grid toggle
    PICTURE,     // gallery: picture missing, empty gallery
    APP,         // drawer grid and empty states: an app without its own icon file (96 px)
    APP_ROW,     // drawer list row: the same icon at 64 px, drawn 1:1
    CHECK,       // list rows: the "done" mark (Tabler check, 64 px)
    CHEVRON_LEFT,  // pager: previous page
    CHEVRON_RIGHT, // pager: next page; row chevron
    SWITCH_ON,   // app screens: a toggle that is on (Nickel's settings switch)
    SWITCH_OFF,  // app screens: a toggle that is off
    ID_COUNT
};

enum Source { SOURCE_NONE, SOURCE_NICKEL, SOURCE_TABLER };

struct Icon {
    QPixmap normal;  // null: no icon at all
    QPixmap pressed; // Nickel's "_hit" image (white on a black disc); null: invert for the press
    Source source;

    bool isNull() const { return normal.isNull(); }
};

// Where the generic Tabler PNGs "<directory>/<name>-<64|96>.png" live; call once from init().
// Without a directory only Nickel's own images are used. The string is copied.
void setIconDirectory(const char *directory);

// Loaded on first use (Nickel's resources are registered by then) and kept;
// the source and size of each icon are logged once. QPixmap reads a file
// whole and closes it, so nothing stays open on /mnt/onboard.
const Icon &icon(Id id);

// image centred in area at its own size (Nickel's icons are drawn 1:1, as
// Nickel draws them), scaled down only when it does not fit.
void drawCentred(QPainter *painter, const QRect &area, const QPixmap &image);

// The same image with white strokes (alpha kept), for drawing a black-stroke
// Tabler icon on black.
QPixmap whiteInk(const QPixmap &image);

} // namespace SystemIcon

#endif
