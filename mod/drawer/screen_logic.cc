#include "screen_logic.h"

#include <string.h>

namespace ScreenLogic {

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

// ---- Component sizes ----

int textBlockHeight(int lineCount, int lineHeightPx)
{
    return lineCount * lineHeightPx;
}

int buttonHeight(int labelLineCount)
{
    return labelLineCount > 1 ? BUTTON_TALL_HEIGHT : BUTTON_HEIGHT;
}

int noticeHeight(int textLineCount, bool hasTitle)
{
    return 2 * NOTICE_PADDING + textLineCount * BODY_LINE_HEIGHT + (hasTitle ? BODY_LINE_HEIGHT : 0);
}

int textSize(TextStyle style)
{
    switch (style) {
    case STYLE_CAPTION:
        return CAPTION_SIZE;
    case STYLE_HEADING:
        return HEADING_SIZE;
    case STYLE_BODY:
        break;
    }
    return BODY_SIZE;
}

int lineHeight(TextStyle style)
{
    switch (style) {
    case STYLE_CAPTION:
        return CAPTION_LINE_HEIGHT;
    case STYLE_HEADING:
        return HEADING_LINE_HEIGHT;
    case STYLE_BODY:
        break;
    }
    return BODY_LINE_HEIGHT;
}

bool rowDetailIsSubline(int measuredDetailWidth)
{
    return measuredDetailWidth > tokens::ROW_DETAIL_WIDTH;
}

int firstKeptItem(int itemCount, int max)
{
    if (max <= 0 || itemCount <= max) {
        return 0;
    }
    return itemCount - max;
}

bool looksLikeFileName(const char *utf8)
{
    return utf8 && strchr(utf8, '.') && !strchr(utf8, ' ') && !strchr(utf8, ':');
}

QrPlacement placeQr(int moduleCount, int side)
{
    QrPlacement placement = {0, 0};
    if (moduleCount <= 0) {
        return placement;
    }
    placement.moduleSize = side / moduleCount;
    placement.offset = (side - placement.moduleSize * moduleCount) / 2;
    return placement;
}

// ---- Gallery and viewer ----

static int clampInt(int value, int low, int high)
{
    return value < low ? low : (value > high ? high : value);
}

int galleryTileWidth(int columns)
{
    columns = clampInt(columns, 1, GALLERY_MAX_COLUMNS);
    return (CONTENT_WIDTH - (columns - 1) * GALLERY_GUTTER) / columns;
}

int galleryTileHeight(int columns)
{
    return galleryTileWidth(columns) * 4 / 3;
}

int galleryGridLeft(int columns)
{
    columns = clampInt(columns, 1, GALLERY_MAX_COLUMNS);
    int leftover = CONTENT_WIDTH - (columns * galleryTileWidth(columns) + (columns - 1) * GALLERY_GUTTER);
    return GUTTER + leftover / 2;
}

static int rowsThatFit(int columns, int wantedRows, int availableHeight)
{
    int tileHeight = galleryTileHeight(columns);
    int rows = wantedRows;
    while (rows > 1 && rows * tileHeight + (rows - 1) * GALLERY_GUTTER > availableHeight) {
        rows--;
    }
    return rows;
}

GalleryFrame galleryFrame(int columns, int wantedRows, int slotCount, int availableHeight)
{
    GalleryFrame frame;
    frame.columns = clampInt(columns, 1, GALLERY_MAX_COLUMNS);
    wantedRows = clampInt(wantedRows, 1, GALLERY_MAX_ROWS);
    frame.rows = rowsThatFit(frame.columns, wantedRows, availableHeight);
    frame.perPage = frame.columns * frame.rows;
    if (slotCount > frame.perPage) {
        frame.rows = rowsThatFit(frame.columns, wantedRows, availableHeight - PAGE_BAR_HEIGHT - RHYTHM);
        frame.perPage = frame.columns * frame.rows;
    }
    frame.pageCount = slotCount > frame.perPage ? (slotCount + frame.perPage - 1) / frame.perPage : 1;
    return frame;
}

IntRect galleryTileRect(int slotOnPage, int columns, int gridTop)
{
    columns = clampInt(columns, 1, GALLERY_MAX_COLUMNS);
    int width = galleryTileWidth(columns);
    int height = galleryTileHeight(columns);
    int row = slotOnPage / columns;
    int column = slotOnPage % columns;
    IntRect rect = {galleryGridLeft(columns) + column * (width + GALLERY_GUTTER), gridTop + row * (height + GALLERY_GUTTER),
                    width, height};
    return rect;
}

int galleryPageOfSlot(int index, int perPage)
{
    return index < 0 || perPage <= 0 ? 0 : index / perPage;
}

int neighbourAfterRemoval(int index, int count)
{
    if (index < 0 || index >= count || count <= 1) {
        return -1;
    }
    return index + 1 < count ? index + 1 : index - 1;
}

int viewerIndexAfterStep(int index, int count, int step)
{
    int next = index + step;
    return next < 0 || next >= count ? index : next;
}

ViewerZone viewerZoneAt(int x, int width)
{
    if (x < width / 3) {
        return ZONE_PREVIOUS;
    }
    return x >= width - width / 3 ? ZONE_NEXT : ZONE_TOGGLE_CHROME;
}

static int viewerBarTop(int viewHeight)
{
    return viewHeight - BOTTOM_MARGIN - VIEWER_BAR_HEIGHT;
}

IntRect viewerPictureArea(bool isChromeShown, int viewHeight)
{
    if (!isChromeShown) {
        IntRect whole = {0, 0, SCREEN_WIDTH, viewHeight};
        return whole;
    }
    int bottom = viewerBarTop(viewHeight) - RHYTHM;
    IntRect area = {0, HEADER_HEIGHT, SCREEN_WIDTH, bottom - HEADER_HEIGHT};
    return area;
}

IntRect viewerSecondaryRect(int viewHeight)
{
    IntRect rect = {GUTTER, viewerBarTop(viewHeight), VIEWER_SECONDARY_WIDTH, VIEWER_BAR_HEIGHT};
    return rect;
}

IntRect viewerPrimaryRect(int viewHeight)
{
    int left = GUTTER + VIEWER_SECONDARY_WIDTH + VIEWER_BAR_GAP;
    IntRect rect = {left, viewerBarTop(viewHeight), GUTTER + CONTENT_WIDTH - left, VIEWER_BAR_HEIGHT};
    return rect;
}

IntRect viewerBackRect()
{
    int arrowCentreX = GUTTER + HEADER_ICON_WIDTH / 2;
    IntRect rect = {arrowCentreX - VIEWER_BACK_TARGET / 2, (HEADER_ROW_HEIGHT - VIEWER_BACK_TARGET) / 2,
                    VIEWER_BACK_TARGET, VIEWER_BACK_TARGET};
    return rect;
}

static IntRect centredIn(const IntRect &outer, int side)
{
    IntRect rect = {outer.x + (outer.width - side) / 2, outer.y + (outer.height - side) / 2, side, side};
    return rect;
}

IntRect viewerBackDiscRect()
{
    return centredIn(viewerBackRect(), VIEWER_BACK_DISC);
}

IntRect viewerBackArrowRect()
{
    return centredIn(viewerBackRect(), VIEWER_BACK_ARROW_SIDE);
}

void fitInside(int sourceWidth, int sourceHeight, int boxWidth, int boxHeight, int *width, int *height)
{
    *width = 0;
    *height = 0;
    if (sourceWidth <= 0 || sourceHeight <= 0 || boxWidth <= 0 || boxHeight <= 0) {
        return;
    }
    // Compare aspect ratios by cross-multiplying (no floating point).
    if ((long long)sourceWidth * boxHeight >= (long long)sourceHeight * boxWidth) {
        *width = boxWidth;
        *height = (int)((long long)sourceHeight * boxWidth / sourceWidth);
    } else {
        *height = boxHeight;
        *width = (int)((long long)sourceWidth * boxHeight / sourceHeight);
    }
    if (*width < 1) {
        *width = 1;
    }
    if (*height < 1) {
        *height = 1;
    }
}

static IntRect makeRect(int x, int y, int width, int height)
{
    IntRect rect = {x, y, width, height};
    return rect;
}

ToggleRow toggleRow(int top, bool hasDetail)
{
    ToggleRow row;
    row.height = hasDetail ? TAP_ROW_HEIGHT : ROW_HEIGHT;
    int switchLeft = GUTTER + CONTENT_WIDTH - TOGGLE_SWITCH_WIDTH;
    int switchTop = top + (row.height - TOGGLE_SWITCH_HEIGHT) / 2;
    row.switchBox = makeRect(switchLeft, switchTop, TOGGLE_SWITCH_WIDTH, TOGGLE_SWITCH_HEIGHT);
    int stateLeft = switchLeft - TOGGLE_STATE_GAP - TOGGLE_STATE_WIDTH;
    row.state = makeRect(stateLeft, switchTop, TOGGLE_STATE_WIDTH, TOGGLE_SWITCH_HEIGHT);
    int labelWidth = stateLeft - TOGGLE_LABEL_GAP - GUTTER;
    int labelTop = top + (hasDetail ? ROW_LABEL_TOP_WITH_DETAIL : ROW_LABEL_TOP);
    row.label = makeRect(GUTTER, labelTop, labelWidth, ROW_LABEL_HEIGHT);
    row.detail = makeRect(GUTTER, top + ROW_DETAIL_TOP, labelWidth, hasDetail ? CAPTION_LINE_HEIGHT : 0);
    return row;
}

int sampleColumnCount(int requestedColumns)
{
    if (requestedColumns < 1) {
        return 1;
    }
    return requestedColumns > SAMPLE_MAX_COLUMNS ? SAMPLE_MAX_COLUMNS : requestedColumns;
}

int sampleCellWidth(int columns)
{
    int count = sampleColumnCount(columns);
    return (CONTENT_WIDTH - (count - 1) * SAMPLE_COLUMN_GAP) / count;
}

int sampleCellHeight(int innerHeight)
{
    return innerHeight + SAMPLE_CAPTION_GAP + CAPTION_LINE_HEIGHT;
}

int placeSamples(const int *innerHeights, int count, int columns, SamplePlacement *placements)
{
    int columnCount = sampleColumnCount(columns);
    int width = sampleCellWidth(columnCount);
    int rowTop = 0;
    int rowHeight = 0;
    for (int i = 0; i < count; i++) {
        int column = i % columnCount;
        if (column == 0 && i > 0) {
            rowTop += rowHeight + SAMPLE_ROW_GAP;
            rowHeight = 0;
        }
        SamplePlacement &cell = placements[i];
        cell.x = GUTTER + column * (width + SAMPLE_COLUMN_GAP);
        cell.y = rowTop;
        cell.width = width;
        cell.innerHeight = innerHeights[i];
        cell.height = sampleCellHeight(innerHeights[i]);
        if (cell.height > rowHeight) {
            rowHeight = cell.height;
        }
    }
    return count == 0 ? 0 : rowTop + rowHeight;
}

int invertedGrey(int grey)
{
    return PAPER_WHITE - grey;
}

int dimmedGrey(int grey)
{
    return grey < INK_DIM ? INK_DIM : grey;
}

static int appendText(char *out, int outSize, int length, const char *text)
{
    for (; *text && length < outSize - 1; text++) {
        out[length++] = *text;
    }
    out[length] = 0;
    return length;
}

int stepsHeadline(const char *label, const char *done, const char *total, char *out, int outSize)
{
    if (outSize <= 0) {
        return 0;
    }
    out[0] = 0;
    int length = appendText(out, outSize, 0, label);
    if (!done[0] && !total[0]) {
        return length;
    }
    length = appendText(out, outSize, length, " ");
    length = appendText(out, outSize, length, done);
    length = appendText(out, outSize, length, " / ");
    return appendText(out, outSize, length, total);
}

ReloadDecision decideReload(bool isQueueIdle, bool isLoading)
{
    if (!isQueueIdle) {
        return RELOAD_AFTER_QUEUE;
    }
    return isLoading ? RELOAD_AFTER_LOAD : RELOAD_NOW;
}

} // namespace ScreenLogic
