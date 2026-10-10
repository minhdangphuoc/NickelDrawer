#ifndef NICKELDRAWER_MOD_SWITCHES_H
#define NICKELDRAWER_MOD_SWITCHES_H

// Flag files the user controls over USB, read once in init() (reboot to apply).
// Paths live here only (arrays, so nh_info can use them in a static initializer).

namespace ModSwitches {

// NickelHook deletes the mod (and this flag) when it exists at boot.
static const char UNINSTALL_FLAG_PATH[] = "/mnt/onboard/.adds/nickeldrawer/uninstall";
// Kill switch: the mod loads but the bridge stays inert (no row, no hooks run).
static const char DISABLE_FLAG_PATH[] = "/mnt/onboard/.adds/nickeldrawer/disable";
// Opens the drawer with pushView/popView (route B) instead of the controller push.
static const char VIEW_PUSH_FLAG_PATH[] = "/mnt/onboard/.adds/nickeldrawer/pushview";
// Opt-out: tapping an app opens its primary action instead of its ui.json screen.
// Screens are on by default.
static const char NO_SCREENS_FLAG_PATH[] = "/mnt/onboard/.adds/nickeldrawer/no-screens";
// Old opt-in flag from before screens were on by default: still read, no longer needed.
static const char LEGACY_SCREENS_FLAG_PATH[] = "/mnt/onboard/.adds/nickeldrawer/screens";
// Test hook (FI-04): the firmware check fails as if the build were unknown.
static const char TEST_UNKNOWN_FIRMWARE_FLAG_PATH[] = "/mnt/onboard/.adds/nickeldrawer/test-unknown-fw";
// Fields show the "edit on your phone" notice instead of Nickel's keyboard.
static const char NO_KEYBOARD_FLAG_PATH[] = "/mnt/onboard/.adds/nickeldrawer/no-keyboard";

bool hasFlag(const char *path);

// Decided in init().
bool isKillSwitchOn();
bool usesViewPushRoute();
bool usesScreens();
bool hasLegacyScreensFlag();
bool isUnknownFirmwareForced();
bool usesNickelKeyboard();
void readAtBoot();

} // namespace ModSwitches

#endif
