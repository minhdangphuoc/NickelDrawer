// Host unit tests for icon_metrics (no Qt, no C++ standard library).
// Run: make -C helpers/extras docker-test

#include "../../../mod/common/test_expect.h"
#include "../icon_metrics.h"

using namespace IconMetrics;

static void testNickelsStatusBarIconsAreBottomAlignedHorizontallyCentred()
{
    // QLabel::setAlignment(0x44) in StatusBarIcon: AlignBottom | AlignHCenter.
    EXPECT_EQ(ALIGN_BOTTOM, verticalAlignmentOf(0x44));
}

static void testOtherAlignmentFlags()
{
    EXPECT_EQ(ALIGN_TOP, verticalAlignmentOf(0x21));    // AlignLeft | AlignTop
    EXPECT_EQ(ALIGN_CENTRE, verticalAlignmentOf(0x81)); // QLabel's default: AlignLeft | AlignVCenter
    EXPECT_EQ(ALIGN_CENTRE, verticalAlignmentOf(0));    // no vertical flag
    EXPECT_EQ(ALIGN_CENTRE, verticalAlignmentOf(0x04)); // horizontal flag only
}

static void testPixmapTopInsideTheBarsLabel()
{
    // The bar is 126 px high on the Libra 2 class and the icons are 80 px.
    EXPECT_EQ(46, pixmapTop(126, 80, ALIGN_BOTTOM));
    EXPECT_EQ(23, pixmapTop(126, 80, ALIGN_CENTRE));
    EXPECT_EQ(0, pixmapTop(126, 80, ALIGN_TOP));
}

static void testAFixedTopPlacedButtonSitsHigherThanItsNeighbours()
{
    // The first version: a fixed 80 px label at the top of the cell, pixmap at 0; the neighbours' pixmaps
    // start at 46. The new button shares their label height and alignment, so the offsets are equal.
    int neighbours = pixmapTop(126, 80, ALIGN_BOTTOM);
    EXPECT_EQ(46, neighbours - pixmapTop(80, 80, ALIGN_TOP) - 0);
    EXPECT_EQ(neighbours, pixmapTop(126, 80, verticalAlignmentOf(0x44)));
}

static void testAPixmapTallerThanTheLabelIsCutEvenly()
{
    EXPECT_EQ(-5, pixmapTop(70, 80, ALIGN_CENTRE));
    EXPECT_EQ(-10, pixmapTop(70, 80, ALIGN_BOTTOM));
}

static void testCanvasSideFollowsNickelWhenKnown()
{
    EXPECT_EQ(80, canvasSide(80, 64));
    EXPECT_EQ(64, canvasSide(0, 64));
    EXPECT_EQ(64, canvasSide(-1, 64));
}

int main()
{
    testNickelsStatusBarIconsAreBottomAlignedHorizontallyCentred();
    testOtherAlignmentFlags();
    testPixmapTopInsideTheBarsLabel();
    testAFixedTopPlacedButtonSitsHigherThanItsNeighbours();
    testAPixmapTallerThanTheLabelIsCutEvenly();
    testCanvasSideFollowsNickelWhenKnown();
    return finishTests("icon_metrics");
}
