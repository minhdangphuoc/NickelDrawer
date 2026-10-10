// Host unit tests for tweaks_logic (no Qt, no C++ standard library): the
// replies of `appsctl drawer tweaks` / `drawer tweak <id>` (docs/spec/tweaks.md)
// and the 2-column grid of the Tweaks modal. Run: make -C mod/drawer docker-test

#include "../tweaks_logic.h"
#include "test_expect.h"

using namespace TweaksLogic;

static List list; // ~40 KB: static, not on the stack

static ParseStatus parseListText(const char *json)
{
    return parseList(json, (int)strlen(json), &list);
}

static ParseStatus parseReplyText(const char *json, Reply *reply)
{
    return parseReply(json, (int)strlen(json), reply);
}

// The example of docs/spec/tweaks.md, unchanged.
static const char SPEC_LIST[] =
    "{\"v\":1,\"items\":[\n"
    "  {\"id\":\"screensaver\",\"label\":\"Sleep screen\",\"kind\":\"toggle\",\"value\":true},\n"
    "  {\"id\":\"screenshots\",\"label\":\"Screenshots\",\"kind\":\"toggle\",\"value\":false},\n"
    "  {\"id\":\"auto-usb\",\"label\":\"Auto USB connect\",\"kind\":\"toggle\",\"value\":false},\n"
    "  {\"id\":\"legibility\",\"label\":\"Text legibility\",\"kind\":\"toggle\",\"value\":false},\n"
    "  {\"id\":\"ip\",\"label\":\"IP address\",\"kind\":\"info\",\"value\":\"192.168.1.5\"},\n"
    "  {\"id\":\"invert-reboot\",\"label\":\"Invert and reboot\",\"kind\":\"action\","
    "\"confirm\":\"Invert the screen colours and reboot now?\"},\n"
    "  {\"id\":\"sleep\",\"label\":\"Sleep\",\"kind\":\"action\"},\n"
    "  {\"id\":\"reboot\",\"label\":\"Reboot\",\"kind\":\"action\",\"confirm\":\"Reboot now?\"},\n"
    "  {\"id\":\"library-rescan\",\"label\":\"Rescan library\",\"kind\":\"action\"}\n"
    "]}\n";

static void testSpecListIsReadInOrder()
{
    EXPECT_EQ(PARSE_OK, parseListText(SPEC_LIST));
    EXPECT_EQ(9, list.count);
    EXPECT_EQ(0, list.skippedCount);
    EXPECT_STREQ("screensaver", list.items[0].id);
    EXPECT_STREQ("Sleep screen", list.items[0].label);
    EXPECT_EQ(KIND_TOGGLE, list.items[0].kind);
    EXPECT_EQ(1, list.items[0].isOn);
    EXPECT_EQ(0, list.items[1].isOn);
    EXPECT_EQ(KIND_INFO, list.items[4].kind);
    EXPECT_STREQ("192.168.1.5", list.items[4].text);
    EXPECT_EQ(KIND_ACTION, list.items[5].kind);
    EXPECT_STREQ("Invert the screen colours and reboot now?", list.items[5].confirm);
    EXPECT_STREQ("", list.items[6].confirm); // no confirm: run at once
    EXPECT_STREQ("library-rescan", list.items[8].id);
}

static void testVietnameseTextAndEscapes()
{
    // Raw UTF-8, \u escapes (U+1EE8 'Ứ'), a surrogate pair (U+1F600) and simple escapes.
    EXPECT_EQ(PARSE_OK, parseListText("{\"v\":1,\"items\":[{\"id\":\"a\",\"label\":\"Tinh ch\xe1\xbb\x89nh \\u1ee8ng\","
                                      "\"kind\":\"info\",\"value\":\"\\ud83d\\ude00 \\\"q\\\" \\\\ \\/\"}]}"));
    EXPECT_EQ(1, list.count);
    EXPECT_STREQ("Tinh ch\xe1\xbb\x89nh \xe1\xbb\xa8ng", list.items[0].label);
    EXPECT_STREQ("\xf0\x9f\x98\x80 \"q\" \\ /", list.items[0].text);
}

static void testUnknownKindsAndBrokenItemsAreSkipped()
{
    EXPECT_EQ(PARSE_OK, parseListText("{\"v\":1,\"extra\":{\"nested\":[1,2,{\"x\":null}]},\"items\":["
                                      "{\"id\":\"slider\",\"label\":\"Brightness\",\"kind\":\"slider\",\"value\":3},"
                                      "{\"label\":\"No id\",\"kind\":\"toggle\"},"
                                      "\"not an object\","
                                      "{\"id\":\"ok\",\"label\":\"Fine\",\"kind\":\"toggle\",\"value\":\"yes\",\"future\":[true]}]}"));
    EXPECT_EQ(1, list.count);
    EXPECT_EQ(3, list.skippedCount);
    EXPECT_STREQ("ok", list.items[0].id);
    EXPECT_EQ(0, list.items[0].isOn); // not a boolean: reported as off
}

static void testListVersionAndShapeErrors()
{
    EXPECT_EQ(PARSE_VERSION_MISSING, parseListText("{\"items\":[]}"));
    EXPECT_EQ(PARSE_VERSION_TOO_NEW, parseListText("{\"v\":2,\"items\":[]}"));
    EXPECT_EQ(PARSE_NO_ITEMS, parseListText("{\"v\":1}"));
    EXPECT_EQ(PARSE_NO_ITEMS, parseListText("{\"v\":1,\"items\":{}}"));
    EXPECT_EQ(PARSE_OK, parseListText("  {\"v\":1,\"items\":[]}  \n"));
    EXPECT_EQ(0, list.count);
    EXPECT_EQ(PARSE_BAD_JSON, parseListText(""));
    EXPECT_EQ(PARSE_BAD_JSON, parseListText("[]"));
    EXPECT_EQ(PARSE_BAD_JSON, parseListText("{\"v\":1,\"items\":[}"));
    EXPECT_EQ(PARSE_BAD_JSON, parseListText("{\"v\":1,\"items\":[]} trailing"));
    EXPECT_EQ(PARSE_BAD_JSON, parseListText("{\"v\":1,\"items\":[{\"id\":\"a\nb\"}]}")); // raw newline in a string
    EXPECT_EQ(0, list.count); // nothing half-read survives a failure
}

static void testMoreThanMaxItemsKeepsTheFirst()
{
    static char json[MAX_ITEMS * 64 + 64];
    int length = 0;
    length += snprintf(json + length, sizeof(json) - length, "{\"v\":1,\"items\":[");
    for (int i = 0; i < MAX_ITEMS + 3; i++) {
        length += snprintf(json + length, sizeof(json) - length, "%s{\"id\":\"t%d\",\"label\":\"T\",\"kind\":\"action\"}",
                           i ? "," : "", i);
    }
    snprintf(json + length, sizeof(json) - length, "]}");
    EXPECT_EQ(PARSE_OK, parseListText(json));
    EXPECT_EQ(MAX_ITEMS, list.count);
    EXPECT_EQ(3, list.skippedCount);
    EXPECT_STREQ("t31", list.items[MAX_ITEMS - 1].id);
}

static void testLongLabelIsCutAtACharacterBoundary()
{
    static char json[LABEL_SIZE * 4];
    int length = snprintf(json, sizeof(json), "{\"v\":1,\"items\":[{\"id\":\"a\",\"kind\":\"action\",\"label\":\"");
    for (int i = 0; i < LABEL_SIZE; i++) {
        length += snprintf(json + length, sizeof(json) - length, "\xe1\xbb\x89"); // 'ỉ', 3 bytes
    }
    snprintf(json + length, sizeof(json) - length, "\"}]}");
    EXPECT_EQ(PARSE_OK, parseListText(json));
    int labelLength = (int)strlen(list.items[0].label);
    EXPECT_EQ(0, labelLength % 3); // whole characters only
    EXPECT_EQ(((LABEL_SIZE - 1) / 3) * 3, labelLength);
}

static void testToggleReply()
{
    Reply reply;
    EXPECT_EQ(PARSE_OK, parseReplyText("{\"v\":1,\"ok\":true,\"toast\":\"Sleep screen on\",\"value\":true}", &reply));
    EXPECT_EQ(1, reply.isOk);
    EXPECT_STREQ("Sleep screen on", reply.toast);
    EXPECT_EQ(1, reply.hasValue);
    EXPECT_EQ(1, reply.value);
    EXPECT_EQ(NICKEL_NONE, reply.nickel);
}

static void testNickelActionReplies()
{
    Reply reply;
    EXPECT_EQ(PARSE_OK, parseReplyText("{\"v\":1,\"ok\":true,\"toast\":\"Rebooting\",\"nickel\":\"reboot\"}", &reply));
    EXPECT_EQ(NICKEL_REBOOT, reply.nickel);
    EXPECT_EQ(0, reply.hasValue);
    EXPECT_EQ(PARSE_OK, parseReplyText("{\"v\":1,\"ok\":true,\"nickel\":\"sleep\"}", &reply));
    EXPECT_EQ(NICKEL_SLEEP, reply.nickel);
    EXPECT_STREQ("", reply.toast);
    EXPECT_EQ(PARSE_OK, parseReplyText("{\"v\":1,\"ok\":true,\"nickel\":\"rescan-library\"}", &reply));
    EXPECT_EQ(NICKEL_RESCAN_LIBRARY, reply.nickel);
    EXPECT_EQ(PARSE_OK, parseReplyText("{\"v\":1,\"ok\":true,\"nickel\":\"self-destruct\"}", &reply));
    EXPECT_EQ(NICKEL_UNKNOWN, reply.nickel);
}

static void testErrorReplies()
{
    Reply reply;
    // tweaks.md: exit 2 with a plain error sentence.
    EXPECT_EQ(PARSE_OK, parseReplyText("{\"v\":1,\"ok\":false,\"error\":\"Cannot write Kobo eReader.conf: "
                                       "permission denied\"}",
                                       &reply));
    EXPECT_EQ(0, reply.isOk);
    EXPECT_STREQ("Cannot write Kobo eReader.conf: permission denied", reply.error);
    // drawer-protocol.md usage error: {"error":{"code","message"}}, no "v".
    EXPECT_EQ(PARSE_VERSION_MISSING,
              parseReplyText("{\"error\":{\"code\":\"usage\",\"message\":\"unknown command\"}}", &reply));
    EXPECT_EQ(0, reply.isOk);
    EXPECT_STREQ("unknown command", reply.error);
    EXPECT_EQ(PARSE_BAD_JSON, parseReplyText("{\"v\":1,\"ok\":tru}", &reply));
    EXPECT_EQ(PARSE_VERSION_TOO_NEW, parseReplyText("{\"v\":3,\"ok\":true}", &reply));
    EXPECT_EQ(PARSE_VERSION_MISSING, parseReplyText("{\"v\":1.5,\"ok\":true}", &reply));
}

static void testGridRowsAndCells()
{
    EXPECT_EQ(0, rowCount(0));
    EXPECT_EQ(1, rowCount(1));
    EXPECT_EQ(1, rowCount(2));
    EXPECT_EQ(5, rowCount(9));

    // 1001 px wide with a 1 px divider: 500 left, 500 right.
    Rect left = cellRect(0, 1001, 128, 1);
    Rect right = cellRect(1, 1001, 128, 1);
    EXPECT_EQ(0, left.x);
    EXPECT_EQ(500, left.width);
    EXPECT_EQ(501, right.x);
    EXPECT_EQ(500, right.width);
    EXPECT_EQ(0, right.y);
    // 1000 px: the right column takes the odd pixel.
    EXPECT_EQ(499, cellRect(0, 1000, 128, 1).width);
    EXPECT_EQ(500, cellRect(1, 1000, 128, 1).width);
    EXPECT_EQ(1000, cellRect(1, 1000, 128, 1).x + cellRect(1, 1000, 128, 1).width);
    // Row 3 (items 6, 7) starts below three rows and three dividers.
    Rect third = cellRect(6, 1000, 128, 1);
    EXPECT_EQ(3 * 129, third.y);
    EXPECT_EQ(0, third.x);
    EXPECT_EQ(128, third.height);
}

static void testGridHeightAndRightCells()
{
    EXPECT_EQ(0, gridHeight(0, 128, 1));
    EXPECT_EQ(129, gridHeight(2, 128, 1));
    EXPECT_EQ(5 * 128 + 5, gridHeight(9, 128, 1));
    EXPECT_EQ(1, rowHasRightCell(0, 9));
    EXPECT_EQ(1, rowHasRightCell(3, 9));
    EXPECT_EQ(0, rowHasRightCell(4, 9)); // item 8 alone in the last row
    EXPECT_EQ(1, rowHasRightCell(4, 10));
    EXPECT_EQ(14, COLUMN_COUNT * ROWS_PER_PAGE);
}

static void testGridUsesTheTokens()
{
    EXPECT_EQ(2, COLUMN_COUNT);
    EXPECT_EQ(7, ROWS_PER_PAGE);
    EXPECT_EQ(128, CELL_HEIGHT);
    EXPECT_EQ(24, CELL_PADDING);
    EXPECT_EQ(160, MESSAGE_HEIGHT);
}

static void testBoxMetricsFollowTheDialogTokens()
{
    EXPECT_EQ(1040, BOX_WIDTH);
    EXPECT_EQ(3, BOX_BORDER);
    EXPECT_EQ(1034, boxInnerWidth());
    EXPECT_EQ(120, TITLE_ROW_HEIGHT);
    // The grid spans the inner width: 516 + 1 + 517.
    EXPECT_EQ(516, cellRect(0, boxInnerWidth(), CELL_HEIGHT, DIVIDER_WIDTH).width);
    EXPECT_EQ(517, cellRect(1, boxInnerWidth(), CELL_HEIGHT, DIVIDER_WIDTH).width);
    EXPECT_EQ(boxInnerWidth(), cellRect(1, boxInnerWidth(), CELL_HEIGHT, DIVIDER_WIDTH).x +
                                   cellRect(1, boxInnerWidth(), CELL_HEIGHT, DIVIDER_WIDTH).width);
}

static void testPanelIsAsHighAsItsFirstPage()
{
    int fullPage = 7 * (128 + 1);
    EXPECT_EQ(1 + fullPage + 56, panelHeight(14, 2)); // rule, grid, pager line
    EXPECT_EQ(1 + fullPage, panelHeight(14, 1));     // a single page shows no pager line
    EXPECT_EQ(1 + 5 * 129, panelHeight(9, 1));
    EXPECT_EQ(1 + 160, panelHeight(0, 1));            // the message
    EXPECT_EQ(6 + 120 + 1 + fullPage + 56, boxHeight(14, 2));
}

static void testActionCellCentresItsLines()
{
    CellLayout one = cellLayout(KIND_ACTION, 516, 128, 1);
    EXPECT_EQ(1, one.isCentred);
    EXPECT_EQ(24, one.label.x);
    EXPECT_EQ(516 - 48, one.label.width);
    EXPECT_EQ(56, one.label.height);
    EXPECT_EQ((128 - 56) / 2, one.label.y);
    EXPECT_EQ(0, one.switchImage.height);
    CellLayout two = cellLayout(KIND_ACTION, 516, 128, 2);
    EXPECT_EQ(112, two.label.height);
    EXPECT_EQ(8, two.label.y);
    EXPECT_EQ(2, maxLabelLines(KIND_ACTION));
}

static void testToggleCellPutsTheSwitchAndStateWordAtTheRight()
{
    CellLayout toggle = cellLayout(KIND_TOGGLE, 516, 128, 1);
    EXPECT_EQ(0, toggle.isCentred);
    EXPECT_EQ(94, toggle.switchImage.width);
    EXPECT_EQ(56, toggle.switchImage.height);
    EXPECT_EQ(516 - 24, toggle.switchImage.x + toggle.switchImage.width); // ends at the right padding
    EXPECT_EQ((128 - 100) / 2, toggle.switchImage.y);                      // 56 + 4 + 40 centred
    EXPECT_EQ(toggle.switchImage.x, toggle.stateWord.x);
    EXPECT_EQ(94, toggle.stateWord.width);
    EXPECT_EQ(toggle.switchImage.y + 56 + 4, toggle.stateWord.y);
    EXPECT_EQ(40, toggle.stateWord.height);
    // The label zone: inner - 94 - 24.
    EXPECT_EQ(516 - 48 - 94 - 24, toggle.label.width);
    EXPECT_EQ(labelWidth(KIND_TOGGLE, 516), toggle.label.width);
    EXPECT_EQ(24, toggle.label.x);
    EXPECT_EQ((128 - 56) / 2, toggle.label.y);
    EXPECT_EQ(112, cellLayout(KIND_TOGGLE, 516, 128, 2).label.height);
    // The label never reaches the switch column.
    EXPECT_EQ(1, toggle.label.x + toggle.label.width + 24 <= toggle.switchImage.x);
}

static void testInfoCellStacksLabelAndValue()
{
    CellLayout info = cellLayout(KIND_INFO, 517, 128, 1);
    EXPECT_EQ(0, info.isCentred);
    EXPECT_EQ(1, maxLabelLines(KIND_INFO));
    EXPECT_EQ(517 - 48, info.label.width);
    EXPECT_EQ((128 - 56 - 46) / 2, info.label.y);
    EXPECT_EQ(56, info.label.height);
    EXPECT_EQ(info.label.y + 56, info.value.y);
    EXPECT_EQ(46, info.value.height);
    EXPECT_EQ(info.label.width, info.value.width);
    EXPECT_EQ(0, info.switchImage.height);
    // An info label asked for two lines still takes one.
    EXPECT_EQ(56, cellLayout(KIND_INFO, 517, 128, 2).label.height);
}

int main()
{
    testSpecListIsReadInOrder();
    testVietnameseTextAndEscapes();
    testUnknownKindsAndBrokenItemsAreSkipped();
    testListVersionAndShapeErrors();
    testMoreThanMaxItemsKeepsTheFirst();
    testLongLabelIsCutAtACharacterBoundary();
    testToggleReply();
    testNickelActionReplies();
    testErrorReplies();
    testGridRowsAndCells();
    testGridHeightAndRightCells();
    testGridUsesTheTokens();
    testBoxMetricsFollowTheDialogTokens();
    testPanelIsAsHighAsItsFirstPage();
    testActionCellCentresItsLines();
    testToggleCellPutsTheSwitchAndStateWordAtTheRight();
    testInfoCellStacksLabelAndValue();
    return finishTests("tweaks_logic");
}
