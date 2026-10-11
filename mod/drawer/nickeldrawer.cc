// NickelDrawer: native app drawer for Nickel (1264x1680 Kobo models, firmware 4.38.x).
// init() stays tiny: read the flag files and .kobo/version. All UI work
// happens later on Nickel's GUI thread (More-tab hook, row tap).

#include <NickelHook.h>

#include "debug_log.h"
#include "entry_row.h"
#include "mod_switches.h"
#include "nickel_bridge.h"
#include "press_area.h"
#include "system_icon.h"
#include "wake_rotation.h"

static const int FAILSAFE_DELAY_SECONDS = 15;
static const char ICON_DIRECTORY[] = "/mnt/onboard/.adds/koboapps/icons/ui";

// Our More-tab row (Apps) and the wake-rotation watcher.
static void addMoreRows(QWidget *moreView)
{
    EntryRow::addTo(moreView);
    WakeRotation::install();
}

static int initialize()
{
    DebugLog::enableIfFlagged("nickeldrawer", ModSwitches::DEBUG_FLAG_PATH, ModSwitches::DEBUG_LOG_PATH);
    SystemIcon::setIconDirectory(ICON_DIRECTORY);
    ModSwitches::readAtBoot();
    if (ModSwitches::isKillSwitchOn()) {
        // Hooks stay installed but pass straight through to Nickel.
        nh_log("kill switch %s present, staying inert", ModSwitches::DISABLE_FLAG_PATH);
        return 0;
    }
    if (!NickelBridge::activate()) {
        return 0;
    }
    NickelBridge::setMoreViewCreatedHandler(addMoreRows);
    PressArea::setTapGestureTypeProvider(NickelBridge::tapGestureType);
    PressArea::setTapPositionProvider(NickelBridge::tapGesturePosition);
    WakeRotation::scheduleInstall();
    nh_log("active on %s (route %s, app screens %s)", NickelBridge::firmware(),
           ModSwitches::usesViewPushRoute() ? "B pushView" : "A controller", ModSwitches::usesScreens() ? "on" : "off");
    return 0;
}

static struct nh_info modInfo = {
    .name = "nickeldrawer",
    .desc = "NickelDrawer: Apps row in the More tab, a native app drawer",
    .uninstall_flag = ModSwitches::UNINSTALL_FLAG_PATH,
    .uninstall_xflag = NULL, // g++ 4.9 rejects skipped designated fields
    .failsafe_delay = FAILSAFE_DELAY_SECONDS,
};

NickelHook(
    .init = initialize,
    .info = &modInfo,
    .hook = NickelBridge::hookTable,
    .dlsym = NickelBridge::symbolTable,
    .uninstall = NULL, )
