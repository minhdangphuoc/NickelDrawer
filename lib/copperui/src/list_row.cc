#include "copperui/list_row.h"

namespace copperui {
namespace v1 {

static const char CHECK_MARK_UTF8[] = "\xE2\x9C\x93"; // U+2713

bool isCheckMarkText(const char *utf8)
{
    for (int i = 0; i < 4; i++) {
        if (utf8[i] != CHECK_MARK_UTF8[i]) {
            return false;
        }
        if (CHECK_MARK_UTF8[i] == '\0') {
            return true;
        }
    }
    return false;
}

ListRowMarks listRowMarks(bool isChecklist, const char *secondaryUtf8)
{
    bool isCheckMark = isCheckMarkText(secondaryUtf8);
    ListRowMarks marks;
    marks.hasCheckbox = isChecklist;
    marks.isChecked = isChecklist && isCheckMark;
    marks.hasCheckMark = !isChecklist && isCheckMark;
    marks.showsSecondary = !isCheckMark && secondaryUtf8[0] != '\0';
    return marks;
}

static Rect rectAt(int x, int y, int width, int height)
{
    Rect rect = {x, y, width, height};
    return rect;
}

ListRowGeometry listRowGeometry(int rowTop, int secondaryWidth, bool hasCheckMark, bool hasCheckbox)
{
    int right = GUTTER + CONTENT_WIDTH;
    ListRowGeometry geometry;
    geometry.secondary = rectAt(right, rowTop + (ROW_COMPACT - TYPE_CAPTION_LINE) / 2, 0, TYPE_CAPTION_LINE);
    geometry.checkIcon = rectAt(right, rowTop + (ROW_COMPACT - ICON_CHEVRON) / 2, 0, ICON_CHEVRON);
    geometry.checkbox = rectAt(GUTTER, rowTop + (ROW_COMPACT - ICON_CHECKBOX) / 2, 0, ICON_CHECKBOX);
    int left = GUTTER;
    if (hasCheckbox) {
        geometry.checkbox.width = ICON_CHECKBOX;
        left += ICON_CHECKBOX + RHYTHM;
    }
    int reserved = 0;
    if (hasCheckMark) {
        geometry.checkIcon.x = right - ICON_CHEVRON;
        geometry.checkIcon.width = ICON_CHEVRON;
        reserved = ICON_CHEVRON + RHYTHM;
    } else if (secondaryWidth > 0) {
        int width = secondaryWidth < LIST_ROW_SECONDARY_MAX ? secondaryWidth : LIST_ROW_SECONDARY_MAX;
        geometry.secondary.x = right - width;
        geometry.secondary.width = width;
        reserved = width + RHYTHM;
    }
    geometry.primary = rectAt(left, rowTop + SPACE_ROW_TEXT, GUTTER + CONTENT_WIDTH - left - reserved, ROW_LABEL_HEIGHT);
    return geometry;
}

} // namespace v1
} // namespace copperui
