// Extras: a screenshot button in Nickel's status bar and a Tweaks row (with its panel) in the More
// tab. A separate NickelHook mod with its own library, flags, folder and failsafe; it needs no other
// mod (README.md, helpers/README.md). init() stays tiny: read the flag files and .kobo/version.
// All UI work happens later on Nickel's GUI thread, in the hooks.

#include <NickelHook.h>

#include "debug_log.h"
#include "helper_flags.h"
#include "nickel_bridge.h"
#include "press_area.h"
#include "screenshot_button.h"
#include "system_icon.h"
#include "tweaks_row.h"

static const int FAILSAFE_DELAY_SECONDS = 15;

static int initialize()
{
    DebugLog::enableIfFlagged("extras", HelperFlags::DEBUG_PATH, HelperFlags::DEBUG_LOG_PATH);
    SystemIcon::setIconDirectory(HelperFlags::ICONS_DIRECTORY);
    if (HelperFlags::hasFlag(HelperFlags::DISABLE_PATH)) {
        // The hooks stay installed but pass straight through to Nickel.
        nh_log("kill switch %s present, staying inert", HelperFlags::DISABLE_PATH);
        return 0;
    }
    if (!NickelBridge::activate()) {
        return 0;
    }
    if (NickelBridge::isButtonActive()) {
        NickelBridge::setStatusBarBuiltHandler(ScreenshotButton::attachTo);
    }
    if (NickelBridge::isTweaksActive()) {
        NickelBridge::setMoreViewSetupHandler(TweaksRow::addTo);
        PressArea::setTapGestureTypeProvider(NickelBridge::tapGestureType);
        PressArea::setTapPositionProvider(NickelBridge::tapGesturePosition);
    }
    nh_log("active: screenshot button %s, tweaks %s", NickelBridge::isButtonActive() ? "on" : "off",
           NickelBridge::isTweaksActive() ? "on" : "off");
    return 0;
}

static struct nh_info modInfo = {
    .name = "extras",
    .desc = "Extras: screenshot button in the status bar and a Tweaks row in the More tab",
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
