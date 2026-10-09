#include "mod_switches.h"

#include <unistd.h>

namespace ModSwitches {

static bool isKillSwitchFlagged = false;
static bool isViewPushFlagged = false;

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

void readAtBoot()
{
    isKillSwitchFlagged = hasFlag(DISABLE_FLAG_PATH);
    isViewPushFlagged = hasFlag(VIEW_PUSH_FLAG_PATH);
}

} // namespace ModSwitches
