#include "navigation.h"

#include <QByteArray>

#include "debug_log.h"

namespace Navigation {

static Stack &sharedStack()
{
    static Stack stack;
    return stack;
}

ViewState stateOf(int viewMode, int page)
{
    ViewState state;
    state.viewMode = viewMode;
    state.page = page;
    return state;
}

static void logDepth(const char *event)
{
    const Entry *entry = sharedStack().top();
    DebugLog::write("navigation: %s, depth %d, top %s \"%s\" (mode %d, page %d)", event, sharedStack().depth(),
                    entry ? levelName(entry->level) : "none", entry ? entry->id : "", entry ? entry->state.viewMode : 0,
                    entry ? entry->state.page + 1 : 0);
}

bool push(Level level, const QString &id, const ViewState &current)
{
    QByteArray utf8 = id.toUtf8();
    if (!sharedStack().push(level, utf8.constData(), current)) {
        bool isTooDeep = level == LEVEL_SUB_SCREEN && sharedStack().countOf(LEVEL_SUB_SCREEN) >= MAX_SUB_SCREENS;
        DebugLog::write("navigation: push %s \"%s\" refused: %s (depth %d, top %s)", levelName(level), utf8.constData(),
                        isTooDeep ? "already 2 screens below main" : "does not fit on the stack", sharedStack().depth(),
                        sharedStack().top() ? levelName(sharedStack().top()->level) : "none");
        return false;
    }
    DebugLog::write("navigation: push %s \"%s\", parent recorded at mode %d, page %d", levelName(level),
                    utf8.constData(), current.viewMode, current.page + 1);
    logDepth("after push");
    return true;
}

bool pop(Entry *parent)
{
    const Entry *leaving = sharedStack().top();
    const char *leavingName = leaving ? levelName(leaving->level) : "none";
    if (!sharedStack().pop(parent)) {
        DebugLog::write("navigation: pop refused (depth %d, top %s)", sharedStack().depth(), leavingName);
        return false;
    }
    DebugLog::write("navigation: pop %s", leavingName);
    logDepth("after pop");
    return true;
}

bool popTo(Level level, Entry *restored)
{
    if (!sharedStack().popTo(level, restored)) {
        DebugLog::write("navigation: pop to %s refused: not in the stack (depth %d)", levelName(level),
                        sharedStack().depth());
        return false;
    }
    logDepth("after pop to");
    return true;
}

void clear(const char *reason)
{
    int screens = sharedStack().countOf(LEVEL_APP_SCREEN) + sharedStack().countOf(LEVEL_SUB_SCREEN) +
                  sharedStack().countOf(LEVEL_MODAL);
    int dropped = sharedStack().clear();
    DebugLog::write("navigation: cleared %d level(s), %d of them screens or modals, with their saved pages: %s",
                    dropped, screens, reason);
}

const Entry *top()
{
    return sharedStack().top();
}

} // namespace Navigation
