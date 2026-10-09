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
// Not in the protocol spec: deleting a folder on FAT and regenerating the
// NickelMenu file can take longer than a model read.
static const int REMOVE_TIMEOUT_MILLISECONDS = 10000;

static const int EXIT_CODE_OK = 0;
static const int EXIT_CODE_PROTOCOL_ERROR = 2;

int itemsPerPage(ViewMode mode)
{
    return mode == VIEW_GRID ? GRID_TILES_PER_PAGE : LIST_ROWS_PER_PAGE;
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

} // namespace DrawerLogic
