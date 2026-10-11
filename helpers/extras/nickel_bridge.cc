#include "nickel_bridge.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include <QApplication>
#include <QDesktopWidget>
#include <QDialog>
#include <QGesture>
#include <QTimer>
#include <QWidget>

#include "debug_log.h"
#include "device_gate.h"
#include "helper_flags.h"
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
static void (*MainWindowController_push)(void *window, AbstractController *controller, bool animate);
static void (*MainWindowController_pop)(void *window, AbstractController *controller);
static void (*IconLeftButton_construct)(void *button, QWidget *parent);
static void (*IconLeftButton_setPixmap)(void *button, QPixmap const &pixmap);
static void (*IconLeftButton_setText)(void *button, QString const &text);
static QWidget *(*N3DialogFactory_getDialog)(QWidget *content, bool hasKeyboard);
static void (*N3Dialog_setTitle)(QWidget *dialog, QString const &title);
static void (*N3Dialog_setLargeTitleLayoutVisible)(QWidget *dialog, bool isVisible);
// Returns the ConfirmationDialog it made (re/notes/G2-findings.md "Popups and dialogs").
static QDialog *(*ConfirmationDialogFactory_getConfirmationDialog)(QWidget *parent);
static void (*ConfirmationDialog_setTitle)(QDialog *dialog, QString const &title);
static void (*ConfirmationDialog_setText)(QDialog *dialog, QString const &text);
static void (*ConfirmationDialog_setAcceptButtonText)(QDialog *dialog, QString const &text);
static void (*ConfirmationDialog_setRejectButtonText)(QDialog *dialog, QString const &text);
static void (*ConfirmationDialog_showCloseButton)(QDialog *dialog, bool isShown);
static void (*ConfirmationDialog_setRejectOnOutsideTap)(QDialog *dialog, bool isRejecting);
static void *(*N3PowerWorkflowManager_sharedInstance)();
static void (*N3PowerWorkflowManager_requestSleep)(void *manager);
static void (*N3PowerWorkflowManager_reboot)(void *manager);
static void *(*PlugWorkflowManager_sharedInstance)();
static void (*PlugWorkflowManager_sync)(void *manager);
// Data symbol: the Qt::GestureType that Nickel's TapGestureRecognizer registered
// (0 until Nickel registers it at startup, after our init).
static int *TapGestureRecognizer_gestureType;
// QPoint TapGesture::position() const: the touch point in screen coordinates.
static QPoint (*TapGesture_position)(const QGesture *gesture);
// Originals returned by the PLT hooks.
static void (*StatusBarView_construct_original)(void *statusBar, QWidget *parent);
static void (*Ui_MoreView_setupUi_original)(void *ui, QWidget *moreView);
}

// Signals connected by name (old-style SIGNAL(), no dlsym); check-symbols.sh verifies them too.
// signal-symbol: _ZN10TouchLabel6tappedEb
// signal-symbol: _ZN14IconButtonBase6tappedEv
// signal-symbol: _ZN8N3Dialog11closeTappedEv

static const char *const VERSION_FILE_PATH = "/mnt/onboard/.kobo/version";

static const int TOAST_DURATION_MILLISECONDS = 2000;
static const int DISABLED_NOTICE_DELAY_MILLISECONDS = 10000;
static const bool PUSH_WITHOUT_ANIMATION = false;
// StatusBarIcon is 0xC0 bytes on all five builds; allocate more, zero-filled, so a slightly larger
// class still fits (the risk table of helpers/extras/docs/statusbar-screenshot.md).
static const size_t STATUS_BAR_ICON_ALLOCATION = 0x100;
// IconLeftButton is 0x58 bytes on 23828; allocate generously and zeroed (as NickelMenu does for
// MainNavButton) so a slightly larger class still fits.
static const size_t ICON_LEFT_BUTTON_ALLOCATION = 256;
// ScreenshotMaker is 4 bytes ({int counter}); the constructor stores 1.
static const size_t SCREENSHOT_MAKER_ALLOCATION = 16;
static const bool WITHOUT_PAGE_MENU = false; // N3Dialog's bool: GoToPageMenuController on its "pages" label
// Kobo's patched Qt reads custom widget attributes as e-ink hints; 0x80 asks for a flashing full
// update (re/notes/G2-findings.md Q4).
static const Qt::WidgetAttribute FULL_REFRESH_ATTRIBUTE = static_cast<Qt::WidgetAttribute>(0x80);

static bool isButtonOn = false;
static bool isTweaksOn = false;
// Set when a feature disables itself (unknown model or firmware, missing symbol), not by the kill switch.
static bool isDisabledNoticePending = false;
// The live screen size can only be read once Nickel's QApplication exists.
static bool isScreenSizeChecked = false;
static char firmwareVersion[32] = "unknown";
static char deviceId[64] = "unknown";
static void *screenshotMaker = NULL;
static NickelBridge::WidgetHandler statusBarBuiltHandler = NULL;
static NickelBridge::WidgetHandler moreViewSetupHandler = NULL;

// ---- NickelHook tables ----

namespace NickelBridge {

struct nh_hook hookTable[] = {
    {.sym = "_ZN13StatusBarViewC1EP7QWidget",
     .sym_new = "extras_StatusBarView_construct",
     .lib = "libnickel.so.1.0.0",
     .out = nh_symoutptr(StatusBarView_construct_original),
     .desc = "StatusBarView::StatusBarView(QWidget*) (screenshot button)",
     .optional = true},
    {.sym = "_ZN11Ui_MoreView7setupUiEP7QWidget",
     .sym_new = "extras_Ui_MoreView_setupUi",
     .lib = "libnickel.so.1.0.0",
     .out = nh_symoutptr(Ui_MoreView_setupUi_original),
     .desc = "Ui_MoreView::setupUi(QWidget*) (Tweaks row)",
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
    NICKEL_SYMBOL("_ZN20MainWindowController14sharedInstanceEv", MainWindowController_sharedInstance, "window"),
    NICKEL_SYMBOL("_ZN20MainWindowController5toastERK7QStringS2_i", MainWindowController_toast, "toast"),
    NICKEL_SYMBOL("_ZN20MainWindowController4pushEP18AbstractControllerb", MainWindowController_push, "panel"),
    NICKEL_SYMBOL("_ZN20MainWindowController3popEP18AbstractController", MainWindowController_pop, "panel"),
    NICKEL_SYMBOL("_ZN14IconLeftButtonC1EP7QWidget", IconLeftButton_construct, "More row"),
    NICKEL_SYMBOL("_ZN14IconLeftButton9setPixmapERK7QPixmap", IconLeftButton_setPixmap, "More row"),
    NICKEL_SYMBOL("_ZN14IconLeftButton7setTextERK7QString", IconLeftButton_setText, "More row"),
    NICKEL_SYMBOL("_ZN15N3DialogFactory9getDialogEP7QWidgetb", N3DialogFactory_getDialog, "panel"),
    NICKEL_SYMBOL("_ZN8N3Dialog8setTitleERK7QString", N3Dialog_setTitle, "panel"),
    NICKEL_SYMBOL("_ZN8N3Dialog26setLargeTitleLayoutVisibleEb", N3Dialog_setLargeTitleLayoutVisible,
                  "panel title"),
    NICKEL_SYMBOL("_ZN25ConfirmationDialogFactory21getConfirmationDialogEP7QWidget",
                  ConfirmationDialogFactory_getConfirmationDialog, "confirm dialog"),
    NICKEL_SYMBOL("_ZN18ConfirmationDialog8setTitleERK7QString", ConfirmationDialog_setTitle, "confirm dialog"),
    NICKEL_SYMBOL("_ZN18ConfirmationDialog7setTextERK7QString", ConfirmationDialog_setText, "confirm dialog"),
    NICKEL_SYMBOL("_ZN18ConfirmationDialog19setAcceptButtonTextERK7QString", ConfirmationDialog_setAcceptButtonText,
                  "confirm dialog"),
    NICKEL_SYMBOL("_ZN18ConfirmationDialog19setRejectButtonTextERK7QString", ConfirmationDialog_setRejectButtonText,
                  "confirm dialog"),
    NICKEL_SYMBOL("_ZN18ConfirmationDialog15showCloseButtonEb", ConfirmationDialog_showCloseButton, "confirm dialog"),
    NICKEL_SYMBOL("_ZN18ConfirmationDialog21setRejectOnOutsideTapEb", ConfirmationDialog_setRejectOnOutsideTap,
                  "dialog outside tap"),
    NICKEL_SYMBOL("_ZN22N3PowerWorkflowManager14sharedInstanceEv", N3PowerWorkflowManager_sharedInstance,
                  "sleep/reboot"),
    NICKEL_SYMBOL("_ZN22N3PowerWorkflowManager12requestSleepEv", N3PowerWorkflowManager_requestSleep, "sleep"),
    NICKEL_SYMBOL("_ZN22N3PowerWorkflowManager6rebootEv", N3PowerWorkflowManager_reboot, "reboot"),
    NICKEL_SYMBOL("_ZN19PlugWorkflowManager14sharedInstanceEv", PlugWorkflowManager_sharedInstance, "rescan"),
    NICKEL_SYMBOL("_ZN19PlugWorkflowManager4syncEv", PlugWorkflowManager_sync, "rescan"),
    NICKEL_SYMBOL("_ZN20TapGestureRecognizer12_gestureTypeE", TapGestureRecognizer_gestureType, "taps"),
    NICKEL_SYMBOL("_ZNK10TapGesture8positionEv", TapGesture_position, "tap point"),
    {0},
};

#undef NICKEL_SYMBOL

} // namespace NickelBridge

// ---- Logging ----

// Always to the syslog, and to debug.log too when the debug flag is on.
static void logCheck(const char *line)
{
    if (DebugLog::isEnabled()) {
        DebugLog::write("bridge: %s", line);
    } else {
        nh_log("bridge: %s", line);
    }
}

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
static const char *missingButtonSymbol()
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

// Without these the row cannot exist or the panel cannot open and take taps. The confirm dialog,
// power, rescan, the large-title hide and the tap point degrade one by one. NULL: all present.
static const char *missingTweaksSymbol()
{
    if (!Ui_MoreView_setupUi_original) {
        return "Ui_MoreView::setupUi(QWidget*)";
    }
    if (!IconLeftButton_construct || !IconLeftButton_setPixmap || !IconLeftButton_setText) {
        return "IconLeftButton";
    }
    if (!MainWindowController_sharedInstance || !MainWindowController_push || !MainWindowController_pop) {
        return "MainWindowController::push/pop";
    }
    if (!N3DialogFactory_getDialog || !N3Dialog_setTitle) {
        return "N3Dialog";
    }
    if (!TapGestureRecognizer_gestureType) {
        return "TapGestureRecognizer::_gestureType";
    }
    return NULL;
}

static bool canToast()
{
    return MainWindowController_sharedInstance && MainWindowController_toast;
}

static void disableFeature(bool *isFeatureOn, const char *feature, const char *reason)
{
    *isFeatureOn = false;
    isDisabledNoticePending = true;
    char line[200];
    snprintf(line, sizeof(line), "%s, %s disabled", reason, feature);
    logCheck(line);
}

static bool passesModelCheck()
{
    const DeviceGate::SupportedModel *model = DeviceGate::findSupportedModel(deviceId);
    if (!model) {
        char reason[160];
        snprintf(reason, sizeof(reason), "device %s is not a supported model", deviceId);
        isDisabledNoticePending = true;
        logCheck(reason);
        return false;
    }
    char line[160];
    snprintf(line, sizeof(line), "device %s is model %s (%s), supported", deviceId, model->idSuffix, model->codename);
    logCheck(line);
    return true;
}

static bool passesFirmwareCheck()
{
    if (!DeviceGate::isSupportedFirmware(firmwareVersion)) {
        char reason[160];
        snprintf(reason, sizeof(reason), "firmware %s not in the tested list", firmwareVersion);
        isDisabledNoticePending = true;
        logCheck(reason);
        return false;
    }
    return true;
}

// Belt and braces for the model check, for the panel's fixed layout: the size Nickel's Qt actually
// reports. Once, on the first hook (init() runs before Nickel creates QApplication).
static void checkScreenSizeOnce()
{
    if (isScreenSizeChecked || !isTweaksOn) {
        return;
    }
    isScreenSizeChecked = true;
    QDesktopWidget *desktop = qApp ? QApplication::desktop() : NULL;
    if (!desktop) {
        disableFeature(&isTweaksOn, "tweaks", "no QApplication desktop, screen size unknown");
        return;
    }
    QRect screen = desktop->screenGeometry();
    if (!DeviceGate::isSupportedScreenSize(screen.width(), screen.height())) {
        char reason[120];
        snprintf(reason, sizeof(reason), "screen %dx%d is not the supported size", screen.width(), screen.height());
        disableFeature(&isTweaksOn, "tweaks", reason);
        return;
    }
    char line[64];
    snprintf(line, sizeof(line), "screen %dx%d, supported", screen.width(), screen.height());
    logCheck(line);
}

namespace NickelBridge {

bool activate()
{
    readVersionFile();
    if (!passesModelCheck() || !passesFirmwareCheck()) {
        return false;
    }
    const char *missingForButton = missingButtonSymbol();
    if (missingForButton) {
        char reason[160];
        snprintf(reason, sizeof(reason), "libnickel symbol %s missing", missingForButton);
        disableFeature(&isButtonOn, "screenshot button", reason);
    } else {
        isButtonOn = true;
    }
    const char *missingForTweaks = missingTweaksSymbol();
    if (missingForTweaks) {
        char reason[160];
        snprintf(reason, sizeof(reason), "libnickel symbol %s missing", missingForTweaks);
        disableFeature(&isTweaksOn, "tweaks", reason);
    } else {
        isTweaksOn = true;
    }
    return isButtonOn || isTweaksOn;
}

bool isButtonActive()
{
    return isButtonOn;
}

bool isTweaksActive()
{
    return isTweaksOn;
}

void setStatusBarBuiltHandler(WidgetHandler handler)
{
    statusBarBuiltHandler = handler;
}

void setMoreViewSetupHandler(WidgetHandler handler)
{
    moreViewSetupHandler = handler;
}

// ---- Status-bar button ----

QWidget *createStatusBarIcon(QWidget *parent)
{
    if (!isButtonOn) {
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
    if (!isButtonOn || !icon) {
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
    void *maker = isButtonOn ? makerInstance() : NULL;
    return maker ? ScreenshotMaker_fileName(maker) : QString();
}

bool saveScreenshot()
{
    void *maker = isButtonOn ? makerInstance() : NULL;
    if (!maker) {
        return false;
    }
    ScreenshotMaker_saveCurrentScreen(maker);
    return true;
}

// ---- More tab row ----

static bool hasIconLeftButton()
{
    return isTweaksOn && IconLeftButton_construct && IconLeftButton_setPixmap && IconLeftButton_setText;
}

static QWidget *createIconLeftButton(QWidget *parent)
{
    if (!hasIconLeftButton()) {
        nh_log("IconLeftButton unavailable");
        return NULL;
    }
    void *memory = calloc(1, ICON_LEFT_BUTTON_ALLOCATION);
    if (!memory) {
        nh_log("out of memory for IconLeftButton");
        return NULL;
    }
    DebugLog::write("bridge: constructing IconLeftButton at %p, parent %p", memory, (void *)parent);
    IconLeftButton_construct(memory, parent);
    QWidget *button = reinterpret_cast<QWidget *>(memory);
    DebugLog::write("bridge: constructed %s", button->metaObject()->className());
    return button;
}

static void setIconLeftButtonPixmap(QWidget *button, const QPixmap &pixmap)
{
    if (hasIconLeftButton() && button) {
        IconLeftButton_setPixmap(button, pixmap);
    }
}

static void setIconLeftButtonText(QWidget *button, const QString &text)
{
    if (hasIconLeftButton() && button) {
        IconLeftButton_setText(button, text);
    }
}

MoreRow::NickelApi moreRowApi()
{
    MoreRow::NickelApi api = {hasIconLeftButton, createIconLeftButton, setIconLeftButtonPixmap,
                              setIconLeftButtonText};
    return api;
}

// ---- Tap gestures ----

int tapGestureType()
{
    return isTweaksOn && TapGestureRecognizer_gestureType ? *TapGestureRecognizer_gestureType : 0;
}

bool tapGesturePosition(QGesture *gesture, QPoint *screenPosition)
{
    // inherits(), not a C++ cast: the class is Nickel's (re/notes/G2-findings.md, RTTI).
    if (!isTweaksOn || !TapGesture_position || !gesture || !gesture->inherits("TapGesture")) {
        return false;
    }
    *screenPosition = TapGesture_position(gesture);
    return true;
}

// ---- Window stack and the Settings-style modal ----

static void *mainWindow()
{
    void *window = MainWindowController_sharedInstance ? MainWindowController_sharedInstance() : NULL;
    if (!window) {
        nh_log("MainWindowController::sharedInstance unavailable");
    }
    return window;
}

bool hasControllerPush()
{
    return isTweaksOn && MainWindowController_sharedInstance && MainWindowController_push && MainWindowController_pop;
}

void pushController(AbstractController *controller)
{
    void *window = hasControllerPush() ? mainWindow() : NULL;
    if (window && controller) {
        MainWindowController_push(window, controller, PUSH_WITHOUT_ANIMATION);
    }
}

void popController(AbstractController *controller)
{
    void *window = hasControllerPush() ? mainWindow() : NULL;
    if (window && controller) {
        MainWindowController_pop(window, controller);
    }
}

bool hasSettingsDialog()
{
    return isTweaksOn && N3DialogFactory_getDialog && N3Dialog_setTitle;
}

QWidget *createSettingsDialog(QWidget *content, const QString &title)
{
    if (!hasSettingsDialog() || !content) {
        nh_log("N3Dialog unavailable");
        return NULL;
    }
    QWidget *dialog = N3DialogFactory_getDialog(content, WITHOUT_PAGE_MENU);
    if (!dialog) {
        nh_log("N3DialogFactory::getDialog returned null");
        return NULL;
    }
    N3Dialog_setTitle(dialog, title);
    // The constructor leaves the large title empty; hide its row too when we can.
    if (N3Dialog_setLargeTitleLayoutVisible) {
        N3Dialog_setLargeTitleLayoutVisible(dialog, false);
    }
    DebugLog::write("bridge: settings dialog %p (%s)", static_cast<void *>(dialog), dialog->metaObject()->className());
    return dialog;
}

// ---- Feedback ----

void showToast(const QString &primary, const QString &secondary)
{
    void *window = canToast() ? mainWindow() : NULL;
    if (!window) {
        nh_log("no toast, text was: %s", qPrintable(primary));
        return;
    }
    MainWindowController_toast(window, primary, secondary, TOAST_DURATION_MILLISECONDS);
}

bool hasConfirmDialog()
{
    return isTweaksOn && ConfirmationDialogFactory_getConfirmationDialog && ConfirmationDialog_setTitle &&
           ConfirmationDialog_setText && ConfirmationDialog_setAcceptButtonText &&
           ConfirmationDialog_setRejectButtonText && ConfirmationDialog_showCloseButton;
}

QDialog *createConfirmDialog(const QString &title, const QString &text, const QString &acceptLabel,
                             const QString &rejectLabel)
{
    if (!hasConfirmDialog()) {
        return NULL;
    }
    QDialog *dialog = ConfirmationDialogFactory_getConfirmationDialog(NULL);
    if (!dialog) {
        nh_log("getConfirmationDialog returned null");
        return NULL;
    }
    ConfirmationDialog_setTitle(dialog, title);
    ConfirmationDialog_setText(dialog, text.toHtmlEscaped());
    ConfirmationDialog_setAcceptButtonText(dialog, acceptLabel);
    ConfirmationDialog_setRejectButtonText(dialog, rejectLabel);
    ConfirmationDialog_showCloseButton(dialog, false);
    if (ConfirmationDialog_setRejectOnOutsideTap) {
        ConfirmationDialog_setRejectOnOutsideTap(dialog, true);
    }
    return dialog;
}

// ---- Power and library ----

bool hasPowerAction(PowerAction action)
{
    bool hasManager = isTweaksOn && N3PowerWorkflowManager_sharedInstance;
    return hasManager && (action == POWER_SLEEP ? N3PowerWorkflowManager_requestSleep != NULL
                                                : N3PowerWorkflowManager_reboot != NULL);
}

bool runPowerAction(PowerAction action)
{
    void *manager = hasPowerAction(action) ? N3PowerWorkflowManager_sharedInstance() : NULL;
    if (!manager) {
        nh_log("power action %s unavailable", action == POWER_SLEEP ? "sleep" : "reboot");
        return false;
    }
    nh_log("power action %s", action == POWER_SLEEP ? "sleep" : "reboot");
    if (action == POWER_SLEEP) {
        N3PowerWorkflowManager_requestSleep(manager);
    } else {
        N3PowerWorkflowManager_reboot(manager);
    }
    return true;
}

bool rescanLibrary()
{
    bool hasRescan = isTweaksOn && PlugWorkflowManager_sharedInstance && PlugWorkflowManager_sync;
    void *manager = hasRescan ? PlugWorkflowManager_sharedInstance() : NULL;
    if (!manager) {
        nh_log("library rescan unavailable");
        return false;
    }
    PlugWorkflowManager_sync(manager);
    return true;
}

// ---- E-ink ----

void requestFullRefresh(QWidget *widget)
{
    if (!widget) {
        return;
    }
    widget->setAttribute(FULL_REFRESH_ATTRIBUTE, true);
    widget->repaint();
    widget->setAttribute(FULL_REFRESH_ATTRIBUTE, false);
}

} // namespace NickelBridge

DisabledNotice::DisabledNotice() : QObject(NULL) {}

void DisabledNotice::show()
{
    HelperStrings::reloadLocale();
    NickelBridge::showToast(HelperStrings::text(HelperStrings::DISABLED), QStringLiteral(""));
    deleteLater();
}

// Once, a few seconds after the first hook ran (Nickel's window is up).
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
// first; adds the button only while that feature is on.
extern "C" __attribute__((visibility("default"))) void extras_StatusBarView_construct(void *statusBar,
                                                                                      QWidget *parent)
{
    StatusBarView_construct_original(statusBar, parent);
    DebugLog::write("bridge: StatusBarView constructed %p", statusBar);
    if (isButtonOn && statusBarBuiltHandler) {
        // StatusBarView derives from QWidget as its first base.
        statusBarBuiltHandler(reinterpret_cast<QWidget *>(statusBar));
    }
    scheduleDisabledNotice();
}

// NickelHook PLT hook target for Ui_MoreView::setupUi(QWidget*), called by the MoreView constructor
// (which NickelDrawer hooks; one hook per symbol). Always calls the original first; the layout exists
// when it returns, so the row can be added now.
extern "C" __attribute__((visibility("default"))) void extras_Ui_MoreView_setupUi(void *ui, QWidget *moreView)
{
    Ui_MoreView_setupUi_original(ui, moreView);
    DebugLog::write("bridge: Ui_MoreView::setupUi done for %p", static_cast<void *>(moreView));
    checkScreenSizeOnce();
    if (isTweaksOn && moreViewSetupHandler) {
        moreViewSetupHandler(moreView);
    }
    scheduleDisabledNotice();
}
