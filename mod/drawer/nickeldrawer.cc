// NickelDrawer: native app drawer for Nickel (Kobo Libra 2, firmware 4.38.x).
// init() stays tiny: read two flag files and the firmware version. All UI work
// happens later on Nickel's GUI thread (More-tab hook, row tap).

#include <NickelHook.h>

#include "entry_row.h"
#include "mod_switches.h"
#include "nickel_bridge.h"

static const int FAILSAFE_DELAY_SECONDS = 15;

static int initialize()
{
    ModSwitches::readAtBoot();
    if (ModSwitches::isKillSwitchOn()) {
        // Hooks stay installed but pass straight through to Nickel.
        nh_log("kill switch %s present, staying inert", ModSwitches::DISABLE_FLAG_PATH);
        return 0;
    }
    if (!NickelBridge::activate()) {
        return 0;
    }
    NickelBridge::setMoreViewCreatedHandler(EntryRow::addTo);
    nh_log("active on %s (route %s)", NickelBridge::firmware(),
           ModSwitches::usesViewPushRoute() ? "B pushView" : "A controller");
    return 0;
}

static struct nh_info modInfo = {
    .name = "nickeldrawer",
    .desc = "NickelDrawer: Apps row in the More tab and a native app drawer",
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
