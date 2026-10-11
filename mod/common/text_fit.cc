#include "text_fit.h"

#include <string.h>

namespace TextFit {

static const char ELLIPSIS[] = "\xE2\x80\xA6"; // U+2026
static const int ELLIPSIS_BYTES = 3;

// ---- UTF-8 ----

// Byte length of the code point that starts with lead (1 for a stray byte).
static int sequenceLength(unsigned char lead)
{
    if (lead >= 0xF0 && lead < 0xF8) {
        return 4;
    }
    if (lead >= 0xE0) {
        return lead < 0xF0 ? 3 : 1;
    }
    return lead >= 0xC0 ? 2 : 1;
}

static int stepAt(const char *text, int position, int end)
{
    int length = sequenceLength((unsigned char)text[position]);
    return position + length > end ? end - position : length;
}

static unsigned decodeAt(const char *text, int position, int length)
{
    const unsigned char *bytes = (const unsigned char *)text + position;
    switch (length) {
    case 2:
        return ((bytes[0] & 0x1Fu) << 6) | (bytes[1] & 0x3Fu);
    case 3:
        return ((bytes[0] & 0x0Fu) << 12) | ((bytes[1] & 0x3Fu) << 6) | (bytes[2] & 0x3Fu);
    case 4:
        return ((bytes[0] & 0x07u) << 18) | ((bytes[1] & 0x3Fu) << 12) | ((bytes[2] & 0x3Fu) << 6) |
               (bytes[3] & 0x3Fu);
    }
    return bytes[0];
}

// Go's unicode.IsSpace, which strings.Fields uses (minus '\n', handled as a paragraph break).
static bool isSpace(unsigned codePoint)
{
    switch (codePoint) {
    case '\t': case '\v': case '\f': case '\r': case ' ': case 0x85: case 0xA0: case 0x1680:
    case 0x2028: case 0x2029: case 0x202F: case 0x205F: case 0x3000:
        return true;
    }
    return codePoint >= 0x2000 && codePoint <= 0x200A;
}

int codePointCount(const char *utf8, int bytes)
{
    int end = bytes < 0 ? (int)strlen(utf8) : bytes;
    int count = 0;
    for (int position = 0; position < end; position += stepAt(utf8, position, end)) {
        count++;
    }
    return count;
}

// Byte offset after skipping count code points from start.
static int advanceCodePoints(const char *text, int start, int end, int count)
{
    int position = start;
    for (int i = 0; i < count && position < end; i++) {
        position += stepAt(text, position, end);
    }
    return position;
}

// ---- Output buffer ----

struct Writer {
    char *out;
    int size;
    int used;
    int lines;
};

static void append(Writer *writer, const char *bytes, int length)
{
    for (int i = 0; i < length && writer->used < writer->size - 1; i++) {
        writer->out[writer->used++] = bytes[i];
    }
    writer->out[writer->used] = '\0';
}

static void emitLine(Writer *writer, const char *bytes, int length)
{
    if (writer->lines > 0) {
        append(writer, "\n", 1);
    }
    append(writer, bytes, length);
    writer->lines++;
}

// ---- Measured text (the algorithm of cmd/ndpreview/textfit.go; vectors in lib/copperui/tests) ----

// What decides whether a run fits: the width limit, the face and the font's measuring function.
struct Fit {
    int width;
    TextRun run;
    TextMeter meter;
};

static bool fits(const Fit &fit, const char *text, int bytes)
{
    return fit.meter.measure(fit.meter.context, text, bytes, fit.run) <= fit.width;
}

// Largest count in [low, high] for which isFitting(count) holds, or low when none does: the
// same binary search as textfit.go largestFitting, so both give the same cut on any font.
template <typename Test> static int largestFitting(int low, int high, Test isFitting)
{
    while (low < high) {
        int middle = (low + high + 1) / 2;
        if (isFitting(middle)) {
            low = middle;
        } else {
            high = middle - 1;
        }
    }
    return low;
}

// The longest prefix, in code points, of a word that does not fit; at least 1, at most count - 1.
struct PrefixFits {
    const Fit *fit;
    const char *word;
    int length;
    bool operator()(int count) const
    {
        return fits(*fit, word, advanceCodePoints(word, 0, length, count));
    }
};

// The line being built: one paragraph's words joined by single spaces.
struct LineBuilder {
    char buffer[2048];
    int length;
};

// textfit.go appendLine: an empty line is never emitted.
static void flushLine(Writer *writer, LineBuilder *line)
{
    if (line->length > 0) {
        emitLine(writer, line->buffer, line->length);
    }
    line->length = 0;
}

static void setLine(LineBuilder *line, const char *bytes, int length)
{
    int kept = length < (int)sizeof(line->buffer) ? length : (int)sizeof(line->buffer) - 1;
    memcpy(line->buffer, bytes, kept);
    line->length = kept;
}

// Appends " word"; false (line unchanged) when it cannot fit the buffer at all.
static bool addWord(LineBuilder *line, const char *bytes, int length)
{
    if (line->length + 1 + length >= (int)sizeof(line->buffer)) {
        return false; // longer than any line that fits on the screen
    }
    line->buffer[line->length++] = ' ';
    memcpy(line->buffer + line->length, bytes, length);
    line->length += length;
    return true;
}

// textfit.go cutLongWord: emits the full lines of a word wider than the limit and leaves its
// last piece open as the current line.
static void cutLongWord(Writer *writer, LineBuilder *line, const Fit &fit, const char *word, int length)
{
    int codePoints = codePointCount(word, length);
    while (codePoints > 1 && !fits(fit, word, length)) {
        PrefixFits test = {&fit, word, length};
        int count = largestFitting(1, codePoints - 1, test);
        int cut = advanceCodePoints(word, 0, length, count);
        emitLine(writer, word, cut);
        word += cut;
        length -= cut;
        codePoints -= count;
    }
    setLine(line, word, length);
}

// textfit.go wrapParagraph for one word.
static void placeWord(Writer *writer, LineBuilder *line, const Fit &fit, const char *word, int length)
{
    if (codePointCount(word, length) > 1 && !fits(fit, word, length)) {
        flushLine(writer, line);
        cutLongWord(writer, line, fit, word, length);
        return;
    }
    if (line->length == 0) {
        setLine(line, word, length);
        return;
    }
    int lengthBefore = line->length;
    if (addWord(line, word, length) && fits(fit, line->buffer, line->length)) {
        return;
    }
    line->length = lengthBefore;
    flushLine(writer, line);
    setLine(line, word, length);
}

static void wrapParagraph(Writer *writer, const Fit &fit, const char *text, int start, int end)
{
    LineBuilder line;
    line.length = 0;
    int position = start;
    while (position < end) {
        int step = stepAt(text, position, end);
        if (isSpace(decodeAt(text, position, step))) {
            position += step;
            continue;
        }
        int wordStart = position;
        while (position < end) {
            step = stepAt(text, position, end);
            if (isSpace(decodeAt(text, position, step))) {
                break;
            }
            position += step;
        }
        placeWord(writer, &line, fit, text + wordStart, position - wordStart);
    }
    flushLine(writer, &line);
}

static int wrapAll(const Fit &fit, const char *utf8, char *out, int outSize)
{
    Writer writer = {out, outSize, 0, 0};
    int end = (int)strlen(utf8);
    int start = 0;
    for (int position = 0; position <= end; position++) {
        if (position == end || utf8[position] == '\n') {
            wrapParagraph(&writer, fit, utf8, start, position);
            start = position + 1;
        }
    }
    return writer.lines;
}

// ---- Eliding ----

static const int MAX_ELIDE_BYTES = 1024; // a longer text is cut here first; no UI string comes close

struct ElideSource {
    const char *text;
    int end;
    int count;
    bool isMiddle;
};

// Byte offset after the last non-space code point of text[0, end).
static int trimmedEnd(const char *text, int end)
{
    int kept = 0;
    for (int position = 0; position < end;) {
        int step = stepAt(text, position, end);
        position += step;
        if (!isSpace(decodeAt(text, position - step, step))) {
            kept = position;
        }
    }
    return kept;
}

// textfit.go elidedCandidate: kept code points as a prefix + "…", or half from each end around
// it. buffer holds MAX_ELIDE_BYTES + 8 bytes: the text, "…" and the NUL.
static int buildCandidate(const ElideSource &source, int kept, char *buffer)
{
    int used = 0;
    if (source.isMiddle) {
        int head = kept / 2;
        int tail = kept - head;
        int headBytes = advanceCodePoints(source.text, 0, source.end, head);
        int tailStart = advanceCodePoints(source.text, 0, source.end, source.count - tail);
        memcpy(buffer, source.text, headBytes);
        used = headBytes;
        memcpy(buffer + used, ELLIPSIS, ELLIPSIS_BYTES);
        used += ELLIPSIS_BYTES;
        memcpy(buffer + used, source.text + tailStart, source.end - tailStart);
        used += source.end - tailStart;
    } else {
        used = trimmedEnd(source.text, advanceCodePoints(source.text, 0, source.end, kept));
        memcpy(buffer, source.text, used);
        memcpy(buffer + used, ELLIPSIS, ELLIPSIS_BYTES);
        used += ELLIPSIS_BYTES;
    }
    buffer[used] = '\0';
    return used;
}

struct CandidateFits {
    const Fit *fit;
    const ElideSource *source;
    bool operator()(int kept) const
    {
        char buffer[MAX_ELIDE_BYTES + 8];
        int used = buildCandidate(*source, kept, buffer);
        return fits(*fit, buffer, used);
    }
};

static void copyOut(const char *bytes, int length, char *out, int outSize)
{
    Writer writer = {out, outSize, 0, 0};
    append(&writer, bytes, length);
}

static void elide(const Fit &fit, const char *utf8, bool isMiddle, char *out, int outSize)
{
    int end = (int)strlen(utf8);
    if (end > MAX_ELIDE_BYTES) {
        end = MAX_ELIDE_BYTES;
        while (end > 0 && ((unsigned char)utf8[end] & 0xC0) == 0x80) {
            end--; // never cut inside a code point
        }
    } else if (fits(fit, utf8, end)) {
        copyOut(utf8, end, out, outSize);
        return;
    }
    ElideSource source = {utf8, end, codePointCount(utf8, end), isMiddle};
    CandidateFits test = {&fit, &source};
    int kept = largestFitting(0, source.count > 0 ? source.count - 1 : 0, test);
    char buffer[MAX_ELIDE_BYTES + 8];
    int used = buildCandidate(source, kept, buffer);
    copyOut(buffer, used, out, outSize);
}

void elideText(const char *utf8, int width, const TextRun &run, const TextMeter &meter, bool isMiddle, char *out,
               int outSize)
{
    if (outSize <= 0) {
        return;
    }
    out[0] = '\0';
    if (!utf8) {
        return;
    }
    Fit fit = {width, run, meter};
    elide(fit, utf8, isMiddle, out, outSize);
}

// ---- Wrapping ----

// Joins the lines from the one at tailStart (a byte offset into out) with spaces and elides
// the result at the end: the last line of a limited wrap.
static void elideTail(const Fit &fit, char *out, int outSize, int tailStart)
{
    char tail[MAX_ELIDE_BYTES + 8];
    int length = (int)strlen(out + tailStart);
    if (length > MAX_ELIDE_BYTES) {
        length = MAX_ELIDE_BYTES;
    }
    memcpy(tail, out + tailStart, length);
    tail[length] = '\0';
    for (int i = 0; i < length; i++) {
        if (tail[i] == '\n') {
            tail[i] = ' ';
        }
    }
    elide(fit, tail, false, out + tailStart, outSize - tailStart);
}

int wrapText(const char *utf8, int width, const TextRun &run, const TextMeter &meter, int maxLines, char *out,
             int outSize)
{
    if (outSize > 0) {
        out[0] = '\0';
    }
    if (!utf8 || outSize <= 0) {
        return 0;
    }
    Fit fit = {width, run, meter};
    int lines = wrapAll(fit, utf8, out, outSize);
    if (maxLines <= 0 || lines <= maxLines) {
        return lines;
    }
    int tailStart = 0;
    for (int line = 0; line < maxLines - 1; line++) {
        while (out[tailStart] != '\n' && out[tailStart] != '\0') {
            tailStart++;
        }
        if (out[tailStart] == '\n') {
            tailStart++;
        }
    }
    elideTail(fit, out, outSize, tailStart);
    return maxLines;
}

} // namespace TextFit
