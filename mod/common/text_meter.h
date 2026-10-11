#ifndef MODS_TEXT_METER_H
#define MODS_TEXT_METER_H

#include <QString>
#include <QStringList>

#include "nickel_fonts.h"
#include "text_fit.h"

// Measures text with QFontMetrics of the font the canvas paints it with (Nickel's own
// families, nickel_fonts.h), so the layout wraps and elides where the glyphs really end.
// text_fit.h stays free of Qt: it receives this as a TextMeter.
namespace QtTextMeter {

// The font role that draws a text_fit family.
NickelFonts::Role roleFor(TextFit::Family family);

// The font a run is painted with.
QFont fontFor(const TextFit::TextRun &run);

// Whether the font of run draws the code point itself (no fallback font). Cached per run.
bool hasGlyph(const TextFit::TextRun &run, uint codePoint);

// Valid for the life of the process; fonts are resolved on first use.
const TextFit::TextMeter &meter();

// The shared text-fit rule (text_fit.h) on Qt strings. wrapLines: at most maxLines lines
// (0: no limit), the last one elided; none for empty text. elideLine: one line, the text or its
// measured prefix (or both ends, isMiddle) and "…".
QStringList wrapLines(const QString &text, int width, const TextFit::TextRun &run, int maxLines);
QString elideLine(const QString &text, int width, const TextFit::TextRun &run, bool isMiddle = false);

} // namespace QtTextMeter

#endif
