#include "mod_switches.h"

#include <unistd.h>

namespace ModSwitches {

static bool isKillSwitchFlagged = false;
static bool isViewPushFlagged = false;
static bool isScreensFlagged = false;
static bool isNoKeyboardFlagged = false;

bool hasFlag(const char *path)
{
    return access(path, F_OK) == 0;
}

bool isKillSwitchOn()
{
    return isKillSwitchFlagged;
}

bool usesViewPushRoute()
{
    return isViewPushFlagged;
}

bool usesScreens()
{
    return isScreensFlagged;
}

bool usesNickelKeyboard()
{
    return !isNoKeyboardFlagged;
}

void readAtBoot()
{
    isKillSwitchFlagged = hasFlag(DISABLE_FLAG_PATH);
    isViewPushFlagged = hasFlag(VIEW_PUSH_FLAG_PATH);
    isScreensFlagged = hasFlag(SCREENS_FLAG_PATH);
    isNoKeyboardFlagged = hasFlag(NO_KEYBOARD_FLAG_PATH);
}

} // namespace ModSwitches
