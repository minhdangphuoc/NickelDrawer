#ifndef NICKELDRAWER_ROW_MENU_LOGIC_H
#define NICKELDRAWER_ROW_MENU_LOGIC_H

// Pure rules of long-press menus on List rows (docs/spec/ui-kit.md §3.16,
// ui-json.md "list"): which declared entries the mod keeps, and when a long
// press opens the menu. No Qt, no libnickel, no C++ standard library, so
// tests/row_menu_logic_test.cc builds it with the host compiler. The JSON
// itself is read by model_parser.cc (QJsonDocument), which asks these rules.

namespace RowMenuLogic {

// ui-json.md: a list's `menu` has 1 to 6 items.
static const int MAX_LIST_ITEMS = 6;
static const int NO_LIMIT = 0;

enum EntryVerdict { ENTRY_KEPT, ENTRY_NO_LABEL, ENTRY_NO_TARGET, ENTRY_TOO_MANY };

// A declared entry needs a label and something to do: a list entry has an
// `id` (sent back in the menu event); a gallery entry an action, a push or
// `info`. keptCount: entries already kept; maxItems: NO_LIMIT or a cap.
EntryVerdict judgeEntry(const char *label, bool hasTarget, int keptCount, int maxItems);

// Whether a held row opens its menu: the row declares one, no menu or action
// is already open, and no action is running.
bool opensMenuOnLongPress(int menuItemCount, bool isMenuOpen, bool isActionRunning);

// A list row is a tap target when it has a tap action or a menu (the menu
// needs the press to be tracked even when a short tap does nothing).
bool isRowTarget(bool hasTapAction, int menuItemCount);

} // namespace RowMenuLogic

#endif
