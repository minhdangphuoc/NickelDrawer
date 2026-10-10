// Host unit tests for row_menu_logic (no Qt, no C++ standard library).
// Run: make -C mod/drawer docker-test

#include "../row_menu_logic.h"
#include "../../common/test_expect.h"

using namespace RowMenuLogic;

static void testEntryNeedsLabelAndTarget()
{
    EXPECT_EQ(ENTRY_KEPT, judgeEntry("Info", true, 0, MAX_LIST_ITEMS));
    EXPECT_EQ(ENTRY_NO_LABEL, judgeEntry("", true, 0, MAX_LIST_ITEMS));
    EXPECT_EQ(ENTRY_NO_LABEL, judgeEntry(0, true, 0, MAX_LIST_ITEMS));
    EXPECT_EQ(ENTRY_NO_TARGET, judgeEntry("Remove…", false, 0, MAX_LIST_ITEMS));
}

static void testListMenuIsCappedAtSix()
{
    EXPECT_EQ(ENTRY_KEPT, judgeEntry("Sixth", true, 5, MAX_LIST_ITEMS));
    EXPECT_EQ(ENTRY_TOO_MANY, judgeEntry("Seventh", true, 6, MAX_LIST_ITEMS));
}

static void testGalleryMenuHasNoCap()
{
    EXPECT_EQ(ENTRY_KEPT, judgeEntry("Many", true, 50, NO_LIMIT));
}

static void testLongPressOpensOnlyAnIdleMenu()
{
    EXPECT_EQ(true, opensMenuOnLongPress(2, false, false));
    EXPECT_EQ(false, opensMenuOnLongPress(0, false, false)); // no menu declared
    EXPECT_EQ(false, opensMenuOnLongPress(2, true, false));  // one is already open
    EXPECT_EQ(false, opensMenuOnLongPress(2, false, true));  // an action is running
}

static void testRowTargetNeedsTapOrMenu()
{
    EXPECT_EQ(true, isRowTarget(true, 0));
    EXPECT_EQ(true, isRowTarget(false, 1));
    EXPECT_EQ(false, isRowTarget(false, 0));
}

int main()
{
    testEntryNeedsLabelAndTarget();
    testListMenuIsCappedAtSix();
    testGalleryMenuHasNoCap();
    testLongPressOpensOnlyAnIdleMenu();
    testRowTargetNeedsTapOrMenu();
    return finishTests("row_menu_logic");
}
