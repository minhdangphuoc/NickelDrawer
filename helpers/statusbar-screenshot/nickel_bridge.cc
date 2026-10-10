#include "nickel_bridge.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include <QTimer>
#include <QWidget>

#include "device_gate.h"
#include "helper_flags.h"
#include "helper_log.h"
#include "helper_strings.h"

// ---- libnickel entry points (filled by NickelHook before init; NULL if absent) ----

extern "C" {
static void (*StatusBarIcon_construct)(void *icon, QWidget *parent);
static void (*TouchLabel_setDeselectedPixmap)(void *label, QPixmap const &pixmap);
static void (*TouchLabel_setSelectedPixmap)(void *label, QPixmap const &pixmap);
static void (*ScreenshotMaker_construct)(void *maker);
static void (*ScreenshotMaker_saveCurrentScreen)(void *maker);
// QString comes back through a hidden pointer, as for any member function returning a class.
static QString (*ScreenshotMaker_fileName)(void *maker);
static void *(*MainWindowController_sharedInstance)();
static void (*MainWindowController_toast)(void *window, QString const &primary, QString const &secondary,
                                          int milliseconds);
// Original StatusBarView::StatusBarView(QWidget*), returned by the PLT hook.
static void (*StatusBarView_construct_original)(void *statusBar, QWidget *parent);
}

// The tap signal is connected by name (old-style SIGNAL(), no dlsym); check-symbols.sh verifies it.
// signal-symbol: _ZN10TouchLabel6tappedEb

static const char *const VERSION_FILE_PATH = "/mnt/onboard/.kobo/version";

static const int TOAST_DURATION_MILLISECONDS = 2000;
static const int DISABLED_NOTICE_DELAY_MILLISECONDS = 10000;
// StatusBarIcon is 0xC0 bytes on all five builds; allocate more, zero-filled, so a slightly larger
// class still fits (the risk table of re/notes/statusbar-screenshot.md).
static const size_t STATUS_BAR_ICON_ALLOCATION = 0x100;
// ScreenshotMaker is 4 bytes ({int counter}); the constructor stores 1.
static const size_t SCREENSHOT_MAKER_ALLOCATION = 16;

static bool isBridgeActive = false;
// Set when the bridge disables itself (unknown model or firmware, missing symbol), not by the kill switch.
static bool isDisabledNoticePending = false;
static char firmwareVersion[32] = "unknown";
static char deviceId[64] = "unknown";
static void *screenshotMaker = NULL;
static NickelBridge::StatusBarBuiltHandler statusBarBuiltHandler = NULL;

// ---- NickelHook tables ----

namespace NickelBridge {

struct nh_hook hookTable[] = {
    {.sym = "_ZN13StatusBarViewC1EP7QWidget",
     .sym_new = "statusbarshot_StatusBarView_construct",
     .lib = "libnickel.so.1.0.0",
     .out = nh_symoutptr(StatusBarView_construct_original),
     .desc = "StatusBarView::StatusBarView(QWidget*) (screenshot button)",
     .optional = true},
    {0},
};

#define NICKEL_SYMBOL(mangled, variable, description)                                                    \
    {.name = mangled, .out = nh_symoutptr(variable), .desc = description, .optional = true}

struct nh_dlsym symbolTable[] = {
    NICKEL_SYMBOL("_ZN13StatusBarIconC1EP7QWidget", StatusBarIcon_construct, "button"),
    NICKEL_SYMBOL("_ZN10TouchLabel19setDeselectedPixmapERK7QPixmap", TouchLabel_setDeselectedPixmap, "button"),
    NICKEL_SYMBOL("_ZN10TouchLabel17setSelectedPixmapERK7QPixmap", TouchLabel_setSelectedPixmap, "button"),
    NICKEL_SYMBOL("_ZN15ScreenshotMakerC1Ev", ScreenshotMaker_construct, "capture"),
    NICKEL_SYMBOL("_ZN15ScreenshotMaker17saveCurrentScreenEv", ScreenshotMaker_saveCurrentScreen, "capture"),
    NICKEL_SYMBOL("_ZN15ScreenshotMaker8fileNameEv", ScreenshotMaker_fileName, "capture"),
    NICKEL_SYMBOL("_ZN20MainWindowController14sharedInstanceEv", MainWindowController_sharedInstance, "toast"),
    NICKEL_SYMBOL("_ZN20MainWindowController5toastERK7QStringS2_i", MainWindowController_toast, "toast"),
    {0},
};

#undef NICKEL_SYMBOL

} // namespace NickelBridge

// ---- Model, firmware and symbol checks ----

static void readVersionFile()
{
    if (!DeviceGate::readVersionFile(VERSION_FILE_PATH, firmwareVersion, sizeof(firmwareVersion), deviceId,
                                     sizeof(deviceId))) {
        nh_log("cannot read %s", VERSION_FILE_PATH);
    }
}

// Without these the button cannot exist or cannot capture. The toast is optional: without it the
// helper saves silently. NULL: all present.
static const char *missingCriticalSymbol()
{
    if (!StatusBarView_construct_original) {
        return "StatusBarView::StatusBarView(QWidget*)";
    }
    if (!StatusBarIcon_construct || !TouchLabel_setDeselectedPixmap || !TouchLabel_setSelectedPixmap) {
        return "StatusBarIcon / TouchLabel pixmap setters";
    }
    if (!ScreenshotMaker_construct || !ScreenshotMaker_saveCurrentScreen || !ScreenshotMaker_fileName) {
        return "ScreenshotMaker";
    }
    return NULL;
}

static bool canToast()
{
    return MainWindowController_sharedInstance && MainWindowController_toast;
}

static void disableBridge(const char *reason)
{
    isBridgeActive = false;
    isDisabledNoticePending = true;
    nh_log("%s, screenshot button disabled", reason);
}

static bool passesModelCheck()
{
    const DeviceGate::SupportedModel *model = DeviceGate::findSupportedModel(deviceId);
    if (!model) {
        char reason[160];
        snprintf(reason, sizeof(reason), "device %s is not a supported model", deviceId);
        disableBridge(reason);
        return false;
    }
    HelperLog::debug("device %s is model %s (%s), supported", deviceId, model->idSuffix, model->codename);
    return true;
}

static bool passesFirmwareCheck()
{
    if (!DeviceGate::isSupportedFirmware(firmwareVersion)) {
        char reason[160];
        snprintf(reason, sizeof(reason), "firmware %s not in the tested list", firmwareVersion);
        disableBridge(reason);
        return false;
    }
    return true;
}

namespace NickelBridge {

bool activate()
{
    readVersionFile();
    if (!passesModelCheck() || !passesFirmwareCheck()) {
        return false;
    }
    const char *missing = missingCriticalSymbol();
    if (missing) {
        char reason[160];
        snprintf(reason, sizeof(reason), "libnickel symbol %s missing", missing);
        disableBridge(reason);
        return false;
    }
    isBridgeActive = true;
    return true;
}

bool isActive()
{
    return isBridgeActive;
}

void setStatusBarBuiltHandler(StatusBarBuiltHandler handler)
{
    statusBarBuiltHandler = handler;
}

QWidget *createStatusBarIcon(QWidget *parent)
{
    if (!isBridgeActive) {
        return NULL;
    }
    // operator new and zero fill: Nickel deletes the icon with its virtual destructor, which frees
    // with operator delete.
    void *memory = ::operator new(STATUS_BAR_ICON_ALLOCATION);
    memset(memory, 0, STATUS_BAR_ICON_ALLOCATION);
    StatusBarIcon_construct(memory, parent);
    // StatusBarIcon is a TouchLabel is a QLabel: QWidget is its first base.
    return reinterpret_cast<QWidget *>(memory);
}

void setIconPixmaps(QWidget *icon, const QPixmap &normal, const QPixmap &pressed)
{
    if (!isBridgeActive || !icon) {
        return;
    }
    TouchLabel_setDeselectedPixmap(icon, normal);
    TouchLabel_setSelectedPixmap(icon, pressed);
}

static void *makerInstance()
{
    if (!screenshotMaker) {
        screenshotMaker = calloc(1, SCREENSHOT_MAKER_ALLOCATION);
        if (!screenshotMaker) {
            nh_log("out of memory for ScreenshotMaker");
            return NULL;
        }
        ScreenshotMaker_construct(screenshotMaker);
    }
    return screenshotMaker;
}

QString screenshotName()
{
    void *maker = isBridgeActive ? makerInstance() : NULL;
    return maker ? ScreenshotMaker_fileName(maker) : QString();
}

bool saveScreenshot()
{
    void *maker = isBridgeActive ? makerInstance() : NULL;
    if (!maker) {
        return false;
    }
    ScreenshotMaker_saveCurrentScreen(maker);
    return true;
}

void showToast(const QString &primary, const QString &secondary)
{
    void *window = canToast() ? MainWindowController_sharedInstance() : NULL;
    if (!window) {
        nh_log("no toast, text was: %s", qPrintable(primary));
        return;
    }
    MainWindowController_toast(window, primary, secondary, TOAST_DURATION_MILLISECONDS);
}

} // namespace NickelBridge

DisabledNotice::DisabledNotice() : QObject(NULL) {}

void DisabledNotice::show()
{
    NickelBridge::showToast(HelperStrings::text(HelperStrings::DISABLED), QStringLiteral(""));
    deleteLater();
}

// Once, a few seconds after the first status bar exists (Nickel's window is up).
static void scheduleDisabledNotice()
{
    if (!isDisabledNoticePending) {
        return;
    }
    isDisabledNoticePending = false;
    if (!canToast()) {
        nh_log("toast symbols missing, the disabled notice is only logged");
        return;
    }
    QTimer::singleShot(DISABLED_NOTICE_DELAY_MILLISECONDS, new DisabledNotice(), SLOT(show()));
}

// NickelHook PLT hook target for StatusBarView::StatusBarView(QWidget*). Always calls the original
// first; adds the button only while the bridge is active.
extern "C" __attribute__((visibility("default"))) void statusbarshot_StatusBarView_construct(void *statusBar,
                                                                                            QWidget *parent)
{
    StatusBarView_construct_original(statusBar, parent);
    HelperLog::debug("StatusBarView constructed %p", statusBar);
    if (isBridgeActive && statusBarBuiltHandler) {
        // StatusBarView derives from QWidget as its first base.
        statusBarBuiltHandler(reinterpret_cast<QWidget *>(statusBar));
    } else if (!isBridgeActive) {
        scheduleDisabledNotice();
    }
}
