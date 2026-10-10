#ifndef NICKELDRAWER_SCREEN_LOGIC_H
#define NICKELDRAWER_SCREEN_LOGIC_H

#include "copperui/tokens.h"

// Pure layout rules of an app screen (docs/design-system.md). They are a port of
// cmd/ndpreview (layout.go, textfit.go), so the Mac preview and the device
// break lines, elide and page exactly alike. No Qt, no C++ standard library:
// tests/screen_logic_test.cc builds it with the host compiler.

namespace ScreenLogic {

// ---- Metrics in device pixels ----
// Every number comes from lib/copperui/tokens/copperui.tokens.json through the generated
// copperui/tokens.h, the same source as cmd/ndpreview/metrics.go; the names below are the
// layout code's short aliases.

namespace tokens = copperui::v1;

static const int SCREEN_WIDTH = tokens::SCREEN_WIDTH;
static const int SCREEN_HEIGHT = tokens::SCREEN_HEIGHT;
static const int GUTTER = tokens::GUTTER;
static const int CONTENT_WIDTH = tokens::CONTENT_WIDTH;
static const int RHYTHM = tokens::RHYTHM;

// The header is the drawer's PageHeader (Nickel's TextHeader): a 120 px row
// and a 1 px divider; the title is Nickel's header sans (34 px on the Libra 2).
static const int HEADER_ROW_HEIGHT = tokens::HEADER_ROW;
static const int HEADER_DIVIDER = tokens::HEADER_DIVIDER;
static const int HEADER_HEIGHT = tokens::HEADER_HEIGHT;
static const int HEADER_TITLE_SIZE = tokens::TYPE_HEADER_TITLE_SIZE;
static const int HEADER_STATUS_SIZE = tokens::TYPE_HEADER_STATUS_SIZE;
static const int HEADER_SIDE_WIDTH = tokens::HEADER_SIDE_WIDTH; // status text and header action, each at most
static const int SEPARATOR = tokens::STROKE_RULE;

static const int ROW_HEIGHT = tokens::ROW_COMPACT;
static const int TAP_ROW_HEIGHT = tokens::ROW_TALL;
static const int FIELD_HEIGHT = tokens::FIELD_HEIGHT;
static const int FIELD_TEXT_HEIGHT = tokens::FIELD_TEXT_HEIGHT;
static const int BUTTON_HEIGHT = tokens::BUTTON_HEIGHT;
static const int BUTTON_TALL_HEIGHT = tokens::BUTTON_TALL_HEIGHT;
static const int BUTTON_MAX_LINES = tokens::BUTTON_MAX_LINES;
static const int SECTION_HEIGHT = tokens::SECTION_HEIGHT;
static const int SECTION_TEXT_SIZE = tokens::TYPE_SECTION_SIZE;
static const int SECTION_LINE_HEIGHT = tokens::TYPE_SECTION_LINE;
static const int SECTION_LABEL_TOP = tokens::SPACE_SECTION;
static const int BOTTOM_MARGIN = tokens::MARGIN_BOTTOM;
static const int PAGE_BAR_HEIGHT = tokens::PAGER_HEIGHT;

static const int BODY_SIZE = tokens::TYPE_BODY_SIZE;
static const int BODY_LINE_HEIGHT = tokens::TYPE_BODY_LINE;
static const int CAPTION_SIZE = tokens::TYPE_CAPTION_SIZE;
static const int CAPTION_LINE_HEIGHT = tokens::TYPE_CAPTION_LINE;
static const int HEADING_SIZE = tokens::TYPE_HEADING_SIZE;
static const int HEADING_LINE_HEIGHT = tokens::TYPE_HEADING_LINE;
static const int ROW_TEXT_SIZE = tokens::TYPE_ROW_SIZE;

static const int BIG_CODE_SIZE = tokens::TYPE_BIG_CODE_SIZE;
static const int BIG_CODE_HEIGHT = tokens::TYPE_BIG_CODE_LINE;
static const int QR_SIDE = tokens::QR_SIDE;
static const int QR_QUIET_ZONE_MODULES = tokens::QR_QUIET_ZONE;
static const int NOTICE_PADDING = tokens::NOTICE_PADDING;
static const int NOTICE_BORDER = tokens::STROKE_RULE;
static const int KEY_COLUMN_WIDTH = tokens::SPACE_KEY_COLUMN;
static const int RADIO_RING_SIDE = tokens::RADIO_RING;
static const int EMPTY_SPACER_HEIGHT = tokens::EMPTY_SPACER;

static const int DIALOG_WIDTH = tokens::DIALOG_WIDTH;
static const int TOAST_BOTTOM = tokens::TOAST_BOTTOM;

// Greys 0 (black) .. 255 (white), all on the panel's 16-level grid. Only the
// notice box has a grey fill; row dividers are LINE_ROW, the section rule black.
static const int INK_BLACK = tokens::INK_BLACK;
static const int INK_MUTED = tokens::INK_MUTED;
static const int INK_DIM = tokens::INK_DIM;
static const int LINE_ROW = tokens::LINE_ROW;
static const int LINE_HEADER = tokens::LINE_HEADER;
static const int LINE_SECTION = tokens::LINE_SECTION;
static const int PAPER_GREY = tokens::PAPER_GREY;
static const int PAPER_WHITE = tokens::PAPER_WHITE;
static const int NO_FILL = -1;

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

// ---- Blocks and pages (layout.go packPages) ----

// A vertical slice of a screen; blocks never split across pages.
struct Block {
    int height;
    int gap; // space after the block when another follows on the page
};

// Fills pageOfBlock[i] and returns the page count (at least 1).
int packPages(const Block *blocks, int count, int available, int *pageOfBlock);

// Height of blocks [first, first + count): the gap of the last one is not counted.
int blocksHeight(const Block *blocks, int first, int count);

// Body area of a screen whose visible height is screenHeight.
struct BodyFrame {
    int top;
    int bottom;        // first pixel below the body (above the footer)
    int footerTop;     // y of the first footer block
    int pageBarTop;    // y of the "1 / N" line; only drawn when pageCount > 1
    int pageCount;
};

// Pages the body blocks; when they need more than one page the page bar takes
// room from every page (layout.go layoutScreen).
BodyFrame layoutBody(const Block *body, int bodyCount, int footerHeight, int screenHeight, int *pageOfBlock);

// ---- Component sizes (components.go) ----

int textBlockHeight(int lineCount, int lineHeight);
// 120 px; 168 px when the label wraps to two lines or more.
int buttonHeight(int labelLineCount);
int noticeHeight(int textLineCount, bool hasTitle);
// Height of a text line style ("caption", "heading", else body).
enum TextStyle { STYLE_BODY, STYLE_CAPTION, STYLE_HEADING };
int textSize(TextStyle style);
int lineHeight(TextStyle style);

// Index of the first list item shown when a list keeps only the last max items.
int firstKeptItem(int itemCount, int max);

// "file.epub" style values elide in the middle: a dot, no space, no colon.
bool looksLikeFileName(const char *utf8);

// Pixel x of a QR module grid of modules (quiet zone included) scaled into side
// px: the module size and the left/top offset that centres it.
struct QrPlacement {
    int moduleSize;
    int offset;
};
QrPlacement placeQr(int moduleCount, int side);

// ---- Gallery and viewer (docs/spec/wallpaper-gallery.md sections 2 and 5) ----
// Numbers come from the CopperUI tokens, like cmd/ndpreview/metrics.go.

// The grid spans the same gutters as the header title and divider (CONTENT_WIDTH).
static const int GALLERY_GUTTER = tokens::GALLERY_GUTTER;
static const int GALLERY_DEFAULT_COLUMNS = tokens::GALLERY_COLUMNS;
static const int GALLERY_DEFAULT_ROWS = tokens::GALLERY_ROWS;
static const int GALLERY_MAX_COLUMNS = tokens::GALLERY_MAX_COLUMNS;
static const int GALLERY_MAX_ROWS = tokens::GALLERY_MAX_ROWS;
static const int GALLERY_BORDER = tokens::STROKE_TILE;
static const int GALLERY_CURRENT_BORDER = tokens::STROKE_SELECT;
static const int GALLERY_BADGE_SIDE = tokens::ICON_BADGE;
static const int GALLERY_BADGE_INSET = tokens::GALLERY_BADGE_INSET;
static const int GALLERY_PLACEHOLDER_BORDER = tokens::STROKE_RULE;
static const int GALLERY_BOTTOM_MARGIN = RHYTHM;
static const int GALLERY_EMPTY_ICON_SIDE = tokens::LEGACY_EMPTY_ICON;

static const int VIEWER_BAR_HEIGHT = tokens::VIEWER_BAR;
static const int VIEWER_SECONDARY_WIDTH = tokens::VIEWER_SECONDARY;
static const int VIEWER_BAR_GAP = tokens::RHYTHM;

// Full view (chrome hidden): a back button where the header's "←" was, so
// the way back is visible over any picture (docs/spec/wallpaper-gallery.md
// section 2.2). The disc is PressArea's pressed circle (the 120 px target less
// 16 px each side); the arrow is PageHeader's drawn one (64 px, 5 px stroke).
static const int HEADER_ICON_WIDTH = tokens::HEADER_ICON; // Nickel's "headerIconWidth" on the Libra 2 class
static const int VIEWER_BACK_TARGET = tokens::HEADER_ICON_TARGET;
static const int VIEWER_BACK_DISC = tokens::VIEWER_BACK_DISC;
static const int VIEWER_BACK_OUTLINE = tokens::VIEWER_BACK_OUTLINE;
static const int VIEWER_BACK_ARROW_SIDE = HEADER_ICON_WIDTH;
static const int VIEWER_BACK_ARROW_STROKE = tokens::VIEWER_BACK_ARROW_STROKE;

// Larger files are never read (a picture is decoded from memory).
static const int MAX_PICTURE_FILE_BYTES = 8 * 1024 * 1024;

struct IntRect {
    int x;
    int y;
    int width;
    int height;
};

// 270 x 360 for 4 columns: 3:4, the screen's aspect.
int galleryTileWidth(int columns);
int galleryTileHeight(int columns);
// Left edge of the grid: GUTTER plus half of the pixels the integer tile width leaves over,
// so the grid stays centred inside the content width.
int galleryGridLeft(int columns);

struct GalleryFrame {
    int columns;
    int rows;
    int perPage;
    int pageCount; // at least 1
};

// Columns and wanted rows are clamped to 1..6. Rows shrink (at least 1) until
// the grid fits availableHeight; when the slots then need more than one page,
// the page bar takes PAGE_BAR_HEIGHT + RHYTHM and the rows are fitted again.
GalleryFrame galleryFrame(int columns, int wantedRows, int slotCount, int availableHeight);

// Slot (tile or placeholder) rectangle on its page, row-major from gridTop.
IntRect galleryTileRect(int slotOnPage, int columns, int gridTop);

// Page that holds the slot at index; 0 for a negative index.
int galleryPageOfSlot(int index, int perPage);

// Index of the picture to show after removing index of count: the next one,
// else the previous one; -1 when none is left.
int neighbourAfterRemoval(int index, int count);

// Picture index after a flip of step (-1 or +1); no wrap-around, the ends stay.
int viewerIndexAfterStep(int index, int count, int step);

enum ViewerZone { ZONE_PREVIOUS, ZONE_TOGGLE_CHROME, ZONE_NEXT };
// Left third, right third, centre of a view width px wide.
ViewerZone viewerZoneAt(int x, int width);

// Picture area: below the header and above the bottom bar, or the whole view.
IntRect viewerPictureArea(bool isChromeShown, int viewHeight);
IntRect viewerSecondaryRect(int viewHeight);
IntRect viewerPrimaryRect(int viewHeight);
// Full view's back button (tap target, view coordinates): centred on the
// header's back arrow, so it lies over the previous-picture zone and must win.
IntRect viewerBackRect();
// The disc (VIEWER_BACK_DISC) and the arrow box (VIEWER_BACK_ARROW_SIDE) centred in the target.
IntRect viewerBackDiscRect();
IntRect viewerBackArrowRect();

// Largest size with the source's aspect that fits the box (scales up too).
void fitInside(int sourceWidth, int sourceHeight, int boxWidth, int boxHeight, int *width, int *height);

// ---- Toggle row (docs/spec/ui-kit.md section 3.8) ----
// Nickel's own settings switch (ToggleSwitch, re/notes/system-icons.md): the
// state word ("ON" / "OFF", uppercase, 29 px sans, right-aligned), 25 px, then
// Nickel's switch image (94 x 56 on the Libra 2 class), ending at the content
// edge. A detail line sits under the label inside the row, like a radio sub-line.
static const int TOGGLE_SWITCH_WIDTH = tokens::TOGGLE_SWITCH_WIDTH;
static const int TOGGLE_SWITCH_HEIGHT = tokens::TOGGLE_SWITCH_HEIGHT;
static const int TOGGLE_STATE_SIZE = tokens::TYPE_TOGGLE_STATE_SIZE;
static const int TOGGLE_STATE_GAP = tokens::TOGGLE_STATE_GAP;
static const int TOGGLE_STATE_WIDTH = tokens::TOGGLE_STATE_WIDTH;
static const int TOGGLE_LABEL_GAP = RHYTHM; // between the label column and the state word
static const int ROW_LABEL_TOP = tokens::SPACE_ROW_TEXT;             // a one-line row's label
static const int ROW_LABEL_TOP_WITH_DETAIL = tokens::SPACE_ROW_LABEL_WITH_DETAIL; // the label of a row with a detail line
static const int ROW_DETAIL_TOP = tokens::SPACE_ROW_DETAIL_TOP;            // the detail line, CAPTION_LINE_HEIGHT high
static const int ROW_LABEL_HEIGHT = tokens::ROW_LABEL_HEIGHT;
static const int TAP_ROW_TEXT_TOP = tokens::SPACE_ROW_TEXT_TALL; // the label of a 128 px row
static const int RADIO_DOT_INSET = tokens::RADIO_DOT_INSET;

struct ToggleRow {
    int height; // ROW_HEIGHT, or TAP_ROW_HEIGHT with a detail line
    IntRect label;
    IntRect detail; // zero height without a detail line
    IntRect state;
    IntRect switchBox;
};

// Rectangles of a toggle row whose top is top (screen coordinates).
ToggleRow toggleRow(int top, bool hasDetail);

// ---- Live screens ----

// A live screen (state action) is re-read this often while visible and idle;
// the e-ink budget is at most 2 updates per second, steps at most every 5 s.
static const int LIVE_REFRESH_MILLISECONDS = tokens::REFRESH_LIVE;

} // namespace ScreenLogic

#endif
