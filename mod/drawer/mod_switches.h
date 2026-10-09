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
// Opens an app's ui.json screen on tap (Phase 4). Off by default until gate G2 passes.
static const char SCREENS_FLAG_PATH[] = "/mnt/onboard/.adds/nickeldrawer/screens";
// Fields show the "edit on your phone" notice instead of Nickel's keyboard.
static const char NO_KEYBOARD_FLAG_PATH[] = "/mnt/onboard/.adds/nickeldrawer/no-keyboard";

bool hasFlag(const char *path);

// Decided in init().
bool isKillSwitchOn();
bool usesViewPushRoute();
bool usesScreens();
bool usesNickelKeyboard();
void readAtBoot();

} // namespace ModSwitches

#endif
