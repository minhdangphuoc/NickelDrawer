#ifndef STATUSBARSHOT_HELPER_FLAGS_H
#define STATUSBARSHOT_HELPER_FLAGS_H

// The helper's flag files on the user partition, all under one folder. A flag is read with a single
// access() call, never held open and never watched: inotify on the FAT partition is unreliable and the
// partition is gone while USB is connected. No Qt, no libnickel, no C++ standard library (host-tested).

namespace HelperFlags {

// Paths are arrays so nh_info can use them in a static initializer.
static const char FOLDER[] = "/mnt/onboard/.adds/helpers/screenshot";
// NickelHook deletes the helper (and this flag) when it exists at boot.
static const char UNINSTALL_PATH[] = "/mnt/onboard/.adds/helpers/screenshot/uninstall";
// Kill switch: the hook passes straight through to Nickel.
static const char DISABLE_PATH[] = "/mnt/onboard/.adds/helpers/screenshot/disable";
// Verbose lines in the syslog (read once at boot).
static const char DEBUG_PATH[] = "/mnt/onboard/.adds/helpers/screenshot/debug";
// The user's switch: present = no button, absent = button (the default). NickelDrawer's Tweaks writes it.
static const char OFF_PATH[] = "/mnt/onboard/.adds/helpers/screenshot/off";

bool hasFlag(const char *path);

// What the flag files say about the button right now. UNKNOWN when the helper's folder cannot be seen
// (USB connected, partition busy): the caller keeps what it shows.
enum ButtonState { BUTTON_ON, BUTTON_OFF, BUTTON_UNKNOWN };

ButtonState readButtonState(const char *offFlagPath, const char *folderPath);
ButtonState readButtonState();

} // namespace HelperFlags

#endif
