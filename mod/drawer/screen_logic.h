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

// ---- Live screens ----

// A live screen (state action) is re-read this often while visible and idle;
// the e-ink budget is at most 2 updates per second, steps at most every 5 s.
static const int LIVE_REFRESH_MILLISECONDS = 5000;

} // namespace ScreenLogic

#endif
