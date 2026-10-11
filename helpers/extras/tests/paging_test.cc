// Host unit tests for the paging the Tweaks panel uses (mod/common/paging.cc, no Qt, no C++ standard
// library). Run: make -C helpers/extras docker-test

#include "../../../mod/common/test_expect.h"
#include "../../../mod/common/paging.h"

using namespace Paging;

static const int ITEMS_PER_PAGE = 14;

static void testPageCountIsAtLeastOne()
{
    EXPECT_EQ(1, pageCount(0, ITEMS_PER_PAGE));
    EXPECT_EQ(1, pageCount(9, ITEMS_PER_PAGE));
    EXPECT_EQ(1, pageCount(14, ITEMS_PER_PAGE));
    EXPECT_EQ(2, pageCount(15, ITEMS_PER_PAGE));
    EXPECT_EQ(1, pageCount(5, 0));
}

static void testPageContents()
{
    EXPECT_EQ(14, firstIndexOnPage(1, ITEMS_PER_PAGE));
    EXPECT_EQ(14, itemCountOnPage(0, 20, ITEMS_PER_PAGE));
    EXPECT_EQ(6, itemCountOnPage(1, 20, ITEMS_PER_PAGE));
    EXPECT_EQ(0, itemCountOnPage(2, 20, ITEMS_PER_PAGE));
}

static void testPageKeysAreTheArrowsNickelSends()
{
    EXPECT_EQ(PAGE_KEY_PREVIOUS, pageKeyFor(0x01000012)); // Left
    EXPECT_EQ(PAGE_KEY_PREVIOUS, pageKeyFor(0x01000013)); // Up
    EXPECT_EQ(PAGE_KEY_NEXT, pageKeyFor(0x01000014));     // Right
    EXPECT_EQ(PAGE_KEY_NEXT, pageKeyFor(0x01000015));     // Down
    EXPECT_EQ(PAGE_KEY_NONE, pageKeyFor(0x01000046));
}

static void testPagingStopsAtTheEdges()
{
    EXPECT_EQ(1, pageAfterKey(0, PAGE_KEY_NEXT, 20, ITEMS_PER_PAGE));
    EXPECT_EQ(1, pageAfterKey(1, PAGE_KEY_NEXT, 20, ITEMS_PER_PAGE));
    EXPECT_EQ(0, pageAfterKey(0, PAGE_KEY_PREVIOUS, 20, ITEMS_PER_PAGE));
    EXPECT_EQ(1, pageAfterKey(1, PAGE_KEY_NONE, 20, ITEMS_PER_PAGE));
    EXPECT_EQ(0, clampPage(-3, 20, ITEMS_PER_PAGE));
    EXPECT_EQ(1, clampPage(9, 20, ITEMS_PER_PAGE));
}

int main()
{
    testPageCountIsAtLeastOne();
    testPageContents();
    testPageKeysAreTheArrowsNickelSends();
    testPagingStopsAtTheEdges();
    return finishTests("paging");
}
