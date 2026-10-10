// Host unit tests for navigation_stack (no Qt, no C++ standard library).
// Run: make -C mod/drawer docker-test

#include "../navigation_stack.h"
#include "test_expect.h"

using namespace Navigation;

static const int LIST = 0;
static const int GRID = 1;

static ViewState state(int viewMode, int page)
{
    ViewState result;
    result.viewMode = viewMode;
    result.page = page;
    return result;
}

static const ViewState NONE = state(0, 0);

// More → drawer, as DrawerNavigator::open builds it.
static Stack openDrawer()
{
    Stack stack;
    stack.push(LEVEL_MORE, "", NONE);
    stack.push(LEVEL_DRAWER, "", NONE);
    return stack;
}

static void testPushRecordsTheParentState()
{
    Stack stack = openDrawer();
    EXPECT_EQ(1, stack.push(LEVEL_APP_SCREEN, "localsend", state(GRID, 2)));
    EXPECT_EQ(3, stack.depth());
    EXPECT_EQ(LEVEL_APP_SCREEN, stack.top()->level);
    EXPECT_STREQ("localsend", stack.top()->id);
    EXPECT_EQ(0, stack.top()->state.page);
    EXPECT_EQ(GRID, stack.at(1)->state.viewMode);
    EXPECT_EQ(2, stack.at(1)->state.page);
}

static void testBackFromAppScreenRestoresDrawerModeAndPage()
{
    Stack stack = openDrawer();
    stack.push(LEVEL_APP_SCREEN, "localsend", state(LIST, 1));
    Entry parent;
    EXPECT_EQ(1, stack.pop(&parent));
    EXPECT_EQ(LEVEL_DRAWER, parent.level);
    EXPECT_EQ(LIST, parent.state.viewMode);
    EXPECT_EQ(1, parent.state.page);
    EXPECT_EQ(2, stack.depth());
}

static void testBackFromSubScreenRestoresParentPage()
{
    Stack stack = openDrawer();
    stack.push(LEVEL_APP_SCREEN, "ghsync", state(GRID, 0));
    stack.push(LEVEL_SUB_SCREEN, "settings", state(0, 3));
    stack.push(LEVEL_SUB_SCREEN, "token", state(0, 1));
    Entry parent;
    EXPECT_EQ(1, stack.pop(&parent));
    EXPECT_EQ(LEVEL_SUB_SCREEN, parent.level);
    EXPECT_STREQ("settings", parent.id);
    EXPECT_EQ(1, parent.state.page);
    EXPECT_EQ(1, stack.pop(&parent));
    EXPECT_EQ(LEVEL_APP_SCREEN, parent.level);
    EXPECT_EQ(3, parent.state.page);
    EXPECT_EQ(1, stack.pop(&parent));
    EXPECT_EQ(LEVEL_DRAWER, parent.level);
    EXPECT_EQ(GRID, parent.state.viewMode);
}

static void testBackFromDrawerReturnsToMore()
{
    Stack stack = openDrawer();
    Entry parent;
    EXPECT_EQ(1, stack.pop(&parent));
    EXPECT_EQ(LEVEL_MORE, parent.level);
    EXPECT_EQ(0, stack.pop(&parent)); // back from More is Nickel's
    EXPECT_EQ(1, stack.depth());
}

static void testModalStacksOverAnyScreen()
{
    Stack stack = openDrawer();
    stack.push(LEVEL_APP_SCREEN, "library", NONE);
    EXPECT_EQ(1, stack.push(LEVEL_MODAL, "options", state(0, 2)));
    Entry parent;
    stack.pop(&parent);
    EXPECT_EQ(LEVEL_APP_SCREEN, parent.level);
    EXPECT_EQ(2, parent.state.page);
    EXPECT_EQ(0, stack.push(LEVEL_APP_SCREEN, "other", NONE)); // a modal or sub-screen never takes an app screen over it
}

static void testRejectsOutOfOrderPushes()
{
    Stack stack = openDrawer();
    EXPECT_EQ(0, stack.push(LEVEL_DRAWER, "", NONE));
    EXPECT_EQ(0, stack.push(LEVEL_MORE, "", NONE));
    stack.push(LEVEL_APP_SCREEN, "a", NONE);
    EXPECT_EQ(0, stack.push(LEVEL_APP_SCREEN, "b", NONE));
    EXPECT_EQ(3, stack.depth());
}

static void testFullStackRefusesPush()
{
    Stack stack = openDrawer();
    stack.push(LEVEL_APP_SCREEN, "a", NONE);
    while (stack.depth() < MAX_DEPTH) {
        EXPECT_EQ(1, stack.push(LEVEL_MODAL, "m", NONE));
    }
    EXPECT_EQ(0, stack.push(LEVEL_MODAL, "m", NONE));
    EXPECT_EQ(MAX_DEPTH, stack.depth());
}

static void testPopToDrawerClosesTheScreenAndItsSubScreens()
{
    Stack stack = openDrawer();
    stack.push(LEVEL_APP_SCREEN, "ghsync", state(GRID, 4));
    stack.push(LEVEL_SUB_SCREEN, "settings", NONE);
    Entry restored;
    EXPECT_EQ(1, stack.popTo(LEVEL_DRAWER, &restored));
    EXPECT_EQ(2, stack.depth());
    EXPECT_EQ(GRID, restored.state.viewMode);
    EXPECT_EQ(4, restored.state.page);
    EXPECT_EQ(0, stack.popTo(LEVEL_MODAL, &restored));
}

static void testSubScreensStopTwoBelowMain()
{
    Stack stack = openDrawer();
    stack.push(LEVEL_APP_SCREEN, "ghsync", NONE);
    EXPECT_EQ(1, stack.push(LEVEL_SUB_SCREEN, "settings", NONE));
    EXPECT_EQ(1, stack.push(LEVEL_SUB_SCREEN, "token", NONE));
    EXPECT_EQ(0, stack.push(LEVEL_SUB_SCREEN, "deeper", NONE));
    EXPECT_EQ(2, stack.countOf(LEVEL_SUB_SCREEN));
    EXPECT_STREQ("token", stack.top()->id);
}

// A tab tap drops every level and its recorded state; reopening starts fresh.
static void testClearOnTab()
{
    Stack stack = openDrawer();
    stack.push(LEVEL_APP_SCREEN, "localsend", state(GRID, 3));
    stack.push(LEVEL_SUB_SCREEN, "settings", state(0, 2));
    EXPECT_EQ(4, stack.clear());
    EXPECT_EQ(0, stack.clear());
    EXPECT_EQ(1, stack.isEmpty());
    EXPECT_EQ(0, stack.top() != 0);
    Entry parent;
    EXPECT_EQ(0, stack.pop(&parent));
    stack.push(LEVEL_MORE, "", NONE);
    stack.push(LEVEL_DRAWER, "", NONE);
    EXPECT_EQ(2, stack.depth());
    EXPECT_EQ(0, stack.at(1)->state.page);
    EXPECT_EQ(0, stack.countOf(LEVEL_APP_SCREEN));
}

static void testLongIdIsCut()
{
    Stack stack = openDrawer();
    char longId[100];
    for (int i = 0; i < 99; i++) {
        longId[i] = 'x';
    }
    longId[99] = '\0';
    stack.push(LEVEL_APP_SCREEN, longId, NONE);
    int length = 0;
    while (stack.top()->id[length]) {
        length++;
    }
    EXPECT_EQ(MAX_ID_BYTES - 1, length);
}

static Stack openMainScreen()
{
    Stack stack = openDrawer();
    stack.push(LEVEL_APP_SCREEN, "wallpaper", NONE);
    return stack;
}

static void testPushKeepsTheValue()
{
    Stack stack = openMainScreen();
    stack.push(LEVEL_SUB_SCREEN, "viewer", state(0, 1), "0003");
    EXPECT_STREQ("0003", stack.top()->value);
    EXPECT_STREQ("", stack.at(2)->value);
}

static void testViewerFollowsThePictureOnScreen()
{
    Stack stack = openMainScreen();
    stack.push(LEVEL_SUB_SCREEN, "viewer", state(0, 1), "0003");
    EXPECT_EQ(1, stack.setTopValue("0004"));
    EXPECT_STREQ("0004", stack.top()->value);
    Entry parent;
    EXPECT_EQ(1, stack.pop(&parent));
    EXPECT_STREQ("", parent.value);
    EXPECT_EQ(1, parent.state.page);
}

static void testLongValueIsCutAndTerminated()
{
    Stack stack = openMainScreen();
    char longValue[200];
    for (int i = 0; i < 199; i++) {
        longValue[i] = 'v';
    }
    longValue[199] = '\0';
    stack.push(LEVEL_SUB_SCREEN, "viewer", NONE, longValue);
    EXPECT_EQ(MAX_VALUE_BYTES - 1, strlen(stack.top()->value));
    EXPECT_EQ(1, stack.setTopValue(longValue));
    EXPECT_EQ(MAX_VALUE_BYTES - 1, strlen(stack.top()->value));
    Stack empty;
    EXPECT_EQ(0, empty.setTopValue("x"));
}

int main()
{
    testPushKeepsTheValue();
    testViewerFollowsThePictureOnScreen();
    testLongValueIsCutAndTerminated();
    testPushRecordsTheParentState();
    testBackFromAppScreenRestoresDrawerModeAndPage();
    testBackFromSubScreenRestoresParentPage();
    testBackFromDrawerReturnsToMore();
    testModalStacksOverAnyScreen();
    testRejectsOutOfOrderPushes();
    testFullStackRefusesPush();
    testPopToDrawerClosesTheScreenAndItsSubScreens();
    testSubScreensStopTwoBelowMain();
    testClearOnTab();
    testLongIdIsCut();
    return finishTests("navigation_stack");
}
