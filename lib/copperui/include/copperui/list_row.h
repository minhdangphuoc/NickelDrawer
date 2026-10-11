#ifndef COPPERUI_LIST_ROW_H
#define COPPERUI_LIST_ROW_H

#include "copperui/geometry.h"
#include "copperui/tokens.h"

// A row of a `list` (docs/design-system.md section 4.16): the primary text on the left, the
// secondary text as a muted caption at the right edge. It is its own component, not a key-value
// row: the primary text takes the whole row width less the secondary text (a file name needs it),
// and a file name is elided in the middle.

namespace copperui {
namespace v1 {

// The secondary text "✓" (U+2713) is drawn as the Tabler check icon, not as a glyph (A13).
bool isCheckMarkText(const char *utf8);

// What a row draws besides its primary text. In a plain list the secondary text "✓" is the check
// icon at the right edge. In a checklist (ui.json `style: "checklist"`) the same "✓" means the
// item is done: it is drawn as a ticked box at the left, never at the right; any other secondary
// text stays a caption.
struct ListRowMarks {
    bool hasCheckbox;     // a box at the left of the primary text
    bool isChecked;       // the box is ticked (checklist only)
    bool hasCheckMark;    // the check icon at the right edge (plain list only)
    bool showsSecondary;  // the secondary text is drawn as a caption
};
ListRowMarks listRowMarks(bool isChecklist, const char *secondaryUtf8);

struct ListRowGeometry {
    Rect checkbox;    // ICON_CHECKBOX square at the left edge; zero width unless the row has a box
    Rect primary;
    Rect secondary;   // zero width when the row has no secondary text
    Rect checkIcon;   // ICON_CHEVRON square at the right edge; zero width unless the mark is a check
};

// rowTop: y of the row. secondaryWidth: the measured width of the secondary caption (0: none);
// it is capped at LIST_ROW_SECONDARY_MAX. hasCheckMark: the secondary text is the check mark.
// hasCheckbox: the row starts with a box and the primary text moves right of it.
ListRowGeometry listRowGeometry(int rowTop, int secondaryWidth, bool hasCheckMark, bool hasCheckbox);

} // namespace v1
} // namespace copperui

#endif
