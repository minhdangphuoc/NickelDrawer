#ifndef NICKELDRAWER_SCREEN_FONTS_H
#define NICKELDRAWER_SCREEN_FONTS_H

#include <QFont>

#include "screen_logic.h"

// Fonts of app screens. G2 found no libnickel API that hands out Nickel's UI
// fonts, but Nickel registers its fonts with Qt in-process, so the first
// installed family of each preference list is Nickel's own (serif titles,
// sans body, mono codes); without any of them Qt's default with a style hint.
// Sizes are pixel sizes from docs/spec/ui-kit.md.
namespace ScreenFonts {

QFont font(ScreenLogic::Family family, int pixelSize, bool isBold, bool isUnderlined);

} // namespace ScreenFonts

#endif
