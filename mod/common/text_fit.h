#ifndef MODS_TEXT_FIT_H
#define MODS_TEXT_FIT_H

// The shared text-fit rule (cmd/ndpreview/textfit.go; vectors in lib/copperui/tests/text-vectors.json):
// the same wrap and elide on the Mac preview and on the device, for every mod. No Qt, no C++
// standard library, so the host tests build it with the host compiler.

namespace TextFit {

enum Family { FAMILY_SANS, FAMILY_SERIF, FAMILY_MONO };

// ---- Text (cmd/ndpreview/textfit.go; vectors in lib/copperui/tests/text-vectors.json) ----
//
// Widths are measured, never estimated: the caller passes a TextMeter that returns the pixel
// width of a UTF-8 run in the real font (QFontMetrics in the mod), so this unit stays free of
// Qt and tests can use a deterministic fake. Rules: paragraphs split at '\n'; words split at
// Unicode spaces and are joined by one space; a line is filled greedily; a word wider than the
// line is cut into the longest pieces that fit (at least one code point); with maxLines the
// rest is joined by spaces into the last line and elided at the end; an elided line is the
// longest prefix (or prefix + suffix in the middle) that fits with "…", found by binary search.

// What decides the width of a run of text.
struct TextRun {
    int size;
    Family family;
    bool isBold;
};

// Pixel width of the UTF-8 run [utf8, utf8 + byteCount) drawn on one line in run.
typedef int (*MeasureFunction)(const void *context, const char *utf8, int byteCount, const TextRun &run);

struct TextMeter {
    MeasureFunction measure;
    const void *context;
};

// Unicode code points in UTF-8 text (bytes < 0: up to the terminating NUL).
int codePointCount(const char *utf8, int bytes);

// Wraps UTF-8 text into at most maxLines lines (0: no limit; the last line is elided when the
// text is longer). Writes the lines into out separated by '\n' (always NUL-terminated,
// truncated to outSize) and returns the number of lines. out needs at most 2 * length + 2 bytes.
int wrapText(const char *utf8, int width, const TextRun &run, const TextMeter &meter, int maxLines, char *out,
             int outSize);

// One line: unchanged when it fits, else cut at the end or in the middle with "…". Writes
// NUL-terminated UTF-8 into out (length + 4 bytes is enough).
void elideText(const char *utf8, int width, const TextRun &run, const TextMeter &meter, bool isMiddle, char *out,
               int outSize);

} // namespace TextFit

#endif
