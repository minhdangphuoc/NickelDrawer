// Host unit tests for drawer_logic (no Qt, no C++ standard library).
// Run: make -C mod/drawer docker-test

#include "../drawer_logic.h"
#include "test_expect.h"

using namespace DrawerLogic;

static void testPageCount()
{
    EXPECT_EQ(1, pageCount(0, LIST_ROWS_PER_PAGE));
    EXPECT_EQ(1, pageCount(7, LIST_ROWS_PER_PAGE));
    EXPECT_EQ(2, pageCount(8, LIST_ROWS_PER_PAGE));
    EXPECT_EQ(2, pageCount(12, GRID_TILES_PER_PAGE));
    EXPECT_EQ(3, pageCount(13, GRID_TILES_PER_PAGE));
    EXPECT_EQ(1, pageCount(5, 0));
}

static void testPerPage()
{
    EXPECT_EQ(7, itemsPerPage(VIEW_LIST));
    EXPECT_EQ(6, itemsPerPage(VIEW_GRID));
}

static void testPageContents()
{
    EXPECT_EQ(7, firstIndexOnPage(1, 7));
    EXPECT_EQ(7, itemCountOnPage(0, 10, 7));
    EXPECT_EQ(3, itemCountOnPage(1, 10, 7));
    EXPECT_EQ(0, itemCountOnPage(2, 10, 7));
    EXPECT_EQ(0, itemCountOnPage(0, 0, 7));
}

static void testClampAndKeys()
{
    EXPECT_EQ(0, clampPage(-3, 10, 7));
    EXPECT_EQ(1, clampPage(9, 10, 7));
    EXPECT_EQ(PAGE_KEY_PREVIOUS, pageKeyFor(0x01000012)); // Left
    EXPECT_EQ(PAGE_KEY_PREVIOUS, pageKeyFor(0x01000013)); // Up
    EXPECT_EQ(PAGE_KEY_NEXT, pageKeyFor(0x01000014));     // Right
    EXPECT_EQ(PAGE_KEY_NEXT, pageKeyFor(0x01000015));     // Down
    EXPECT_EQ(PAGE_KEY_NONE, pageKeyFor(0x01000046));     // raw F23 is never handled
    EXPECT_EQ(1, pageAfterKey(0, PAGE_KEY_NEXT, 10, 7));
    EXPECT_EQ(1, pageAfterKey(1, PAGE_KEY_NEXT, 10, 7)); // at the last page
    EXPECT_EQ(0, pageAfterKey(0, PAGE_KEY_PREVIOUS, 10, 7));
    EXPECT_EQ(0, pageAfterKey(1, PAGE_KEY_PREVIOUS, 10, 7));
    EXPECT_EQ(1, pageAfterKey(1, PAGE_KEY_NONE, 10, 7));
}

static void testViewSwitchKeepsItem()
{
    // Item 8 is on list page 2 (index 1) and grid page 2 (index 1).
    EXPECT_EQ(1, pageContainingItem(8, LIST_ROWS_PER_PAGE));
    EXPECT_EQ(1, pageContainingItem(8, GRID_TILES_PER_PAGE));
    EXPECT_EQ(2, pageContainingItem(12, GRID_TILES_PER_PAGE));
    EXPECT_EQ(0, pageContainingItem(-1, GRID_TILES_PER_PAGE));
}

static void testSectionLabels()
{
    EXPECT_EQ(true, startsSectionOnPage(0, 1, 1)); // first row repeats the label
    EXPECT_EQ(false, startsSectionOnPage(3, 0, 0));
    EXPECT_EQ(true, startsSectionOnPage(3, 1, 0));
}

static void testRefreshPolicy()
{
    RefreshPolicy policy;
    resetRefreshPolicy(&policy);
    EXPECT_EQ(REFRESH_FULL, refreshForOpen(&policy));
    for (int i = 0; i < PARTIAL_UPDATES_BEFORE_FULL; i++) {
        EXPECT_EQ(REFRESH_PARTIAL, refreshForStatusChange(&policy));
    }
    EXPECT_EQ(REFRESH_FULL, refreshForStatusChange(&policy));
    EXPECT_EQ(REFRESH_PARTIAL, refreshForStatusChange(&policy));
    EXPECT_EQ(REFRESH_FULL, refreshForPageChange(&policy));
    EXPECT_EQ(0, policy.partialUpdatesSinceFull);
}

static void testProtocol()
{
    EXPECT_EQ(PROTOCOL_OK, checkProtocolVersion(true, 1));
    EXPECT_EQ(PROTOCOL_TOO_NEW, checkProtocolVersion(true, 2));
    EXPECT_EQ(PROTOCOL_MISSING, checkProtocolVersion(false, 1));
    EXPECT_EQ(PROTOCOL_MISSING, checkProtocolVersion(true, 0));
    EXPECT_EQ(EXIT_REPLY, classifyExitCode(0));
    EXPECT_EQ(EXIT_PROTOCOL_ERROR, classifyExitCode(2));
    EXPECT_EQ(EXIT_UNEXPECTED, classifyExitCode(1));
    EXPECT_EQ(3000, timeoutMillisecondsFor(COMMAND_MODEL));
    EXPECT_EQ(3000, timeoutMillisecondsFor(COMMAND_RUN_BACKGROUND));
    EXPECT_EQ(true, timeoutMillisecondsFor(COMMAND_RUN_SHOW) > 30000);
    EXPECT_EQ(5000, timeoutMillisecondsFor(COMMAND_SCREEN));
    EXPECT_EQ(true, isHostItemId("_refresh"));
    EXPECT_EQ(false, isHostItemId("localsend"));
    EXPECT_EQ(false, isHostItemId(""));
    EXPECT_EQ(false, isHostItemId(0));
}

static void testWifiRetry()
{
    EXPECT_EQ(RUN_SHOW_RESULT, nextStepAfterRun(false, false));
    EXPECT_EQ(RUN_ASK_FOR_WIFI, nextStepAfterRun(true, false));
    EXPECT_EQ(RUN_GIVE_UP_WIFI, nextStepAfterRun(true, true));
    EXPECT_EQ(RUN_SHOW_RESULT, nextStepAfterRun(false, true));
}

static void testLocale()
{
    EXPECT_EQ(LANGUAGE_VIETNAMESE, languageForLocale("vi"));
    EXPECT_EQ(LANGUAGE_VIETNAMESE, languageForLocale("vi_VN"));
    EXPECT_EQ(LANGUAGE_ENGLISH, languageForLocale("en_US"));
    EXPECT_EQ(LANGUAGE_ENGLISH, languageForLocale("video"));
    EXPECT_EQ(LANGUAGE_ENGLISH, languageForLocale(""));
    EXPECT_EQ(LANGUAGE_ENGLISH, languageForLocale(0));
}

int main()
{
    testPageCount();
    testPerPage();
    testPageContents();
    testClampAndKeys();
    testViewSwitchKeepsItem();
    testSectionLabels();
    testRefreshPolicy();
    testProtocol();
    testWifiRetry();
    testLocale();
    return finishTests("drawer_logic");
}
