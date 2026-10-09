#ifndef NICKELDRAWER_NAVIGATION_H
#define NICKELDRAWER_NAVIGATION_H

#include <QString>

#include "navigation_stack.h"

// The one back stack of the mod (navigation_stack.h), with every push and pop
// written to the debug log. GUI thread only.
//   DrawerNavigator: open → More, drawer; back → pop; Nickel closed the page
//                    (tab tap, Home) → clear
//   DrawerSession:   app screen opened → push (records the drawer's view
//                    mode and page); screen closed → back to the drawer level,
//                    restoring them
//   ScreenSession:   header "push" → sub-screen (records the parent's page);
//                    back → pop, the parent is reloaded at its page
namespace Navigation {

bool push(Level level, const QString &id, const ViewState &current);
// parent: the new top with its recorded state (may be NULL).
bool pop(Entry *parent);
bool popTo(Level level, Entry *restored);
void clear(const char *reason);
const Entry *top();

ViewState stateOf(int viewMode, int page);

} // namespace Navigation

#endif
