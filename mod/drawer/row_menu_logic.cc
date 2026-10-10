#include "row_menu_logic.h"

#include <stddef.h>

namespace RowMenuLogic {

EntryVerdict judgeEntry(const char *label, bool hasTarget, int keptCount, int maxItems)
{
    if (label == NULL || label[0] == '\0') {
        return ENTRY_NO_LABEL;
    }
    if (!hasTarget) {
        return ENTRY_NO_TARGET;
    }
    if (maxItems != NO_LIMIT && keptCount >= maxItems) {
        return ENTRY_TOO_MANY;
    }
    return ENTRY_KEPT;
}

bool opensMenuOnLongPress(int menuItemCount, bool isMenuOpen, bool isActionRunning)
{
    return menuItemCount > 0 && !isMenuOpen && !isActionRunning;
}

bool isRowTarget(bool hasTapAction, int menuItemCount)
{
    return hasTapAction || menuItemCount > 0;
}

} // namespace RowMenuLogic
