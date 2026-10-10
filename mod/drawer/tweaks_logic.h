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

namespace TweaksLogic {

// ---- Replies ----

enum Kind { KIND_TOGGLE, KIND_INFO, KIND_ACTION };

// The native Nickel function a reply asks for ("nickel" field).
enum NickelAction { NICKEL_NONE, NICKEL_SLEEP, NICKEL_REBOOT, NICKEL_RESCAN_LIBRARY, NICKEL_UNKNOWN };

static const int MAX_ITEMS = 32;
static const int ID_SIZE = 64;
static const int LABEL_SIZE = 256;
static const int VALUE_SIZE = 256;
static const int MESSAGE_SIZE = 512;

struct Item {
    char id[ID_SIZE];
    char label[LABEL_SIZE];
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

static const int COLUMN_COUNT = 2;
static const int ROWS_PER_PAGE = 7; // 14 tweaks a page; the modal stays above the tab bar

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

} // namespace TweaksLogic

#endif
