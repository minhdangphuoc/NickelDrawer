#ifndef NICKELDRAWER_NAVIGATION_STACK_H
#define NICKELDRAWER_NAVIGATION_STACK_H

// The back stack of the drawer: More → drawer → app screen → sub-screen or
// modal. "←" pops one level and returns to the view the user came from, with
// the state it had (view mode and page). Each push records the current top's
// state first; each pop hands back the parent with that state. Nickel's tab
// bar (or Home) closes everything at once: clear().
//
// Pure logic, no Qt, no C++ standard library: tests/navigation_stack_test.cc
// builds it with the host compiler. navigation.{h,cc} is the logged Qt glue.

namespace Navigation {

enum Level {
    LEVEL_MORE,       // Nickel's More tab, the root
    LEVEL_DRAWER,     // the app drawer page
    LEVEL_APP_SCREEN, // an app's start screen over the drawer
    LEVEL_SUB_SCREEN, // a screen pushed from an app screen (ui.json header "push")
    LEVEL_MODAL,      // an options window over a screen
};

static const int MAX_DEPTH = 8;
// Screens stacked above an app's main screen (main → sub → detail); the same
// limit as appsctl validate (internal/uikit MaxPushDepth).
static const int MAX_SUB_SCREENS = 2;
static const int MAX_ID_BYTES = 64; // app or screen id, NUL included

// What a view needs to look the same when the user comes back to it.
struct ViewState {
    int viewMode; // drawer: DrawerLogic::ViewMode; screens: 0
    int page;     // 0-based
};

struct Entry {
    Level level;
    char id[MAX_ID_BYTES]; // "" for More and the drawer
    ViewState state;       // recorded when something was pushed over it
};

class Stack {
public:
    Stack();

    int depth() const { return count; }
    int countOf(Level level) const;
    bool isEmpty() const { return count == 0; }
    // NULL when empty.
    const Entry *top() const;
    const Entry *at(int index) const;

    // Records current (the top's state as it is now) on the top, then pushes
    // a new level with a zero state. A child level must be deeper than the
    // top (More < drawer < app screen < sub-screen/modal; sub-screens and
    // modals may stack). False, and nothing changed, when the stack is full
    // or the level does not fit, or it would be a third sub-screen.
    bool push(Level level, const char *id, const ViewState &current);
    // Removes the top; parent (may be NULL) receives the new top with the
    // state recorded when the removed level was pushed. False when the stack
    // is empty or only the root is left (back from the root is Nickel's).
    bool pop(Entry *parent);
    // Pops every level above the topmost entry of level, so that one is the
    // top (back from a screen closes its sub-screens too). False when absent.
    bool popTo(Level level, Entry *restored);
    // A tab tap or Home: Nickel discarded the whole page stack. Returns how
    // many levels were dropped.
    int clear();

private:
    Entry entries[MAX_DEPTH];
    int count;
};

// "drawer", "app screen", … for the log.
const char *levelName(Level level);

} // namespace Navigation

#endif
