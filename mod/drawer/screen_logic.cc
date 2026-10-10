#include "screen_logic.h"

#include <string.h>

namespace ScreenLogic {

// Average glyph advance as a fraction of the font size (metrics.go advanceByFamily).
static const double SANS_ADVANCE = 0.56;
static const double SERIF_ADVANCE = 0.52;
static const double MONO_ADVANCE = 0.6;

static const char ELLIPSIS[] = "\xE2\x80\xA6"; // U+2026

static double advanceOf(Family family)
{
    switch (family) {
    case FAMILY_SERIF:
        return SERIF_ADVANCE;
    case FAMILY_MONO:
        return MONO_ADVANCE;
    case FAMILY_SANS:
        break;
    }
    return SANS_ADVANCE;
}

int charsPerLine(int width, int size, Family family)
{
    int perLine = (int)((double)width / ((double)size * advanceOf(family)));
    return perLine < 1 ? 1 : perLine;
}

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

// The line being built: [start, start + length) of one paragraph after word joining.
struct LineBuilder {
    char buffer[2048];
    int length;
    int codePoints;
};

static void emitLine(Writer *writer, const char *bytes, int length)
{
    if (writer->lines > 0) {
        append(writer, "\n", 1);
    }
    append(writer, bytes, length);
    writer->lines++;
}

// layout.go appendLine: an empty line is never emitted.
static void flushLine(Writer *writer, LineBuilder *line)
{
    if (line->length > 0) {
        emitLine(writer, line->buffer, line->length);
    }
    line->length = 0;
    line->codePoints = 0;
}

static void setLine(LineBuilder *line, const char *bytes, int length, int codePoints)
{
    int kept = length < (int)sizeof(line->buffer) ? length : (int)sizeof(line->buffer) - 1;
    memcpy(line->buffer, bytes, kept);
    line->length = kept;
    line->codePoints = codePoints;
}

static void addWord(LineBuilder *line, const char *bytes, int length, int codePoints)
{
    if (line->length + 1 + length >= (int)sizeof(line->buffer)) {
        return; // longer than any line that fits on the screen
    }
    line->buffer[line->length++] = ' ';
    memcpy(line->buffer + line->length, bytes, length);
    line->length += length;
    line->codePoints += 1 + codePoints;
}

// layout.go wrapParagraph for one word.
static void placeWord(Writer *writer, LineBuilder *line, const char *word, int length, int perLine)
{
    int codePoints = codePointCount(word, length);
    while (codePoints > perLine) {
        flushLine(writer, line);
        int cut = advanceCodePoints(word, 0, length, perLine);
        emitLine(writer, word, cut);
        word += cut;
        length -= cut;
        codePoints -= perLine;
    }
    if (line->length == 0) {
        setLine(line, word, length, codePoints);
    } else if (line->codePoints + 1 + codePoints <= perLine) {
        addWord(line, word, length, codePoints);
    } else {
        flushLine(writer, line);
        setLine(line, word, length, codePoints);
    }
}

static void wrapParagraph(Writer *writer, const char *text, int start, int end, int perLine)
{
    LineBuilder line;
    line.length = 0;
    line.codePoints = 0;
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
        placeWord(writer, &line, text + wordStart, position - wordStart, perLine);
    }
    flushLine(writer, &line);
}

int wrapText(const char *utf8, int width, int size, Family family, char *out, int outSize)
{
    Writer writer = {out, outSize, 0, 0};
    if (outSize > 0) {
        out[0] = '\0';
    }
    if (!utf8 || outSize <= 0) {
        return 0;
    }
    int perLine = charsPerLine(width, size, family);
    int end = (int)strlen(utf8);
    int start = 0;
    for (int position = 0; position <= end; position++) {
        if (position == end || utf8[position] == '\n') {
            wrapParagraph(&writer, utf8, start, position, perLine);
            start = position + 1;
        }
    }
    return writer.lines;
}

void elideText(const char *utf8, int width, int size, Family family, bool isMiddle, char *out, int outSize)
{
    Writer writer = {out, outSize, 0, 0};
    if (outSize <= 0) {
        return;
    }
    out[0] = '\0';
    if (!utf8) {
        return;
    }
    int end = (int)strlen(utf8);
    int count = codePointCount(utf8, end);
    int limit = charsPerLine(width, size, family);
    if (count <= limit) {
        append(&writer, utf8, end);
        return;
    }
    if (limit < 3) {
        append(&writer, utf8, advanceCodePoints(utf8, 0, end, limit));
        return;
    }
    if (!isMiddle) {
        append(&writer, utf8, advanceCodePoints(utf8, 0, end, limit - 1));
        append(&writer, ELLIPSIS, 3);
        return;
    }
    int head = (limit - 1) / 2;
    int tail = limit - 1 - head;
    append(&writer, utf8, advanceCodePoints(utf8, 0, end, head));
    append(&writer, ELLIPSIS, 3);
    int tailStart = advanceCodePoints(utf8, 0, end, count - tail);
    append(&writer, utf8 + tailStart, end - tailStart);
}

// ---- Blocks and pages ----

int packPages(const Block *blocks, int count, int available, int *pageOfBlock)
{
    int page = 0;
    int used = 0;
    int onPage = 0;
    for (int i = 0; i < count; i++) {
        int needed = blocks[i].height + (onPage > 0 ? blocks[i - 1].gap : 0);
        if (onPage > 0 && used + needed > available) {
            page++;
            used = 0;
            onPage = 0;
            needed = blocks[i].height;
        }
        pageOfBlock[i] = page;
        used += needed;
        onPage++;
    }
    return page + 1;
}

int blocksHeight(const Block *blocks, int first, int count)
{
    int total = 0;
    for (int i = first; i < first + count; i++) {
        total += blocks[i].height + (i < first + count - 1 ? blocks[i].gap : 0);
    }
    return total;
}

BodyFrame layoutBody(const Block *body, int bodyCount, int footerHeight, int screenHeight, int *pageOfBlock)
{
    BodyFrame frame;
    frame.top = HEADER_HEIGHT + RHYTHM;
    frame.footerTop = screenHeight - BOTTOM_MARGIN - footerHeight;
    frame.bottom = screenHeight - BOTTOM_MARGIN;
    if (footerHeight > 0) {
        frame.bottom -= footerHeight + RHYTHM;
    }
    frame.pageCount = packPages(body, bodyCount, frame.bottom - frame.top, pageOfBlock);
    if (frame.pageCount > 1) {
        frame.pageCount = packPages(body, bodyCount, frame.bottom - frame.top - PAGE_BAR_HEIGHT - RHYTHM, pageOfBlock);
    }
    frame.pageBarTop = frame.bottom - PAGE_BAR_HEIGHT;
    return frame;
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

} // namespace ScreenLogic
