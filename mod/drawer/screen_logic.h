#ifndef NICKELDRAWER_SCREEN_LOGIC_H
#define NICKELDRAWER_SCREEN_LOGIC_H

// Pure layout rules of an app screen (docs/spec/ui-kit.md). They are a port of
// cmd/ndpreview (metrics.go, layout.go), so the Mac preview and the device
// break lines, elide and page exactly alike. No Qt, no C++ standard library:
// tests/screen_logic_test.cc builds it with the host compiler.

namespace ScreenLogic {

// ---- Metrics in device pixels (cmd/ndpreview/metrics.go; change both) ----

static const int SCREEN_WIDTH = 1264;
static const int SCREEN_HEIGHT = 1680;
static const int GUTTER = 72;
static const int CONTENT_WIDTH = SCREEN_WIDTH - 2 * GUTTER;
static const int RHYTHM = 24;

// The header is the drawer's PageHeader (Nickel's TextHeader): a 120 px row
// and a 1 px divider; the title is Nickel's header sans (34 px on the Libra 2).
static const int HEADER_ROW_HEIGHT = 120;
static const int HEADER_DIVIDER = 1;
static const int HEADER_HEIGHT = HEADER_ROW_HEIGHT + HEADER_DIVIDER;
static const int HEADER_TITLE_SIZE = 34;
static const int HEADER_STATUS_SIZE = 32;
static const int HEADER_SIDE_WIDTH = 340; // status text and header action, each at most
static const int SEPARATOR = 2;

static const int ROW_HEIGHT = 104;
static const int TAP_ROW_HEIGHT = 128;
static const int FIELD_HEIGHT = 120;
static const int BUTTON_HEIGHT = 120;
static const int BUTTON_TALL_HEIGHT = 168;
static const int SECTION_HEIGHT = 64;
static const int SECTION_TEXT_SIZE = 28;
static const int BOTTOM_MARGIN = GUTTER;
static const int PAGE_BAR_HEIGHT = 56;

static const int BODY_SIZE = 40;
static const int BODY_LINE_HEIGHT = 56;
static const int CAPTION_SIZE = 32;
static const int CAPTION_LINE_HEIGHT = 44;
static const int HEADING_SIZE = 52;
static const int HEADING_LINE_HEIGHT = 72;
static const int ROW_TEXT_SIZE = 36;

static const int BIG_CODE_SIZE = 144;
static const int BIG_CODE_HEIGHT = 200;
static const int QR_SIDE = 480;
static const int QR_QUIET_ZONE_MODULES = 4;
static const int NOTICE_PADDING = 24;
static const int NOTICE_BORDER = SEPARATOR;
static const int KEY_COLUMN_WIDTH = 440;
static const int SWITCH_WIDTH = 96;
static const int SWITCH_HEIGHT = 52;
static const int RADIO_RING_SIDE = 52;
static const int EMPTY_SPACER_HEIGHT = 240;

static const int DIALOG_WIDTH = 1040;
static const int TOAST_BOTTOM = 260;

// Greys 0 (black) .. 255 (white). Only the notice box has a grey fill.
static const int INK_BLACK = 0;
static const int INK_MUTED = 64;
static const int INK_DIM = 140;
static const int PAPER_GREY = 232;
static const int PAPER_WHITE = 255;
static const int NO_FILL = -1;

enum Family { FAMILY_SANS, FAMILY_SERIF, FAMILY_MONO };

// ---- Text (layout.go wrapText / elide) ----

// Characters that fit in width at size, from the family's average advance; at least 1.
int charsPerLine(int width, int size, Family family);

// Unicode code points in UTF-8 text (bytes < 0: up to the terminating NUL).
int codePointCount(const char *utf8, int bytes);

// Wraps UTF-8 text like ndpreview: paragraphs split at '\n', words at Unicode
// spaces, joined with one space, over-long words cut. Writes the lines into
// out separated by '\n' (always NUL-terminated, truncated to outSize) and
// returns the number of lines. out needs at most 2 * length + 2 bytes.
int wrapText(const char *utf8, int width, int size, Family family, char *out, int outSize);

// One line: unchanged when it fits, else cut at the end or in the middle with
// "…". Writes NUL-terminated UTF-8 into out (length + 4 bytes is enough).
void elideText(const char *utf8, int width, int size, Family family, bool isMiddle, char *out, int outSize);

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
// Constants mirror cmd/ndpreview/metrics.go GALLERY_* and VIEWER_*; change both.

static const int GALLERY_SIDE_PADDING = 56;
static const int GALLERY_CONTENT_WIDTH = SCREEN_WIDTH - 2 * GALLERY_SIDE_PADDING;
static const int GALLERY_GUTTER = 24;
static const int GALLERY_DEFAULT_COLUMNS = 4;
static const int GALLERY_DEFAULT_ROWS = 3;
static const int GALLERY_MAX_COLUMNS = 6;
static const int GALLERY_MAX_ROWS = 6;
static const int GALLERY_BORDER = 1;
static const int GALLERY_CURRENT_BORDER = 4;
static const int GALLERY_BADGE_SIDE = 56;
static const int GALLERY_BADGE_INSET = 12;
static const int GALLERY_PLACEHOLDER_BORDER = 2;
static const int GALLERY_BOTTOM_MARGIN = RHYTHM;
static const int GALLERY_EMPTY_ICON_SIDE = 160;

static const int VIEWER_BAR_HEIGHT = 120;
static const int VIEWER_SECONDARY_WIDTH = 360;
static const int VIEWER_BAR_GAP = 24;

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

// Largest size with the source's aspect that fits the box (scales up too).
void fitInside(int sourceWidth, int sourceHeight, int boxWidth, int boxHeight, int *width, int *height);

// ---- Live screens ----

// A live screen (state action) is re-read this often while visible and idle;
// the e-ink budget is at most 2 updates per second, steps at most every 5 s.
static const int LIVE_REFRESH_MILLISECONDS = 5000;

} // namespace ScreenLogic

#endif
