#include "drawer_logic.h"

#include <string.h>

namespace DrawerLogic {

// Qt::Key values (Qt 5.2 qnamespace.h); repeated here so this file stays Qt-free.
static const int QT_KEY_LEFT = 0x01000012;
static const int QT_KEY_UP = 0x01000013;
static const int QT_KEY_RIGHT = 0x01000014;
static const int QT_KEY_DOWN = 0x01000015;

static const int MODEL_TIMEOUT_MILLISECONDS = 3000;
static const int ACTION_START_TIMEOUT_MILLISECONDS = 3000;
// appsctl caps kind "show" scripts at 30 s and then replies ok:false itself;
// the extra 5 s lets that reply arrive before the mod gives up.
static const int SHOW_ACTION_TIMEOUT_MILLISECONDS = 35000;
static const int INFO_TIMEOUT_MILLISECONDS = 3000;
// `drawer screen` validates ui.json and runs the app's state script (1 s budget).
static const int SCREEN_TIMEOUT_MILLISECONDS = 5000;
static const int TWEAKS_TIMEOUT_MILLISECONDS = 3000; // docs/spec/tweaks.md: 3 s per call
// Not in the protocol spec: deleting a folder on FAT and regenerating the
// NickelMenu file can take longer than a model read.
static const int REMOVE_TIMEOUT_MILLISECONDS = 10000;

static const int EXIT_CODE_OK = 0;
static const int EXIT_CODE_PROTOCOL_ERROR = 2;

int listRowsThatFit(int availableHeight)
{
    int rows = (availableHeight - copperui::v1::SECTION_HEIGHT) / copperui::v1::ROW_DRAWER;
    if (rows > LIST_ROWS_PER_PAGE) {
        return LIST_ROWS_PER_PAGE;
    }
    return rows < 1 ? 1 : rows;
}

int itemsPerPage(ViewMode mode, int gridRows, int listRows)
{
    if (mode != VIEW_GRID) {
        return listRows < 1 ? 1 : listRows;
    }
    return GRID_COLUMNS * (gridRows < 1 ? 1 : gridRows);
}

int gridCellWidth(int contentWidth, int columns)
{
    return columns > 0 ? contentWidth / columns : 0;
}

int gridSideInset(int contentWidth, int columns)
{
    return (contentWidth - columns * gridCellWidth(contentWidth, columns)) / 2;
}

int gridRowsThatFit(int availableHeight, int rowHeight)
{
    int rows = rowHeight > 0 ? availableHeight / rowHeight : 1;
    return rows < 1 ? 1 : rows;
}

CellRect gridCellRect(int indexOnPage, int columns, int contentWidth, int rowHeight)
{
    int cellWidth = gridCellWidth(contentWidth, columns);
    int column = columns > 0 ? indexOnPage % columns : 0;
    int row = columns > 0 ? indexOnPage / columns : 0;
    CellRect cell = {gridSideInset(contentWidth, columns) + column * cellWidth, row * rowHeight, cellWidth, rowHeight};
    return cell;
}

static int gridGroupHeight()
{
    return GRID_ICON_SIDE + GRID_ICON_NAME_GAP + GRID_NAME_LINES * GRID_NAME_LINE_HEIGHT;
}

int gridIconTop(int rowHeight)
{
    int top = (rowHeight - gridGroupHeight()) / 2;
    return top < 0 ? 0 : top;
}

int gridNameTop(int rowHeight)
{
    return gridIconTop(rowHeight) + GRID_ICON_SIDE + GRID_ICON_NAME_GAP;
}

int rowTextWidth(bool hasIcon, bool hasChevron)
{
    using namespace copperui::v1;
    int width = CONTENT_WIDTH - (DOT_UPDATE + RHYTHM);
    if (hasIcon) {
        width -= ICON_DRAWER_ROW + RHYTHM;
    }
    if (hasChevron) {
        width -= ICON_CHEVRON + RHYTHM;
    }
    return width;
}

bool emptyStateShowsDetails(bool hasModel)
{
    return hasModel;
}

bool emptyStateShowsButton(bool hasModel, bool hasRefreshItem)
{
    return hasModel && hasRefreshItem;
}

int pageCount(int itemCount, int perPage)
{
    if (itemCount <= 0 || perPage <= 0) {
        return 1;
    }
    return (itemCount + perPage - 1) / perPage;
}

int clampPage(int page, int itemCount, int perPage)
{
    int lastPage = pageCount(itemCount, perPage) - 1;
    if (page < 0) {
        return 0;
    }
    return page > lastPage ? lastPage : page;
}

int firstIndexOnPage(int page, int perPage)
{
    return page * perPage;
}

int itemCountOnPage(int page, int itemCount, int perPage)
{
    int remaining = itemCount - firstIndexOnPage(page, perPage);
    if (remaining <= 0) {
        return 0;
    }
    return remaining < perPage ? remaining : perPage;
}

int pageContainingItem(int itemIndex, int perPage)
{
    if (itemIndex <= 0 || perPage <= 0) {
        return 0;
    }
    return itemIndex / perPage;
}

PageKey pageKeyFor(int qtKey)
{
    if (qtKey == QT_KEY_LEFT || qtKey == QT_KEY_UP) {
        return PAGE_KEY_PREVIOUS;
    }
    if (qtKey == QT_KEY_RIGHT || qtKey == QT_KEY_DOWN) {
        return PAGE_KEY_NEXT;
    }
    return PAGE_KEY_NONE;
}

int pageAfterKey(int currentPage, PageKey key, int itemCount, int perPage)
{
    int step = key == PAGE_KEY_NEXT ? 1 : (key == PAGE_KEY_PREVIOUS ? -1 : 0);
    return clampPage(currentPage + step, itemCount, perPage);
}

bool startsSectionOnPage(int indexOnPage, int sectionOfItem, int sectionOfPreviousItem)
{
    return indexOnPage == 0 || sectionOfItem != sectionOfPreviousItem;
}

void resetRefreshPolicy(RefreshPolicy *policy)
{
    policy->partialUpdatesSinceFull = 0;
}

static RefreshKind fullRefresh(RefreshPolicy *policy)
{
    resetRefreshPolicy(policy);
    return REFRESH_FULL;
}

RefreshKind refreshForOpen(RefreshPolicy *policy)
{
    return fullRefresh(policy);
}

RefreshKind refreshForPageChange(RefreshPolicy *policy)
{
    return fullRefresh(policy);
}

RefreshKind refreshForStatusChange(RefreshPolicy *policy)
{
    if (policy->partialUpdatesSinceFull >= PARTIAL_UPDATES_BEFORE_FULL) {
        return fullRefresh(policy);
    }
    policy->partialUpdatesSinceFull++;
    return REFRESH_PARTIAL;
}

ProtocolCheck checkProtocolVersion(bool hasVersion, int version)
{
    if (!hasVersion || version < 1) {
        return PROTOCOL_MISSING;
    }
    return version > SUPPORTED_PROTOCOL_VERSION ? PROTOCOL_TOO_NEW : PROTOCOL_OK;
}

int timeoutMillisecondsFor(AppsctlCommand command)
{
    switch (command) {
    case COMMAND_MODEL:
        return MODEL_TIMEOUT_MILLISECONDS;
    case COMMAND_RUN_SHOW:
        return SHOW_ACTION_TIMEOUT_MILLISECONDS;
    case COMMAND_RUN_BACKGROUND:
        return ACTION_START_TIMEOUT_MILLISECONDS;
    case COMMAND_INFO:
        return INFO_TIMEOUT_MILLISECONDS;
    case COMMAND_REMOVE:
        return REMOVE_TIMEOUT_MILLISECONDS;
    case COMMAND_SCREEN:
        return SCREEN_TIMEOUT_MILLISECONDS;
    case COMMAND_TWEAKS:
    case COMMAND_TWEAK:
        return TWEAKS_TIMEOUT_MILLISECONDS;
    }
    return MODEL_TIMEOUT_MILLISECONDS;
}

ExitClass classifyExitCode(int exitCode)
{
    if (exitCode == EXIT_CODE_OK) {
        return EXIT_REPLY;
    }
    return exitCode == EXIT_CODE_PROTOCOL_ERROR ? EXIT_PROTOCOL_ERROR : EXIT_UNEXPECTED;
}

bool isHostItemId(const char *id)
{
    return id && id[0] == '_';
}

bool isHeaderControlId(const char *id)
{
    return id && (strcmp(id, "_refresh") == 0 || strcmp(id, "_settings") == 0);
}

HeaderButton headerButtonAt(int position)
{
    static const HeaderButton ORDER[HEADER_BUTTON_COUNT] = {HEADER_REFRESH, HEADER_LIST_TOGGLE, HEADER_GRID_TOGGLE,
                                                            HEADER_SETTINGS};
    return ORDER[position >= 0 && position < HEADER_BUTTON_COUNT ? position : 0];
}

const char *hostItemIdOfHeaderButton(HeaderButton button)
{
    switch (button) {
    case HEADER_REFRESH:
        return "_refresh";
    case HEADER_SETTINGS:
        return "_settings";
    default:
        return 0;
    }
}

int menuActionLimit(int fixedEntries)
{
    int limit = MAX_MENU_ENTRIES - fixedEntries;
    return limit > 0 ? limit : 0;
}

bool isMenuAction(const char *id, bool isInMenu)
{
    if (!isInMenu || !id) {
        return false;
    }
    return strcmp(id, "status") != 0 && strcmp(id, "state") != 0;
}

bool needsConfirmation(const char *confirmQuestion)
{
    return confirmQuestion && confirmQuestion[0] != '\0';
}

RunStep nextStepAfterRun(bool needsWifi, bool hasRetriedAfterWifi)
{
    if (!needsWifi) {
        return RUN_SHOW_RESULT;
    }
    return hasRetriedAfterWifi ? RUN_GIVE_UP_WIFI : RUN_ASK_FOR_WIFI;
}

Language languageForLocale(const char *locale)
{
    if (locale && strncmp(locale, "vi", 2) == 0 && (locale[2] == '\0' || locale[2] == '_' || locale[2] == '-')) {
        return LANGUAGE_VIETNAMESE;
    }
    return LANGUAGE_ENGLISH;
}

static bool isLocaleCharacter(char letter)
{
    return (letter >= 'a' && letter <= 'z') || (letter >= 'A' && letter <= 'Z') || letter == '_' || letter == '-';
}

static int addName(char names[][LOCALE_NAME_SIZE], int count, int maxNames, const char *name, int length)
{
    if (length <= 0 || length >= LOCALE_NAME_SIZE || count >= maxNames) {
        return count;
    }
    for (int i = 0; i < count; i++) {
        if (strlen(names[i]) == (size_t)length && strncmp(names[i], name, length) == 0) {
            return count;
        }
    }
    memcpy(names[count], name, length);
    names[count][length] = '\0';
    return count + 1;
}

int localeFileChain(const char *locale, char names[][LOCALE_NAME_SIZE], int maxNames)
{
    int count = 0;
    int length = locale ? (int)strlen(locale) : 0;
    bool isSafe = length > 0 && length < LOCALE_NAME_SIZE;
    int languageLength = length;
    for (int i = 0; i < length; i++) {
        isSafe = isSafe && isLocaleCharacter(locale[i]);
        if (languageLength == length && (locale[i] == '_' || locale[i] == '-')) {
            languageLength = i;
        }
    }
    if (isSafe) {
        count = addName(names, count, maxNames, locale, length);
        count = addName(names, count, maxNames, locale, languageLength);
    }
    return addName(names, count, maxNames, "en", 2);
}

bool isQt6UpgradePending(const char *offeredUrl, const char *skippedUrl)
{
    if (!offeredUrl || offeredUrl[0] == '\0') {
        return false;
    }
    return !skippedUrl || strcmp(offeredUrl, skippedUrl) != 0;
}

} // namespace DrawerLogic
