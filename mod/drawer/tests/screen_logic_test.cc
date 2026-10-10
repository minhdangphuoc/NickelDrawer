// Host unit tests for screen_logic (no Qt, no C++ standard library).
// Expected texts come from cmd/ndpreview's wrapText/elide (layout.go), so the
// device breaks and elides exactly like the Mac preview.
// Run: make -C mod/drawer docker-test

#include "../screen_logic.h"
#include "test_expect.h"

using namespace ScreenLogic;

static void expectWrap(const char *text, int width, int size, int expectedLines, const char *expected)
{
    char out[512];
    int lines = wrapText(text, width, size, FAMILY_SANS, out, sizeof(out));
    EXPECT_EQ(expectedLines, lines);
    EXPECT_STREQ(expected, out);
}

static void expectElide(const char *text, int width, int size, Family family, bool isMiddle, const char *expected)
{
    char out[256];
    elideText(text, width, size, family, isMiddle, out, sizeof(out));
    EXPECT_STREQ(expected, out);
}

// A large serif size, to check the serif advance estimate (ndpreview_test.go uses the same).
static const int LARGE_SERIF_SIZE = 60;

static void testCharsPerLine()
{
    EXPECT_EQ(49, charsPerLine(CONTENT_WIDTH, BODY_SIZE, FAMILY_SANS));
    EXPECT_EQ(17, charsPerLine(400, BODY_SIZE, FAMILY_SANS));
    EXPECT_EQ(24, charsPerLine(772, LARGE_SERIF_SIZE, FAMILY_SERIF));
    EXPECT_EQ(1, charsPerLine(10, BIG_CODE_SIZE, FAMILY_MONO));
}

static void testCodePoints()
{
    EXPECT_EQ(5, codePointCount("short", -1));
    EXPECT_EQ(3, codePointCount("tệp", -1)); // ệ is 3 bytes
    EXPECT_EQ(2, codePointCount("\xE2\x80\xA6x", 4));
}

static void testWrap()
{
    expectWrap("Keep this screen open while files arrive.", CONTENT_WIDTH, BODY_SIZE, 1,
               "Keep this screen open while files arrive.");
    expectWrap("Để màn hình này mở khi đang nhận tệp.", 400, BODY_SIZE, 3, "Để màn hình này\nmở khi đang nhận\ntệp.");
    expectWrap("supercalifragilisticexpialidocious word", 300, BODY_SIZE, 3,
               "supercalifrag\nilisticexpial\nidocious word");
    expectWrap("line one\n\nline  two\twith tabs", CONTENT_WIDTH, BODY_SIZE, 2, "line one\nline two with tabs");
    expectWrap("Dừng và cập nhật thư viện", CONTENT_WIDTH - 2 * RHYTHM, BODY_SIZE, 1, "Dừng và cập nhật thư viện");
    expectWrap("  ", CONTENT_WIDTH, BODY_SIZE, 0, "");
    expectWrap("", CONTENT_WIDTH, BODY_SIZE, 0, "");
}

static void testWrapTruncatesToTheBuffer()
{
    char small[8];
    int lines = wrapText("one two three four", 100, BODY_SIZE, FAMILY_SANS, small, sizeof(small));
    EXPECT_EQ(5, lines); // one / two / thre / e / four
    EXPECT_EQ(7, (int)strlen(small));
    EXPECT_STREQ("one\ntwo", small);
}

static void testElide()
{
    const char *fileName = "A Very Long Title of a Book That Needs Eliding.epub";
    expectElide(fileName, KEY_COLUMN_WIDTH, ROW_TEXT_SIZE, FAMILY_SANS, true, "A Very Lon…iding.epub");
    expectElide(fileName, KEY_COLUMN_WIDTH, ROW_TEXT_SIZE, FAMILY_SANS, false, "A Very Long Title of…");
    expectElide("Thiết bị của tôi có tên rất dài", 300, ROW_TEXT_SIZE, FAMILY_SANS, false, "Thiết bị của …");
    expectElide("short", KEY_COLUMN_WIDTH, ROW_TEXT_SIZE, FAMILY_SANS, true, "short");
    expectElide("abcdef", 40, ROW_TEXT_SIZE, FAMILY_SANS, false, "a");
    expectElide("Việc cần làm rất nhiều thứ", 772, LARGE_SERIF_SIZE, FAMILY_SERIF, false, "Việc cần làm rất nhiều …");
}

static void testPackPages()
{
    Block blocks[] = {{200, 24}, {200, 24}, {200, 0}, {104, 0}};
    int pages[4];
    EXPECT_EQ(1, packPages(blocks, 4, 1000, pages));
    EXPECT_EQ(2, packPages(blocks, 4, 450, pages));
    EXPECT_EQ(0, pages[1]); // 200 + 24 + 200 = 424 fits
    EXPECT_EQ(1, pages[2]);
    EXPECT_EQ(1, pages[3]);
    EXPECT_EQ(1, packPages(blocks, 0, 100, pages)); // an empty body still has one page
    Block tall[] = {{900, 24}, {900, 24}};
    EXPECT_EQ(2, packPages(tall, 2, 500, pages)); // a block taller than a page gets a page of its own
    EXPECT_EQ(424, blocksHeight(blocks, 0, 2));
}

static void testLayoutBody()
{
    Block footer[] = {{BUTTON_HEIGHT, RHYTHM}};
    int footerHeight = blocksHeight(footer, 0, 1);
    Block body[12];
    for (int i = 0; i < 12; i++) {
        body[i].height = ROW_HEIGHT;
        body[i].gap = 0;
    }
    int pages[12];
    // 1680 - 72 - 144 (footer + rhythm) - 145 (header 121 + rhythm) = 1319 px: 12 rows fit.
    BodyFrame frame = layoutBody(body, 12, footerHeight, SCREEN_HEIGHT, pages);
    EXPECT_EQ(HEADER_HEIGHT + RHYTHM, frame.top);
    EXPECT_EQ(1319 + frame.top, frame.bottom);
    EXPECT_EQ(SCREEN_HEIGHT - BOTTOM_MARGIN - BUTTON_HEIGHT, frame.footerTop);
    EXPECT_EQ(1, frame.pageCount);
    // On a shorter view (Nickel's status and tab bars) the page bar takes room from each page.
    frame = layoutBody(body, 12, footerHeight, 1300, pages);
    EXPECT_EQ(2, frame.pageCount);
    EXPECT_EQ(frame.bottom - PAGE_BAR_HEIGHT, frame.pageBarTop);
    // 939 px body, minus 80 for the bar: 8 rows of 104 on page 1.
    EXPECT_EQ(0, pages[7]);
    EXPECT_EQ(1, pages[8]);
}

static void testComponentSizes()
{
    EXPECT_EQ(BUTTON_HEIGHT, buttonHeight(1));
    EXPECT_EQ(BUTTON_TALL_HEIGHT, buttonHeight(2));
    EXPECT_EQ(2 * NOTICE_PADDING + BODY_LINE_HEIGHT, noticeHeight(1, false));
    EXPECT_EQ(2 * NOTICE_PADDING + 3 * BODY_LINE_HEIGHT, noticeHeight(2, true));
    EXPECT_EQ(CAPTION_SIZE, textSize(STYLE_CAPTION));
    EXPECT_EQ(HEADING_LINE_HEIGHT, lineHeight(STYLE_HEADING));
    EXPECT_EQ(3 * BODY_LINE_HEIGHT, textBlockHeight(3, lineHeight(STYLE_BODY)));
}

static void testListAndValues()
{
    EXPECT_EQ(0, firstKeptItem(10, 50));
    EXPECT_EQ(5, firstKeptItem(55, 50));
    EXPECT_EQ(0, firstKeptItem(3, 0));
    EXPECT_EQ(true, looksLikeFileName("book.epub"));
    EXPECT_EQ(false, looksLikeFileName("192.168.1.2:53317"));
    EXPECT_EQ(false, looksLikeFileName("two words."));
    EXPECT_EQ(false, looksLikeFileName(0));
}

static void testQrPlacement()
{
    QrPlacement placement = placeQr(21 + 2 * QR_QUIET_ZONE_MODULES, QR_SIDE);
    EXPECT_EQ(16, placement.moduleSize); // 480 / 29
    EXPECT_EQ(8, placement.offset);      // (480 - 464) / 2
    EXPECT_EQ(0, placeQr(0, QR_SIDE).moduleSize);
}

static void testGalleryMetricsMatchTheSpec()
{
    EXPECT_EQ(270, galleryTileWidth(4));
    EXPECT_EQ(360, galleryTileHeight(4));
    EXPECT_EQ(1152, GALLERY_CONTENT_WIDTH);
    IntRect last = galleryTileRect(11, 4, 100);
    EXPECT_EQ(56 + 3 * (270 + 24), last.x);
    EXPECT_EQ(100 + 2 * (360 + 24), last.y);
    EXPECT_EQ(100 + 1128, last.y + last.height); // three rows are 1128 px
}

static void testGalleryRowsThatFit()
{
    GalleryFrame frame = galleryFrame(4, 3, 12, 1128);
    EXPECT_EQ(3, frame.rows);
    EXPECT_EQ(12, frame.perPage);
    EXPECT_EQ(1, frame.pageCount);
    frame = galleryFrame(4, 3, 12, 1127); // one pixel short: two rows
    EXPECT_EQ(2, frame.rows);
    EXPECT_EQ(8, frame.perPage);
    EXPECT_EQ(2, frame.pageCount);
    frame = galleryFrame(4, 3, 5, 10); // never below one row
    EXPECT_EQ(1, frame.rows);
    EXPECT_EQ(2, frame.pageCount);
}

static void testGalleryPageBarTakesRoomOnlyWhenPaging()
{
    EXPECT_EQ(3, galleryFrame(4, 3, 12, 1200).rows);
    GalleryFrame paged = galleryFrame(4, 3, 13, 1200);
    EXPECT_EQ(2, paged.rows); // 1200 - 80 = 1120 < 1128
    EXPECT_EQ(2, paged.pageCount);
    EXPECT_EQ(3, galleryFrame(4, 3, 13, 1208).rows);
    EXPECT_EQ(1, galleryFrame(4, 3, 0, 1500).pageCount);
}

static void testGalleryClampsColumnsAndRows()
{
    GalleryFrame frame = galleryFrame(0, 99, 1, 100000);
    EXPECT_EQ(1, frame.columns);
    EXPECT_EQ(GALLERY_MAX_ROWS, frame.rows);
    EXPECT_EQ(GALLERY_MAX_COLUMNS, galleryFrame(50, 1, 1, 1000).columns);
}

static void testGalleryPageOfSlot()
{
    EXPECT_EQ(0, galleryPageOfSlot(0, 12));
    EXPECT_EQ(0, galleryPageOfSlot(11, 12));
    EXPECT_EQ(1, galleryPageOfSlot(12, 12));
    EXPECT_EQ(2, galleryPageOfSlot(17, 8));
    EXPECT_EQ(0, galleryPageOfSlot(-1, 12));
    EXPECT_EQ(0, galleryPageOfSlot(5, 0));
}

static void testViewerZones()
{
    EXPECT_EQ(ZONE_PREVIOUS, viewerZoneAt(0, 1264));
    EXPECT_EQ(ZONE_PREVIOUS, viewerZoneAt(420, 1264));
    EXPECT_EQ(ZONE_TOGGLE_CHROME, viewerZoneAt(421, 1264));
    EXPECT_EQ(ZONE_TOGGLE_CHROME, viewerZoneAt(842, 1264));
    EXPECT_EQ(ZONE_NEXT, viewerZoneAt(843, 1264));
    EXPECT_EQ(ZONE_NEXT, viewerZoneAt(1263, 1264));
}

static void testViewerNavigationHasNoWrap()
{
    EXPECT_EQ(2, viewerIndexAfterStep(1, 5, 1));
    EXPECT_EQ(0, viewerIndexAfterStep(1, 5, -1));
    EXPECT_EQ(0, viewerIndexAfterStep(0, 5, -1));
    EXPECT_EQ(4, viewerIndexAfterStep(4, 5, 1));
    EXPECT_EQ(0, viewerIndexAfterStep(0, 0, 1));
}

static void testNeighbourAfterRemoval()
{
    EXPECT_EQ(3, neighbourAfterRemoval(2, 5));
    EXPECT_EQ(3, neighbourAfterRemoval(4, 5)); // the last one: the previous
    EXPECT_EQ(1, neighbourAfterRemoval(0, 3));
    EXPECT_EQ(-1, neighbourAfterRemoval(0, 1));
    EXPECT_EQ(-1, neighbourAfterRemoval(7, 3));
    EXPECT_EQ(-1, neighbourAfterRemoval(-1, 3));
}

static void testViewerAreas()
{
    IntRect full = viewerPictureArea(false, 1500);
    EXPECT_EQ(0, full.y);
    EXPECT_EQ(1500, full.height);
    IntRect shown = viewerPictureArea(true, 1500);
    EXPECT_EQ(HEADER_HEIGHT, shown.y);
    IntRect secondary = viewerSecondaryRect(1500);
    IntRect primary = viewerPrimaryRect(1500);
    EXPECT_EQ(1500 - BOTTOM_MARGIN - 120, secondary.y);
    EXPECT_EQ(360, secondary.width);
    EXPECT_EQ(secondary.x + secondary.width + 24, primary.x);
    EXPECT_EQ(GUTTER + CONTENT_WIDTH, primary.x + primary.width);
    EXPECT_EQ(secondary.y - RHYTHM, shown.y + shown.height);
}

// Full view: the only visible way back sits where the header's "←" was, a
// 120 px target inside the picture area (below the status bar: the view starts
// under it), over the previous-picture zone, so the layout must add it last.
static void testViewerBackButtonInFullView()
{
    IntRect target = viewerBackRect();
    EXPECT_EQ(120, target.width);
    EXPECT_EQ(120, target.height);
    EXPECT_EQ(0, target.y);
    EXPECT_EQ(GUTTER + HEADER_ICON_WIDTH / 2, target.x + target.width / 2);
    IntRect full = viewerPictureArea(false, 1552);
    EXPECT_EQ(1, target.x >= full.x && target.y >= full.y);
    EXPECT_EQ(1, target.x + target.width <= full.x + full.width && target.y + target.height <= full.y + full.height);
    EXPECT_EQ(ZONE_PREVIOUS, viewerZoneAt(target.x, full.width));
    EXPECT_EQ(ZONE_PREVIOUS, viewerZoneAt(target.x + target.width - 1, full.width));
    IntRect disc = viewerBackDiscRect();
    EXPECT_EQ(88, disc.width);
    EXPECT_EQ(target.x + 16, disc.x);
    EXPECT_EQ(target.y + 16, disc.y);
    IntRect arrow = viewerBackArrowRect();
    EXPECT_EQ(64, arrow.width);
    EXPECT_EQ(target.x + 28, arrow.x);
    EXPECT_EQ(1, arrow.width < disc.width);
}

static void testFitInside()
{
    int w = 0;
    int h = 0;
    fitInside(300, 400, 270, 360, &w, &h);
    EXPECT_EQ(270, w);
    EXPECT_EQ(360, h);
    fitInside(4000, 3000, 270, 360, &w, &h); // landscape in a portrait tile
    EXPECT_EQ(270, w);
    EXPECT_EQ(202, h);
    fitInside(100, 1000, 500, 500, &w, &h);
    EXPECT_EQ(50, w);
    EXPECT_EQ(500, h);
    fitInside(0, 10, 500, 500, &w, &h);
    EXPECT_EQ(0, w);
    EXPECT_EQ(0, h);
}

int main()
{
    testGalleryMetricsMatchTheSpec();
    testGalleryRowsThatFit();
    testGalleryPageBarTakesRoomOnlyWhenPaging();
    testGalleryClampsColumnsAndRows();
    testGalleryPageOfSlot();
    testViewerZones();
    testViewerNavigationHasNoWrap();
    testNeighbourAfterRemoval();
    testViewerAreas();
    testViewerBackButtonInFullView();
    testFitInside();
    testCharsPerLine();
    testCodePoints();
    testWrap();
    testWrapTruncatesToTheBuffer();
    testElide();
    testPackPages();
    testLayoutBody();
    testComponentSizes();
    testListAndValues();
    testQrPlacement();
    return finishTests("screen_logic");
}
