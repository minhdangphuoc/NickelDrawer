// nddrawer: gate-G2 on-device prototypes for NickelDrawer.
//   Prototype A (flag proto-a): "Apps" row in the More tab.
//   Prototype C (flag proto-c): empty drawer page opened from that row.
// init() only reads the firmware and the flag files; the work happens later in
// the MoreView constructor hook and the row's tap slot, on Nickel's GUI thread.

#include <NickelHook.h>

#include "device_state.h"
#include "nickel_api.h"

static const char *const FLAG_PROTOTYPE_A = "/mnt/onboard/.adds/nickeldrawer/proto-a";
static const char *const FLAG_PROTOTYPE_C = "/mnt/onboard/.adds/nickeldrawer/proto-c";
static const char *const FLAG_PROTOTYPE_C_VIEW_PUSH = "/mnt/onboard/.adds/nickeldrawer/proto-c-pushview";

static void decidePrototypeA()
{
    bool isRequested = DeviceState::hasFlagFile(FLAG_PROTOTYPE_A);
    bool isHooked = MoreView_construct_original != NULL;
    prototypeSwitches.isMoreRowEnabled = isRequested && isHooked && NickelApi::hasIconLeftButton();
    if (isRequested && !prototypeSwitches.isMoreRowEnabled) {
        nh_log("prototype A disabled: %s", isHooked ? "IconLeftButton symbols missing" : "MoreView hook not installed");
    }
}

static void decidePrototypeC()
{
    bool isRequested = DeviceState::hasFlagFile(FLAG_PROTOTYPE_C);
    prototypeSwitches.usesViewPushRoute = DeviceState::hasFlagFile(FLAG_PROTOTYPE_C_VIEW_PUSH);
    bool hasRouteApi = prototypeSwitches.usesViewPushRoute ? NickelApi::hasViewPush() : NickelApi::hasControllerPush();
    prototypeSwitches.isDrawerEnabled = isRequested && hasRouteApi;
    if (isRequested && !hasRouteApi) {
        nh_log("prototype C disabled: push symbols for route %s missing",
               prototypeSwitches.usesViewPushRoute ? "B" : "A");
    }
    if (prototypeSwitches.isDrawerEnabled && !prototypeSwitches.isMoreRowEnabled) {
        nh_log("prototype C is enabled but only opens from the prototype A row; enable proto-a too");
    }
}

static int initialize()
{
    char firmware[32];
    DeviceState::readFirmware(firmware, sizeof(firmware));
    if (!DeviceState::isSupportedFirmware(firmware)) {
        // Hooks stay installed but pass through; nothing of ours runs.
        nh_log("firmware %s not in the tested list, all prototypes disabled", firmware);
        return 0;
    }

    decidePrototypeA();
    decidePrototypeC();
    nh_log("loaded on %s: prototype A %s, prototype C %s (route %s)", firmware,
           prototypeSwitches.isMoreRowEnabled ? "on" : "off", prototypeSwitches.isDrawerEnabled ? "on" : "off",
           prototypeSwitches.usesViewPushRoute ? "B pushView" : "A controller");
    return 0;
}

static struct nh_info modInfo = {
    .name = "nddrawer",
    .desc = "NickelDrawer gate-G2 prototypes (More row, drawer page)",
    .uninstall_flag = "/mnt/onboard/.adds/nickeldrawer/uninstall-proto",
    .uninstall_xflag = NULL, // g++ 4.9 rejects skipped designated fields; NULL: no delete-to-uninstall
    .failsafe_delay = 15,
};

// Every symbol is optional: a missing one disables the prototype that needs it
// (checked in initialize() and again at use), never the whole of Nickel.
static struct nh_hook modHooks[] = {
    {.sym = "_ZN8MoreViewC1EP7QWidget",
     .sym_new = "nddrawer_MoreView_construct",
     .lib = "libnickel.so.1.0.0",
     .out = nh_symoutptr(MoreView_construct_original),
     .desc = "MoreView::MoreView(QWidget*) (prototype A)",
     .optional = true},
    {0},
};

static struct nh_dlsym modSymbols[] = {
    {.name = "_ZN20MainWindowController14sharedInstanceEv",
     .out = nh_symoutptr(MainWindowController_sharedInstance),
     .desc = "MainWindowController::sharedInstance",
     .optional = true},
    {.name = "_ZN20MainWindowController5toastERK7QStringS2_i",
     .out = nh_symoutptr(MainWindowController_toast),
     .desc = "MainWindowController::toast",
     .optional = true},
    {.name = "_ZN20MainWindowController4pushEP18AbstractControllerb",
     .out = nh_symoutptr(MainWindowController_push),
     .desc = "MainWindowController::push (prototype C route A)",
     .optional = true},
    {.name = "_ZN20MainWindowController3popEP18AbstractController",
     .out = nh_symoutptr(MainWindowController_pop),
     .desc = "MainWindowController::pop (prototype C route A)",
     .optional = true},
    {.name = "_ZN20MainWindowController8pushViewEP7QWidget",
     .out = nh_symoutptr(MainWindowController_pushView),
     .desc = "MainWindowController::pushView (prototype C route B)",
     .optional = true},
    {.name = "_ZN20MainWindowController7popViewEP7QWidget",
     .out = nh_symoutptr(MainWindowController_popView),
     .desc = "MainWindowController::popView (prototype C route B)",
     .optional = true},
    {.name = "_ZN14IconLeftButtonC1EP7QWidget",
     .out = nh_symoutptr(IconLeftButton_construct),
     .desc = "IconLeftButton::IconLeftButton (prototype A)",
     .optional = true},
    {.name = "_ZN14IconLeftButton9setPixmapERK7QPixmap",
     .out = nh_symoutptr(IconLeftButton_setPixmap),
     .desc = "IconLeftButton::setPixmap (prototype A)",
     .optional = true},
    {.name = "_ZN14IconLeftButton7setTextERK7QString",
     .out = nh_symoutptr(IconLeftButton_setText),
     .desc = "IconLeftButton::setText (prototype A)",
     .optional = true},
    {0},
};

NickelHook(
    .init = initialize,
    .info = &modInfo,
    .hook = modHooks,
    .dlsym = modSymbols,
    .uninstall = NULL, )
