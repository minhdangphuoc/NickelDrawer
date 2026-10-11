#include "screen_logic.h"

#include <string.h>

namespace ScreenLogic {

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

static ReplyDecision retryOrError(int retriesDone)
{
    return retriesDone < MAX_READ_RETRIES ? REPLY_RETRY : REPLY_SHOW_ERROR;
}

ReplyDecision decideReply(const ReplyFacts &facts)
{
    if (facts.isLoaded && !facts.isStale) {
        return REPLY_SHOW;
    }
    bool isKeepable = facts.isLoaded ? facts.isSameScreenShown : facts.isRealScreenShown && facts.isReload;
    if (isKeepable) {
        return REPLY_KEEP_PREVIOUS;
    }
    return retryOrError(facts.retriesDone);
}

} // namespace ScreenLogic
