#ifndef NICKELDRAWER_NICKEL_FONTS_H
#define NICKELDRAWER_NICKEL_FONTS_H

#include <QFont>
#include <QString>

// Font families for everything the mods draw, taken from Nickel itself.
// Shared by mod/drawer-proto and mod/drawer.
//
// Generic families ("serif", "sans-serif", "monospace") are never used: they
// resolve through the system fontconfig default, which has no Vietnamese, so
// "Ứng dụng" rendered as "□ng d□ng" on the device. Nickel's UI fonts do have
// it once kobo-tieng-viet replaced their files (Georgia, Avenir Next,
// Rakuten Serif, Rakuten Sans).
//
// Resolved once, on first use (Nickel's UI exists by then): the families of
// Nickel's live labels and buttons (their QSS is applied, so this follows the
// firmware), split into serif and sans by name; then known Nickel family names
// that are installed; then the first known name anyway. The choice and a
// self-check that 'Ứ' (U+1EE8) and 'ụ' (U+1EE5) are in each font go to the
// debug log, as a warning when a glyph is missing.
namespace NickelFonts {

enum Role {
    ROLE_TITLE, // serif, bold: page and dialog titles
    ROLE_BODY,  // sans: rows, captions, buttons
    ROLE_MONO,  // codes and addresses; Nickel's sans when no mono font exists
};

QString family(Role role);

// A pixel-sized font of the role's family (for QPainter text).
QFont font(Role role, int pixelSize, bool isBold);

// A style sheet with the placeholders @TITLE_FAMILY@, @BODY_FAMILY@ and
// @MONO_FAMILY@ replaced by the quoted family names, e.g.
// "#title { font-family: @TITLE_FAMILY@; }".
QString styleSheet(const char *templateText);

} // namespace NickelFonts

#endif
