#ifndef NICKELDRAWER_DRAWER_LOGIC_H
#define NICKELDRAWER_DRAWER_LOGIC_H

// Pure decision logic of the drawer: no Qt, no libnickel, no C++ standard
// library, so tests/drawer_logic_test.cc builds it with the host compiler.

namespace DrawerLogic {

// ---- Paging (docs/spec/ui-kit.md §5.2, §5.3) ----

enum ViewMode { VIEW_LIST = 0, VIEW_GRID = 1 };

static const int LIST_ROWS_PER_PAGE = 7;
static const int GRID_COLUMNS = 3;
static const int GRID_ROWS = 2;
static const int GRID_TILES_PER_PAGE = GRID_COLUMNS * GRID_ROWS;

int itemsPerPage(ViewMode mode);

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

enum AppsctlCommand { COMMAND_MODEL, COMMAND_RUN_SHOW, COMMAND_RUN_BACKGROUND, COMMAND_INFO, COMMAND_REMOVE };

// Hard timeout the mod applies to an appsctl call.
int timeoutMillisecondsFor(AppsctlCommand command);

// appsctl exits 2 for protocol/usage errors and 0 otherwise (a failed
// script is ok:false with exit 0).
enum ExitClass { EXIT_REPLY, EXIT_PROTOCOL_ERROR, EXIT_UNEXPECTED };

ExitClass classifyExitCode(int exitCode);

// Host items ("_refresh", "_updates") cannot be removed or inspected.
bool isHostItemId(const char *id);

// ---- Wi-Fi retry (drawer-protocol "needs_wifi": connect, then retry once) ----

enum RunStep { RUN_SHOW_RESULT, RUN_ASK_FOR_WIFI, RUN_GIVE_UP_WIFI };

RunStep nextStepAfterRun(bool needsWifi, bool hasRetriedAfterWifi);

static const int WIFI_POLL_INTERVAL_MILLISECONDS = 1000;
static const int WIFI_POLL_ATTEMPTS = 30;

// ---- Locale ----

enum Language { LANGUAGE_ENGLISH, LANGUAGE_VIETNAMESE };

// Nickel's CurrentLocale ("vi", "vi_VN", "en_US", empty) to a fallback table.
Language languageForLocale(const char *locale);

// ---- Long-press ----

static const int LONG_PRESS_MILLISECONDS = 600;
static const int CONFIRM_TIMEOUT_MILLISECONDS = 120000;

} // namespace DrawerLogic

#endif
