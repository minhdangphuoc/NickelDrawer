#ifndef NICKELDRAWER_TEXT_METER_H
#define NICKELDRAWER_TEXT_METER_H

#include <QString>
#include <QStringList>

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

// The shared text-fit rule (screen_logic.h) on Qt strings. wrapLines: at most maxLines lines
// (0: no limit), the last one elided; none for empty text. elideLine: one line, the text or its
// measured prefix (or both ends, isMiddle) and "…".
QStringList wrapLines(const QString &text, int width, const ScreenLogic::TextRun &run, int maxLines);
QString elideLine(const QString &text, int width, const ScreenLogic::TextRun &run, bool isMiddle = false);

} // namespace QtTextMeter

#endif
