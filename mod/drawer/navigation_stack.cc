#include "navigation_stack.h"

namespace Navigation {

static void copyText(char *target, const char *source, int capacity)
{
    int i = 0;
    for (; source && source[i] && i < capacity - 1; i++) {
        target[i] = source[i];
    }
    target[i] = '\0';
}

// Sub-screens and modals stack on each other; every other level appears once,
// below anything deeper.
static bool fitsOver(Level child, Level parent)
{
    bool isRepeatable = child == LEVEL_SUB_SCREEN || child == LEVEL_MODAL;
    return child > parent || (isRepeatable && parent >= LEVEL_SUB_SCREEN);
}

Stack::Stack() : count(0) {}

const Entry *Stack::top() const
{
    return count > 0 ? &entries[count - 1] : 0;
}

const Entry *Stack::at(int index) const
{
    return index >= 0 && index < count ? &entries[index] : 0;
}

bool Stack::push(Level level, const char *id, const ViewState &current, const char *value)
{
    if (count >= MAX_DEPTH) {
        return false;
    }
    if (count > 0 && !fitsOver(level, entries[count - 1].level)) {
        return false;
    }
    if (level == LEVEL_SUB_SCREEN && countOf(LEVEL_SUB_SCREEN) >= MAX_SUB_SCREENS) {
        return false;
    }
    if (count > 0) {
        entries[count - 1].state = current;
    }
    Entry &entry = entries[count];
    entry.level = level;
    copyText(entry.id, id, MAX_ID_BYTES);
    copyText(entry.value, value, MAX_VALUE_BYTES);
    entry.state.viewMode = 0;
    entry.state.page = 0;
    count++;
    return true;
}

bool Stack::setTopValue(const char *value)
{
    if (count == 0) {
        return false;
    }
    copyText(entries[count - 1].value, value, MAX_VALUE_BYTES);
    return true;
}

bool Stack::pop(Entry *parent)
{
    if (count <= 1) {
        return false;
    }
    count--;
    if (parent) {
        *parent = entries[count - 1];
    }
    return true;
}

bool Stack::popTo(Level level, Entry *restored)
{
    int index = count - 1;
    while (index >= 0 && entries[index].level != level) {
        index--;
    }
    if (index < 0) {
        return false;
    }
    count = index + 1;
    if (restored) {
        *restored = entries[index];
    }
    return true;
}

int Stack::countOf(Level level) const
{
    int found = 0;
    for (int i = 0; i < count; i++) {
        found += entries[i].level == level ? 1 : 0;
    }
    return found;
}

int Stack::clear()
{
    int dropped = count;
    count = 0;
    return dropped;
}

const char *levelName(Level level)
{
    switch (level) {
    case LEVEL_MORE:
        return "More";
    case LEVEL_DRAWER:
        return "drawer";
    case LEVEL_APP_SCREEN:
        return "app screen";
    case LEVEL_SUB_SCREEN:
        return "sub-screen";
    case LEVEL_MODAL:
        return "modal";
    }
    return "unknown";
}

} // namespace Navigation
