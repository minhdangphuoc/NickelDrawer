// Host tests for the pager, button and list-row rules of CopperUI (no Qt, no C++ standard library).

#include "copperui/button.h"
#include "copperui/divider.h"
#include "copperui/empty_state.h"
#include "copperui/list_row.h"
#include "copperui/pager.h"
#include "test_expect.h"

using namespace copperui::v1;

static void testPagerIsHiddenOnASinglePage()
{
    EXPECT_EQ(0, pagerIsShown(0));
    EXPECT_EQ(0, pagerIsShown(1));
    EXPECT_EQ(1, pagerIsShown(2));
}

static void testPagerHalvesAreTheTouchTargets()
{
    PagerGeometry pager = pagerGeometry(1000);
    EXPECT_EQ(PAGER_TARGET_WIDTH, pager.previousTarget.width);
    EXPECT_EQ(PAGER_TARGET_HEIGHT, pager.previousTarget.height);
    EXPECT_EQ(GUTTER, pager.previousTarget.x);
    EXPECT_EQ(GUTTER + PAGER_TARGET_WIDTH, pager.nextTarget.x);
    EXPECT_EQ(PAGER_TARGET_WIDTH, pager.nextTarget.width);
    EXPECT_EQ(1000, pager.previousTarget.y);
}

static void testPagerArrowsFlankTheLabelAtIconSize()
{
    PagerGeometry pager = pagerGeometry(1000);
    EXPECT_EQ(ICON_CHEVRON, pager.previousArrow.width);
    EXPECT_EQ(ICON_CHEVRON, pager.nextArrow.height);
    int centre = SCREEN_WIDTH / 2;
    EXPECT_EQ(centre - PAGER_LABEL_WIDTH / 2, pager.label.x);
    EXPECT_EQ(pager.label.x - PAGER_ARROW_GAP, pager.previousArrow.x + pager.previousArrow.width);
    EXPECT_EQ(pager.label.x + pager.label.width + PAGER_ARROW_GAP, pager.nextArrow.x);
    // Symmetric around the page centre, and centred on the label line.
    EXPECT_EQ(centre - (pager.previousArrow.x + ICON_CHEVRON), pager.nextArrow.x - centre);
    EXPECT_EQ(1000 + PAGER_HEIGHT / 2, pager.previousArrow.y + ICON_CHEVRON / 2);
}

static void testPagerInABoxCentresOnTheBox()
{
    PagerGeometry pager = pagerGeometryIn(100, 600, 50);
    EXPECT_EQ(400 - PAGER_LABEL_WIDTH / 2, pager.label.x);
    EXPECT_EQ(100, pager.previousTarget.x);
    EXPECT_EQ(300, pager.previousTarget.width);
    EXPECT_EQ(400, pager.nextTarget.x);
}

static void testEnabledButtonIsNormal()
{
    ButtonPresentation look = buttonPresentation(true, false, false);
    EXPECT_EQ(0, look.showsPendingLabel);
    EXPECT_EQ(0, look.isDimmed);
    EXPECT_EQ(1, look.isTappable);
}

static void testDisabledButtonIsDimmedWithoutATarget()
{
    ButtonPresentation look = buttonPresentation(false, false, false);
    EXPECT_EQ(1, look.isDimmed);
    EXPECT_EQ(0, look.isTappable);
    EXPECT_EQ(0, look.showsPendingLabel);
    // Pending does not change a disabled button: nothing was tapped.
    look = buttonPresentation(false, true, false);
    EXPECT_EQ(1, look.isDimmed);
    EXPECT_EQ(0, look.showsPendingLabel);
}

static void testQuickActionShowsThePendingLabelInTheNormalLook()
{
    ButtonPresentation look = buttonPresentation(true, true, false);
    EXPECT_EQ(1, look.showsPendingLabel);
    EXPECT_EQ(0, look.isDimmed); // A9: the look stays normal
    EXPECT_EQ(0, look.isTappable);
}

static void testAsyncButtonStaysTappableWhilePending()
{
    ButtonPresentation look = buttonPresentation(true, true, true);
    EXPECT_EQ(0, look.showsPendingLabel);
    EXPECT_EQ(1, look.isTappable);
}

static void testCheckMarkText()
{
    EXPECT_EQ(1, isCheckMarkText("\xE2\x9C\x93"));
    EXPECT_EQ(0, isCheckMarkText(""));
    EXPECT_EQ(0, isCheckMarkText("\xE2\x9C\x93 done"));
    EXPECT_EQ(0, isCheckMarkText("\xE2\x9C"));
    EXPECT_EQ(0, isCheckMarkText("1.2 MB"));
}

static void testListRowPrimaryTakesTheWholeRowWithoutSecondary()
{
    ListRowGeometry row = listRowGeometry(200, 0, false, false);
    EXPECT_EQ(CONTENT_WIDTH, row.primary.width);
    EXPECT_EQ(0, row.secondary.width);
    EXPECT_EQ(0, row.checkIcon.width);
    EXPECT_EQ(GUTTER, row.primary.x);
}

static void testListRowPrimaryYieldsToTheSecondaryCaption()
{
    ListRowGeometry row = listRowGeometry(200, 150, false, false);
    EXPECT_EQ(150, row.secondary.width);
    EXPECT_EQ(GUTTER + CONTENT_WIDTH, row.secondary.x + row.secondary.width);
    EXPECT_EQ(CONTENT_WIDTH - 150 - RHYTHM, row.primary.width);
    EXPECT_EQ(TYPE_CAPTION_LINE, row.secondary.height);
    // Both texts are centred on the same line of the 104 px row.
    EXPECT_EQ(row.primary.y + row.primary.height / 2, row.secondary.y + row.secondary.height / 2);
}

static void testListRowSecondaryIsCappedSoFileNamesKeepRoom()
{
    ListRowGeometry row = listRowGeometry(0, 900, false, false);
    EXPECT_EQ(LIST_ROW_SECONDARY_MAX, row.secondary.width);
    EXPECT_EQ(CONTENT_WIDTH - LIST_ROW_SECONDARY_MAX - RHYTHM, row.primary.width);
}

static void testListRowCheckMarkIsAnIconAtTheEdge()
{
    ListRowGeometry row = listRowGeometry(0, 40, true, false);
    EXPECT_EQ(ICON_CHEVRON, row.checkIcon.width);
    EXPECT_EQ(GUTTER + CONTENT_WIDTH, row.checkIcon.x + row.checkIcon.width);
    EXPECT_EQ(0, row.secondary.width);
    EXPECT_EQ(CONTENT_WIDTH - ICON_CHEVRON - RHYTHM, row.primary.width);
    EXPECT_EQ(ROW_COMPACT / 2, row.checkIcon.y + ICON_CHEVRON / 2);
}

static void testChecklistRowMarks()
{
    ListRowMarks open = listRowMarks(true, "");
    EXPECT_EQ(1, open.hasCheckbox);
    EXPECT_EQ(0, open.isChecked);
    EXPECT_EQ(0, open.hasCheckMark);
    EXPECT_EQ(0, open.showsSecondary);
    ListRowMarks done = listRowMarks(true, "\xE2\x9C\x93");
    EXPECT_EQ(1, done.isChecked);
    EXPECT_EQ(0, done.hasCheckMark); // the done mark is the ticked box, not a second icon
    EXPECT_EQ(0, done.showsSecondary);
    ListRowMarks caption = listRowMarks(true, "5 min");
    EXPECT_EQ(0, caption.isChecked);
    EXPECT_EQ(1, caption.showsSecondary);
}

static void testPlainListRowMarksKeepTheCheckIconAtTheRight()
{
    ListRowMarks done = listRowMarks(false, "\xE2\x9C\x93");
    EXPECT_EQ(0, done.hasCheckbox);
    EXPECT_EQ(0, done.isChecked);
    EXPECT_EQ(1, done.hasCheckMark);
    EXPECT_EQ(0, done.showsSecondary);
    ListRowMarks caption = listRowMarks(false, "1.2 MB");
    EXPECT_EQ(0, caption.hasCheckMark);
    EXPECT_EQ(1, caption.showsSecondary);
}

static void testChecklistRowBoxSitsLeftOfTheTextOnTheSameBaseline()
{
    ListRowGeometry plain = listRowGeometry(200, 0, false, false);
    ListRowGeometry row = listRowGeometry(200, 0, false, true);
    EXPECT_EQ(ICON_CHECKBOX, row.checkbox.width);
    EXPECT_EQ(GUTTER, row.checkbox.x);
    EXPECT_EQ(ROW_COMPACT / 2, row.checkbox.y - 200 + ICON_CHECKBOX / 2);
    EXPECT_EQ(GUTTER + ICON_CHECKBOX + RHYTHM, row.primary.x);
    EXPECT_EQ(GUTTER + CONTENT_WIDTH, row.primary.x + row.primary.width);
    // Adding the box never moves the text vertically.
    EXPECT_EQ(plain.primary.y, row.primary.y);
    EXPECT_EQ(plain.primary.height, row.primary.height);
    EXPECT_EQ(0, plain.checkbox.width);
}

static void testChecklistRowPrimaryYieldsToBoxAndCaption()
{
    ListRowGeometry row = listRowGeometry(0, 150, false, true);
    EXPECT_EQ(CONTENT_WIDTH - ICON_CHECKBOX - RHYTHM - 150 - RHYTHM, row.primary.width);
    EXPECT_EQ(GUTTER + CONTENT_WIDTH, row.secondary.x + row.secondary.width);
}

static void testRowDividerIsGreyAndTheSectionRuleIsBlack()
{
    Divider row = divider(DIVIDER_ROW, 300);
    EXPECT_EQ(LINE_ROW, row.ink);
    EXPECT_EQ(1, row.ink > 0);
    EXPECT_EQ(STROKE_RULE, row.rect.height);
    EXPECT_EQ(GUTTER, row.rect.x);
    EXPECT_EQ(CONTENT_WIDTH, row.rect.width);
    EXPECT_EQ(300, row.rect.y);
    Divider section = divider(DIVIDER_SECTION, 40);
    EXPECT_EQ(0, section.ink);
    EXPECT_EQ(STROKE_RULE, section.rect.height);
}

static void testEmptyStateStacksIconHeadingCaptionAndButton()
{
    EmptyStateContent content = {true, 1, 2, true, 1};
    EmptyStateGeometry empty = emptyStateGeometry(content, 100, GUTTER, CONTENT_WIDTH);
    int iconTop = 100 + EMPTY_SPACER;
    EXPECT_EQ(iconTop, empty.icon.y);
    EXPECT_EQ(ICON_GRID, empty.icon.height);
    EXPECT_EQ(GUTTER + (CONTENT_WIDTH - ICON_GRID) / 2, empty.icon.x);
    EXPECT_EQ(iconTop + ICON_GRID + RHYTHM, empty.heading.y);
    EXPECT_EQ(TYPE_HEADING_LINE, empty.heading.height);
    EXPECT_EQ(CONTENT_WIDTH, empty.heading.width);
    EXPECT_EQ(empty.heading.y + TYPE_HEADING_LINE + RHYTHM, empty.caption.y);
    EXPECT_EQ(2 * TYPE_CAPTION_LINE, empty.caption.height);
    EXPECT_EQ(empty.caption.y + empty.caption.height + RHYTHM, empty.button.y);
    EXPECT_EQ(BUTTON_HEIGHT, empty.button.height);
    EXPECT_EQ(CONTENT_WIDTH, empty.button.width);
    EXPECT_EQ(empty.button.y + BUTTON_HEIGHT - 100, empty.height);
}

static void testEmptyStateWithoutIconOrButtonLeavesNoGap()
{
    EmptyStateContent loading = {false, 1, 0, false, 0};
    EmptyStateGeometry empty = emptyStateGeometry(loading, 0, GUTTER, CONTENT_WIDTH);
    EXPECT_EQ(0, empty.icon.height);
    EXPECT_EQ(EMPTY_SPACER, empty.heading.y);
    EXPECT_EQ(0, empty.caption.height);
    EXPECT_EQ(0, empty.button.height);
    EXPECT_EQ(EMPTY_SPACER + TYPE_HEADING_LINE, empty.height);
}

static void testEmptyStateButtonGrowsForTwoLines()
{
    EmptyStateContent content = {false, 2, 0, true, 2};
    EmptyStateGeometry empty = emptyStateGeometry(content, 0, GUTTER, CONTENT_WIDTH);
    EXPECT_EQ(BUTTON_TALL_HEIGHT, empty.button.height);
    EXPECT_EQ(2 * TYPE_HEADING_LINE, empty.heading.height);
    EXPECT_EQ(EMPTY_SPACER + 2 * TYPE_HEADING_LINE + RHYTHM, empty.button.y);
}

int main()
{
    testEmptyStateStacksIconHeadingCaptionAndButton();
    testEmptyStateWithoutIconOrButtonLeavesNoGap();
    testEmptyStateButtonGrowsForTwoLines();
    testRowDividerIsGreyAndTheSectionRuleIsBlack();
    testPagerIsHiddenOnASinglePage();
    testPagerHalvesAreTheTouchTargets();
    testPagerArrowsFlankTheLabelAtIconSize();
    testPagerInABoxCentresOnTheBox();
    testEnabledButtonIsNormal();
    testDisabledButtonIsDimmedWithoutATarget();
    testQuickActionShowsThePendingLabelInTheNormalLook();
    testAsyncButtonStaysTappableWhilePending();
    testCheckMarkText();
    testListRowPrimaryTakesTheWholeRowWithoutSecondary();
    testListRowPrimaryYieldsToTheSecondaryCaption();
    testListRowSecondaryIsCappedSoFileNamesKeepRoom();
    testListRowCheckMarkIsAnIconAtTheEdge();
    testChecklistRowMarks();
    testPlainListRowMarksKeepTheCheckIconAtTheRight();
    testChecklistRowBoxSitsLeftOfTheTextOnTheSameBaseline();
    testChecklistRowPrimaryYieldsToBoxAndCaption();
    return finishTests("copperui components");
}
