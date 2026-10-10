#ifndef NICKELDRAWER_TWEAKS_LOGIC_H
#define NICKELDRAWER_TWEAKS_LOGIC_H

// Pure logic of the "Tweaks" modal (docs/spec/tweaks.md): the JSON replies of
// `appsctl drawer tweaks` and `appsctl drawer tweak <id>` read into plain
// structs, and the 2-column grid math. No Qt, no libnickel, no C++ standard
// library, so tests/tweaks_logic_test.cc builds it with the host compiler.
//
// Why a parser of its own next to model_parser's QJsonDocument: the host test
// build has no Qt. It reads only these two small documents; texts are copied
// as UTF-8 (escapes decoded) and cut at a character boundary when too long.

#include "copperui/tokens.h"

namespace TweaksLogic {

// ---- Replies ----

enum Kind { KIND_TOGGLE, KIND_INFO, KIND_ACTION };

// The native Nickel function a reply asks for ("nickel" field).
enum NickelAction {
    NICKEL_NONE,
    NICKEL_SLEEP,
    NICKEL_REBOOT,
    NICKEL_RESCAN_LIBRARY,
    NICKEL_STATUSBAR_SCREENSHOT, // tell the status-bar helper its flag file changed (no libnickel symbol)
    NICKEL_UNKNOWN
};

static const int MAX_ITEMS = 32;
static const int ID_SIZE = 64;
static const int LABEL_SIZE = 256;
static const int VALUE_SIZE = 256;
static const int MESSAGE_SIZE = 512;

struct Item {
    char id[ID_SIZE];
    char label[LABEL_SIZE];
    char caption[VALUE_SIZE]; // optional muted line under the label (toggle or action); empty: none
    Kind kind;
    bool isOn;                  // toggle: the current value
    char text[VALUE_SIZE];      // info: the value shown
    char confirm[MESSAGE_SIZE]; // action: the question asked first; empty: none
};

struct List {
    int count;
    Item items[MAX_ITEMS];
    int skippedCount; // unknown kind, no id, or more than MAX_ITEMS
};

struct Reply {
    bool isOk;
    char toast[MESSAGE_SIZE];
    bool hasValue; // a toggle's new value is in value
    bool value;
    NickelAction nickel;
    char error[MESSAGE_SIZE]; // "error" as a string, or the message of {"code","message"}
};

enum ParseStatus {
    PARSE_OK,
    PARSE_BAD_JSON,        // not a JSON object, or malformed
    PARSE_VERSION_MISSING, // "v" absent or < 1
    PARSE_VERSION_TOO_NEW, // "v" newer than the mod understands
    PARSE_NO_ITEMS         // tweaks: "items" absent or not an array
};

// json need not be NUL-terminated; length is its size in bytes.
ParseStatus parseList(const char *json, int length, List *list);
ParseStatus parseReply(const char *json, int length, Reply *reply);

// ---- Grid (2 columns, row by row: left, right, next row) ----
// Numbers are CopperUI tokens (tweaks.* and the dialog tokens, docs/design-system.md A6);
// cmd/ndpreview/tweaks.go mirrors the rules below.

static const int COLUMN_COUNT = copperui::v1::TWEAKS_COLUMNS;
static const int ROWS_PER_PAGE = copperui::v1::TWEAKS_ROWS_PER_PAGE; // 14 tweaks a page; the modal stays above the tab bar
static const int CELL_HEIGHT = copperui::v1::TWEAKS_CELL_HEIGHT;
static const int CELL_PADDING = copperui::v1::TWEAKS_CELL_PADDING;
static const int MESSAGE_HEIGHT = copperui::v1::TWEAKS_MESSAGE_HEIGHT;
static const int DIVIDER_WIDTH = copperui::v1::STROKE_HAIRLINE;

struct Rect {
    int x;
    int y;
    int width;
    int height;
};

int rowCount(int itemCount);
// Cell of the item at indexOnPage in a grid of the given width. Columns are
// split by a divider of dividerWidth px (the right column takes the odd
// pixel); rows by a divider of dividerWidth px below each row.
Rect cellRect(int indexOnPage, int gridWidth, int cellHeight, int dividerWidth);
// Height of a page holding itemCountOnPage items (no divider under the last row).
int gridHeight(int itemCountOnPage, int cellHeight, int dividerWidth);
// The vertical divider between the columns of a row is drawn only when the
// row has a right cell.
bool rowHasRightCell(int row, int itemCountOnPage);

// ---- The box (own overlay; Nickel's N3Dialog draws its own chrome) ----
//  1. The box is DIALOG_WIDTH wide with a STROKE_BUTTON black border, white inside, centred on the screen.
//  2. Inside: the title row (HEADER_ICON_TARGET high: the title centred between two square targets,
//     the close icon in the right one), a 1 px LINE_ROW rule, then the message or the grid and, with
//     more than one page, the pager line (PAGER_HEIGHT, caption, muted, "1 / 2").
//  3. The box is as high as its first page: rows of CELL_HEIGHT, each followed by a 1 px divider,
//     and one vertical divider between the columns of every row that has a right cell.

static const int BOX_WIDTH = copperui::v1::DIALOG_WIDTH;
static const int BOX_BORDER = copperui::v1::STROKE_BUTTON;
static const int TITLE_ROW_HEIGHT = copperui::v1::HEADER_ICON_TARGET;
static const int RULE_HEIGHT = copperui::v1::STROKE_HAIRLINE;
static const int PAGER_LINE_HEIGHT = copperui::v1::PAGER_HEIGHT;

int boxInnerWidth();
// The part under the title row: the rule, then the message (itemCountOnFirstPage 0) or the grid of
// the first page and the pager line when pageCount > 1.
int panelHeight(int itemCountOnFirstPage, int pageCount);
// The whole box: border, title row and panel.
int boxHeight(int itemCountOnFirstPage, int pageCount);

// ---- Cells ----
// Everything is Nickel sans, no italic. Coordinates are relative to the cell's top-left; cellWidth
// and the cell height CELL_HEIGHT come from cellRect. inner = cellWidth - 2 * CELL_PADDING.
//  ACTION (no value; Sleep, Reboot): the label is wrapped to `inner`, at most 2 lines (the last
//     elided), each line centred; the lines are LABEL_LINE_HEIGHT high and the block of lines is
//     centred vertically.
//  TOGGLE: the right column is TOGGLE_COLUMN_WIDTH (94) wide and ends at the right padding. Its block
//     is the switch image (94 x 56), TOGGLE_STATE_TOP_GAP, then the state word (TOGGLE_STATE_HEIGHT,
//     centred in the column, regular uppercase TYPE_TOGGLE_STATE_SIZE), vertically centred in the cell. The
//     label sits left of it in `inner - 94 - RHYTHM`, at most 2 lines (last elided), start-aligned,
//     the block of lines centred vertically.
//  INFO (label and value; IP address; not tappable): the label is one line (elided to `inner`)
//     and under it the value, one caption line (elided to `inner`, muted); both start-aligned and
//     the two-line block (LABEL_LINE_HEIGHT + VALUE_LINE_HEIGHT) centred vertically.
//  CAPTION (toggle or action with a "caption"): like INFO the label is one line and the caption one muted
//     caption line under it (both elided to the label width), the two-line block centred vertically; a
//     toggle keeps its switch column and an action stays centred.
//  A tappable cell pressed is inverted as a whole (PressArea).

static const int LABEL_LINE_HEIGHT = copperui::v1::TYPE_ROW_LINE;
static const int VALUE_LINE_HEIGHT = copperui::v1::TYPE_CAPTION_LINE;
static const int LABEL_MAX_LINES = copperui::v1::BUTTON_MAX_LINES;
static const int TOGGLE_COLUMN_WIDTH = copperui::v1::TOGGLE_SWITCH_WIDTH;
static const int TOGGLE_SWITCH_HEIGHT = copperui::v1::TOGGLE_SWITCH_HEIGHT;
static const int TOGGLE_STATE_TOP_GAP = 4;
static const int TOGGLE_STATE_HEIGHT = 40;

// Most label lines a cell of this kind shows (one when a caption stacks under the label).
int maxLabelLines(Kind kind, bool hasCaption = false);

struct CellLayout {
    Rect label;        // all label lines; height = lines * LABEL_LINE_HEIGHT
    bool isCentred;    // label lines centred (action) or start-aligned
    Rect value;        // info value or caption line; zero height otherwise
    Rect switchImage;  // toggle: Nickel's switch image; zero height otherwise
    Rect stateWord;    // toggle: the state word under it; zero height otherwise
};

// labelLineCount: the lines the label took after wrapping (1..maxLabelLines), at least 1.
CellLayout cellLayout(Kind kind, int cellWidth, int cellHeight, int labelLineCount, bool hasCaption = false);
// Width the label lines are wrapped or elided to.
int labelWidth(Kind kind, int cellWidth);

} // namespace TweaksLogic

#endif
