// Screenshot button in Nickel's status bar: a personal helper for the maintainer's own Kobo, a
// separate NickelHook mod with its own library, flags and failsafe (README.md, and the design in
// re/notes/statusbar-screenshot.md). init() stays tiny: read the flag files and .kobo/version.
// All UI work happens later on Nickel's GUI thread, in the StatusBarView hook.

#include <NickelHook.h>

#include "helper_flags.h"
#include "helper_log.h"
#include "nickel_bridge.h"
#include "screenshot_button.h"

static const int FAILSAFE_DELAY_SECONDS = 15;

static int initialize()
{
    HelperLog::enableIfFlagged();
    if (HelperFlags::hasFlag(HelperFlags::DISABLE_PATH)) {
        // The hook stays installed but passes straight through to Nickel.
        nh_log("kill switch %s present, staying inert", HelperFlags::DISABLE_PATH);
        return 0;
    }
    if (NickelBridge::activate()) {
        NickelBridge::setStatusBarBuiltHandler(ScreenshotButton::attachTo);
        nh_log("active");
    }
    return 0;
}

static struct nh_info modInfo = {
    .name = "statusbarshot",
    .desc = "Screenshot button in the status bar (maintainer's helper)",
    .uninstall_flag = HelperFlags::UNINSTALL_PATH,
    .uninstall_xflag = NULL, // g++ 4.9 rejects skipped designated fields
    .failsafe_delay = FAILSAFE_DELAY_SECONDS,
};

NickelHook(
    .init = initialize,
    .info = &modInfo,
    .hook = NickelBridge::hookTable,
    .dlsym = NickelBridge::symbolTable,
    .uninstall = NULL, )
