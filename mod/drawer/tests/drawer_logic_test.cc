// Host unit tests for drawer_logic (no Qt, no C++ standard library).
// Run: make -C mod/drawer docker-test

#include "../drawer_logic.h"
#include "test_expect.h"

using namespace DrawerLogic;

#define EXPECT_TRUE(condition) EXPECT_EQ(1, (condition) ? 1 : 0)

static void testPageCount()
{
    EXPECT_EQ(1, pageCount(0, LIST_ROWS_PER_PAGE));
    EXPECT_EQ(1, pageCount(7, LIST_ROWS_PER_PAGE));
    EXPECT_EQ(2, pageCount(8, LIST_ROWS_PER_PAGE));
    EXPECT_EQ(1, pageCount(20, itemsPerPage(VIEW_GRID, 5)));
    EXPECT_EQ(2, pageCount(21, itemsPerPage(VIEW_GRID, 5)));
    EXPECT_EQ(1, pageCount(5, 0));
}

static void testPerPage()
{
    EXPECT_EQ(7, itemsPerPage(VIEW_LIST, 5)); // rows do not matter in list mode
    EXPECT_EQ(20, itemsPerPage(VIEW_GRID, 5));
    EXPECT_EQ(4, itemsPerPage(VIEW_GRID, 1));
    EXPECT_EQ(4, itemsPerPage(VIEW_GRID, 0)); // at least one row
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
    // Item 8 is on list page 2 (index 1); with 8 per grid page (2 rows) too.
    EXPECT_EQ(1, pageContainingItem(8, LIST_ROWS_PER_PAGE));
    EXPECT_EQ(1, pageContainingItem(8, itemsPerPage(VIEW_GRID, 2)));
    EXPECT_EQ(0, pageContainingItem(7, itemsPerPage(VIEW_GRID, 2)));
    EXPECT_EQ(0, pageContainingItem(-1, itemsPerPage(VIEW_GRID, 2)));
}

static const int CONTENT_WIDTH = 1120; // 1264 - 2 * 72, the header's gutters

static void testGridCellMath()
{
    EXPECT_EQ(280, gridCellWidth(CONTENT_WIDTH, 4));
    EXPECT_EQ(0, gridSideInset(CONTENT_WIDTH, 4));
    // 1121 / 4 = 280 rem 1: the leftover pixel goes to the sides, never into a cell.
    EXPECT_EQ(280, gridCellWidth(1123, 4));
    EXPECT_EQ(1, gridSideInset(1123, 4));
    EXPECT_EQ(1, gridRowsThatFit(100, GRID_ROW_HEIGHT));
    EXPECT_EQ(5, gridRowsThatFit(1300, GRID_ROW_HEIGHT));
    EXPECT_EQ(5, gridRowsThatFit(1200, GRID_ROW_HEIGHT));
    EXPECT_EQ(4, gridRowsThatFit(1199, GRID_ROW_HEIGHT));
    // The icon and the two name lines sit inside the cell.
    EXPECT_TRUE(gridNameTop(GRID_ROW_HEIGHT) + GRID_NAME_LINES * GRID_NAME_LINE_HEIGHT <= GRID_ROW_HEIGHT);
    EXPECT_EQ(24, gridIconTop(GRID_ROW_HEIGHT));
}

// Cells for 1..12 items: identical size, on the grid lines, inside the content
// width, no overlap, last column ends where the content ends.
static void testGridCellRects()
{
    for (int widthCase = 0; widthCase < 3; widthCase++) {
        int contentWidth = CONTENT_WIDTH + widthCase; // 1120, 1121, 1122: leftover pixels
        int cellWidth = gridCellWidth(contentWidth, GRID_COLUMNS);
        int inset = gridSideInset(contentWidth, GRID_COLUMNS);
        for (int count = 1; count <= 12; count++) {
            for (int i = 0; i < count; i++) {
                CellRect cell = gridCellRect(i, GRID_COLUMNS, contentWidth, GRID_ROW_HEIGHT);
                EXPECT_EQ(cellWidth, cell.width);
                EXPECT_EQ(GRID_ROW_HEIGHT, cell.height);
                EXPECT_EQ(inset + (i % GRID_COLUMNS) * cellWidth, cell.x);
                EXPECT_EQ((i / GRID_COLUMNS) * GRID_ROW_HEIGHT, cell.y);
                EXPECT_TRUE(cell.x >= 0 && cell.x + cell.width <= contentWidth);
                if (i > 0) {
                    CellRect before = gridCellRect(i - 1, GRID_COLUMNS, contentWidth, GRID_ROW_HEIGHT);
                    bool isSameRow = before.y == cell.y;
                    EXPECT_TRUE(isSameRow ? cell.x == before.x + before.width : cell.y == before.y + GRID_ROW_HEIGHT);
                }
            }
        }
        CellRect last = gridCellRect(GRID_COLUMNS - 1, GRID_COLUMNS, contentWidth, GRID_ROW_HEIGHT);
        // Left and right leftovers differ by at most one pixel.
        int rightGap = contentWidth - (last.x + last.width);
        EXPECT_TRUE(rightGap - inset >= 0 && rightGap - inset <= 1);
    }
    CellRect third = gridCellRect(2, GRID_COLUMNS, CONTENT_WIDTH, GRID_ROW_HEIGHT);
    EXPECT_EQ(560, third.x); // the cell the device photo showed offset
    EXPECT_EQ(240, gridCellRect(4, GRID_COLUMNS, CONTENT_WIDTH, GRID_ROW_HEIGHT).y);
}

static void testHeaderControls()
{
    EXPECT_TRUE(isHeaderControlId("_refresh"));
    EXPECT_TRUE(isHeaderControlId("_updates"));
    EXPECT_TRUE(!isHeaderControlId("_retry"));
    EXPECT_TRUE(!isHeaderControlId("localsend"));
    EXPECT_TRUE(!isHeaderControlId(0));
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
    EXPECT_EQ(3000, timeoutMillisecondsFor(COMMAND_TWEAKS));
    EXPECT_EQ(3000, timeoutMillisecondsFor(COMMAND_TWEAK));
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

static void testQt6UpgradePending()
{
    EXPECT_EQ(false, isQt6UpgradePending("", ""));
    EXPECT_EQ(false, isQt6UpgradePending(0, 0));
    EXPECT_EQ(false, isQt6UpgradePending("", "http://x/a"));
    EXPECT_EQ(true, isQt6UpgradePending("http://x/a", ""));
    EXPECT_EQ(true, isQt6UpgradePending("http://x/a", 0));
    EXPECT_EQ(false, isQt6UpgradePending("http://x/a", "http://x/a")); // declined
    EXPECT_EQ(true, isQt6UpgradePending("http://x/b", "http://x/a"));  // a new offer
}

int main()
{
    testPageCount();
    testPerPage();
    testPageContents();
    testClampAndKeys();
    testViewSwitchKeepsItem();
    testGridCellMath();
    testGridCellRects();
    testHeaderControls();
    testSectionLabels();
    testRefreshPolicy();
    testProtocol();
    testWifiRetry();
    testLocale();
    testQt6UpgradePending();
    return finishTests("drawer_logic");
}
