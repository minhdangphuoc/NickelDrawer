#include "mod_switches.h"

#include <unistd.h>

namespace ModSwitches {

static bool isKillSwitchFlagged = false;
static bool isViewPushFlagged = false;
static bool isNoScreensFlagged = false;
static bool isLegacyScreensFlagged = false;
static bool isUnknownFirmwareTestFlagged = false;
static bool isUnknownModelTestFlagged = false;
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
    return !isNoScreensFlagged;
}

bool hasLegacyScreensFlag()
{
    return isLegacyScreensFlagged;
}

bool isUnknownFirmwareForced()
{
    return isUnknownFirmwareTestFlagged;
}

bool isUnknownModelForced()
{
    return isUnknownModelTestFlagged;
}

bool usesNickelKeyboard()
{
    return !isNoKeyboardFlagged;
}

void readAtBoot()
{
    isKillSwitchFlagged = hasFlag(DISABLE_FLAG_PATH);
    isViewPushFlagged = hasFlag(VIEW_PUSH_FLAG_PATH);
    isNoScreensFlagged = hasFlag(NO_SCREENS_FLAG_PATH);
    isLegacyScreensFlagged = hasFlag(LEGACY_SCREENS_FLAG_PATH);
    isUnknownFirmwareTestFlagged = hasFlag(TEST_UNKNOWN_FIRMWARE_FLAG_PATH);
    isUnknownModelTestFlagged = hasFlag(TEST_UNKNOWN_MODEL_FLAG_PATH);
    isNoKeyboardFlagged = hasFlag(NO_KEYBOARD_FLAG_PATH);
}

} // namespace ModSwitches
