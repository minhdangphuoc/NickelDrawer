#ifndef NICKELDRAWER_TEXT_METER_H
#define NICKELDRAWER_TEXT_METER_H

#include "nickel_fonts.h"
#include "screen_logic.h"

// Measures text with QFontMetrics of the font the canvas paints it with (Nickel's own
// families, nickel_fonts.h), so the layout wraps and elides where the glyphs really end.
// screen_logic.h stays free of Qt: it receives this as a TextMeter.
namespace QtTextMeter {

// The font role that draws a screen_logic family.
NickelFonts::Role roleFor(ScreenLogic::Family family);

// The font a run is painted with.
QFont fontFor(const ScreenLogic::TextRun &run);

// Valid for the life of the process; fonts are resolved on first use.
const ScreenLogic::TextMeter &meter();

} // namespace QtTextMeter

#endif
