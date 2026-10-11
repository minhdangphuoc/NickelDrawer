#ifndef EXTRAS_HELPER_FLAGS_H
#define EXTRAS_HELPER_FLAGS_H

// The helper's files on the user partition, all under one folder. A flag is read with a single
// access() call, never held open and never watched: inotify on the FAT partition is unreliable and the
// partition is gone while USB is connected. No Qt, no libnickel, no C++ standard library (host-tested).
// Nothing here names another mod's folder: Extras and NickelDrawer each own theirs.

namespace HelperFlags {

// Paths are arrays so nh_info can use them in a static initializer.
static const char FOLDER[] = "/mnt/onboard/.adds/helpers/extras";
// NickelHook deletes the helper (and this flag) when it exists at boot.
static const char UNINSTALL_PATH[] = "/mnt/onboard/.adds/helpers/extras/uninstall";
// Kill switch: the hooks pass straight through to Nickel (read at boot).
static const char DISABLE_PATH[] = "/mnt/onboard/.adds/helpers/extras/disable";
// Verbose lines in the debug log and the syslog (read at boot).
static const char DEBUG_PATH[] = "/mnt/onboard/.adds/helpers/extras/debug";
static const char DEBUG_LOG_PATH[] = "/mnt/onboard/.adds/helpers/extras/debug.log";
// The user's switch for the status-bar button: present = no button, absent = button (the default).
// The Tweaks panel creates and removes it (helpers/extras/internal/tweaks); it can be done by hand too.
static const char SCREENSHOT_OFF_PATH[] = "/mnt/onboard/.adds/helpers/extras/screenshot-off";
// What the helper installs next to the library.
static const char TWEAKS_BINARY_PATH[] = "/mnt/onboard/.adds/helpers/extras/bin/extras-tweaks";
static const char LOCALES_DIRECTORY[] = "/mnt/onboard/.adds/helpers/extras/locales";
static const char ICONS_DIRECTORY[] = "/mnt/onboard/.adds/helpers/extras/icons";

bool hasFlag(const char *path);

// What the flag files say about the button right now. UNKNOWN when the helper's folder cannot be seen
// (USB connected, partition busy): the caller keeps what it shows.
enum ButtonState { BUTTON_ON, BUTTON_OFF, BUTTON_UNKNOWN };

ButtonState readButtonState(const char *offFlagPath, const char *folderPath);
ButtonState readButtonState();

} // namespace HelperFlags

#endif
