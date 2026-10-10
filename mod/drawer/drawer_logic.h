#ifndef NICKELDRAWER_DRAWER_LOGIC_H
#define NICKELDRAWER_DRAWER_LOGIC_H

// Pure decision logic of the drawer: no Qt, no libnickel, no C++ standard
// library, so tests/drawer_logic_test.cc builds it with the host compiler.

namespace DrawerLogic {

// ---- Paging (docs/spec/ui-kit.md §5.2, §5.3) ----

enum ViewMode { VIEW_LIST = 0, VIEW_GRID = 1 };

static const int LIST_ROWS_PER_PAGE = 7;

// ---- Grid geometry (docs/spec/ui-kit.md §5.2 "Grid view") ----
// A phone-launcher grid: GRID_COLUMNS identical cells spanning the content
// width (the header's gutters), icon over name, no borders.

static const int GRID_COLUMNS = 4;
static const int GRID_ROW_HEIGHT = 240;
static const int GRID_ICON_SIDE = 96;
static const int GRID_ICON_NAME_GAP = 12;
static const int GRID_NAME_LINES = 2;
static const int GRID_NAME_LINE_HEIGHT = 42;
// Rows assumed before the page area is laid out, and the lower bound after.
static const int GRID_DEFAULT_ROWS = 5;

struct CellRect {
    int x; // relative to the page area (which already sits inside the gutters)
    int y;
    int width;
    int height;
};

// Integer width of one cell; the pixels left over (contentWidth % columns)
// are split between the two sides by gridSideInset, so the grid stays centred.
int gridCellWidth(int contentWidth, int columns);
int gridSideInset(int contentWidth, int columns);
// At least 1, so a small screen still shows one row.
int gridRowsThatFit(int availableHeight, int rowHeight);
// Cell of the item at indexOnPage (row-major). Identical size for every index,
// columns*cellWidth + 2*inset == contentWidth: no drift.
CellRect gridCellRect(int indexOnPage, int columns, int contentWidth, int rowHeight);
// Top of the icon inside a cell: icon + gap + name lines are centred as one
// group, the icon always at this offset so icons line up across a row.
int gridIconTop(int rowHeight);
int gridNameTop(int rowHeight);

// gridRows is ignored in list mode.
int itemsPerPage(ViewMode mode, int gridRows);

// At least 1, so an empty drawer still shows "1 / 1".
int pageCount(int itemCount, int perPage);

int clampPage(int page, int itemCount, int perPage);
int firstIndexOnPage(int page, int perPage);
int itemCountOnPage(int page, int itemCount, int perPage);

// Page that keeps the item at itemIndex visible after a view switch.
int pageContainingItem(int itemIndex, int perPage);

enum PageKey { PAGE_KEY_NONE, PAGE_KEY_PREVIOUS, PAGE_KEY_NEXT };

// Qt key code to paging direction. Nickel's TactileButtonFilter already turned
// the page buttons into arrows with the swap setting applied:
// Left/Up = previous, Right/Down = next (re/notes/G2-findings.md Q3).
PageKey pageKeyFor(int qtKey);

// Target page for a key, or the current page when at an edge.
int pageAfterKey(int currentPage, PageKey key, int itemCount, int perPage);

// True when this item opens a new section on the page, so the view draws the
// section label above it.
bool startsSectionOnPage(int indexOnPage, int sectionOfItem, int sectionOfPreviousItem);

// ---- E-ink refresh (docs/spec/ui-kit.md §3 "E-ink refresh") ----

static const int PARTIAL_UPDATES_BEFORE_FULL = 8;

// Counts partial updates; the 9th change is upgraded to a full refresh.
struct RefreshPolicy {
    int partialUpdatesSinceFull;
};

void resetRefreshPolicy(RefreshPolicy *policy);

enum RefreshKind { REFRESH_PARTIAL, REFRESH_FULL };

// Drawer open and page change always flash; a status change is partial
// unless the partial budget is used up.
RefreshKind refreshForOpen(RefreshPolicy *policy);
RefreshKind refreshForPageChange(RefreshPolicy *policy);
RefreshKind refreshForStatusChange(RefreshPolicy *policy);

// ---- Protocol (docs/spec/drawer-protocol.md) ----

static const int SUPPORTED_PROTOCOL_VERSION = 1;

enum ProtocolCheck { PROTOCOL_OK, PROTOCOL_MISSING, PROTOCOL_TOO_NEW };

ProtocolCheck checkProtocolVersion(bool hasVersion, int version);

enum AppsctlCommand {
    COMMAND_MODEL,
    COMMAND_RUN_SHOW,
    COMMAND_RUN_BACKGROUND,
    COMMAND_INFO,
    COMMAND_REMOVE,
    COMMAND_SCREEN,
    COMMAND_TWEAKS, // docs/spec/tweaks.md: 3 s per call
    COMMAND_TWEAK
};

// Hard timeout the mod applies to an appsctl call.
int timeoutMillisecondsFor(AppsctlCommand command);

// appsctl exits 2 for protocol/usage errors and 0 otherwise (a failed
// script is ok:false with exit 0).
enum ExitClass { EXIT_REPLY, EXIT_PROTOCOL_ERROR, EXIT_UNEXPECTED };

ExitClass classifyExitCode(int exitCode);

// Host items ("_refresh", "_settings") cannot be removed or inspected.
bool isHostItemId(const char *id);

// Host items both views show as header buttons instead of rows or cells
// ("_refresh", "_settings"); other host rows (the error "_retry") stay cells.
bool isHeaderControlId(const char *id);

// The buttons at the right end of the header, left to right. The settings gear
// is last, after refresh and the list/grid toggles; it replaces the old
// updates button and carries the update dot.
enum HeaderButton { HEADER_REFRESH, HEADER_LIST_TOGGLE, HEADER_GRID_TOGGLE, HEADER_SETTINGS };

static const int HEADER_BUTTON_COUNT = 4;

HeaderButton headerButtonAt(int position);

// The host item a header button stands for ("_refresh", "_settings"); NULL for
// the list/grid toggles, which are view controls and have no item.
const char *hostItemIdOfHeaderButton(HeaderButton button);

// ---- Long-press menu ----

// Entries a Nickel popup shows without scrolling (Info + actions + Remove).
static const int MAX_MENU_ENTRIES = 8;

// How many action entries fit next to the drawer's own entries (Info, Remove).
int menuActionLimit(int fixedEntries);
// An action is a menu entry unless it is the status / state feed of the drawer
// and its screen, or the model did not mark it "menu": true (opt-in, so that
// screen-only actions never run unasked from the long-press menu).
bool isMenuAction(const char *id, bool isInMenu);
// True when the action carries a "confirm" question the user must accept first.
bool needsConfirmation(const char *confirmQuestion);

// ---- Wi-Fi retry (drawer-protocol "needs_wifi": connect, then retry once) ----

enum RunStep { RUN_SHOW_RESULT, RUN_ASK_FOR_WIFI, RUN_GIVE_UP_WIFI };

RunStep nextStepAfterRun(bool needsWifi, bool hasRetriedAfterWifi);

static const int WIFI_POLL_INTERVAL_MILLISECONDS = 1000;
static const int WIFI_POLL_ATTEMPTS = 30;

// ---- Locale ----

enum Language { LANGUAGE_ENGLISH, LANGUAGE_VIETNAMESE };

// Nickel's CurrentLocale ("vi", "vi_VN", "en_US", empty) to a fallback table.
Language languageForLocale(const char *locale);

// ---- Kobo 5.x (Qt6) upgrade offer ----

// Nickel stores the offered package URL (AvailableQt6Upgrade) and the one the
// user declined (SkippedQt6UpgradeUrl) in Kobo eReader.conf. An offer that is
// present and not declined is pending: the next sync would install firmware 5.x,
// which NickelDrawer does not support (docs/DECISIONS.md).
bool isQt6UpgradePending(const char *offeredUrl, const char *skippedUrl);

// ---- Confirmations ----

static const int CONFIRM_TIMEOUT_MILLISECONDS = 120000;

} // namespace DrawerLogic

#endif
