#include "nickel_bridge.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include <QDialog>
#include <QGesture>
#include <QLineEdit>
#include <QMenu>
#include <QWidgetAction>
#include <QLocale>
#include <QPixmap>
#include <QTimer>
#include <QVariant>
#include <QWidget>

#include "debug_log.h"

// ---- libnickel entry points (filled by NickelHook before init; NULL if absent) ----

extern "C" {
static void *(*MainWindowController_sharedInstance)();
static void (*MainWindowController_toast)(void *window, QString const &primary, QString const &secondary,
                                          int milliseconds);
static void (*MainWindowController_push)(void *window, AbstractController *controller, bool animate);
static void (*MainWindowController_pop)(void *window, AbstractController *controller);
static void (*MainWindowController_pushView)(void *window, QWidget *view);
static void (*MainWindowController_popView)(void *window, QWidget *view);
static void (*IconLeftButton_construct)(void *button, QWidget *parent);
static void (*IconLeftButton_setPixmap)(void *button, QPixmap const &pixmap);
static void (*IconLeftButton_setText)(void *button, QString const &text);
// Returns the ConfirmationDialog it opened (re/notes/G2-findings.md "Popups and dialogs").
static QDialog *(*ConfirmationDialogFactory_showOKDialog)(QString const &title, QString const &body);
static QDialog *(*ConfirmationDialogFactory_getConfirmationDialog)(QWidget *parent);
static void (*ConfirmationDialog_setTitle)(QDialog *dialog, QString const &title);
static void (*ConfirmationDialog_setText)(QDialog *dialog, QString const &text);
static void (*ConfirmationDialog_setAcceptButtonText)(QDialog *dialog, QString const &text);
static void (*ConfirmationDialog_setRejectButtonText)(QDialog *dialog, QString const &text);
static void (*ConfirmationDialog_showCloseButton)(QDialog *dialog, bool isShown);
static void (*ConfirmationDialog_setRejectOnOutsideTap)(QDialog *dialog, bool isRejecting);
static void *(*WirelessWorkflowManager_sharedInstance)();
static bool (*WirelessWorkflowManager_isInternetAccessible)(void *manager);
static void (*WirelessWorkflowManager_connectWireless)(void *manager, bool, bool);
static void *(*PlugWorkflowManager_sharedInstance)();
static void (*PlugWorkflowManager_sync)(void *manager);
static void (*PlugWorkflowManager_onCancelAndConnect)(void *manager);
static void (*PlugWorkflowManager_unplugged)(void *manager);
static QWidget *(*N3DialogFactory_getDialog)(QWidget *content, bool hasKeyboard);
static void (*N3Dialog_setTitle)(QWidget *dialog, QString const &title);
static QWidget *(*N3Dialog_keyboardFrame)(QWidget *dialog);
static void (*N3Dialog_showKeyboard)(QWidget *dialog);
static void (*N3Dialog_setLargeTitleLayoutVisible)(QWidget *dialog, bool isVisible);
static void *(*N3PowerWorkflowManager_sharedInstance)();
static void (*N3PowerWorkflowManager_requestSleep)(void *manager);
static void (*N3PowerWorkflowManager_reboot)(void *manager);
// KeyboardScript is an enum passed in a register; 0 is what NickelHardcover passes.
static QObject *(*KeyboardFrame_createKeyboard)(QWidget *frame, int keyboardScript, QLocale const &locale);
static void (*KeyboardReceiver_construct)(void *receiver, QLineEdit *lineEdit, bool unknown);
static void (*SearchKeyboardController_setReceiver)(QObject *controller, void *receiver, bool unknown);
static void (*SearchKeyboardController_setGoText)(QObject *controller, QString const &text);
static void (*TextHeader_construct)(void *header, QWidget *parent);
static void (*TextHeader_setText)(void *header, QString const &text);
static void (*TextHeaderBase_setDeselectedPixmap)(void *header, QPixmap const &pixmap);
static void (*TextHeaderBase_setSelectedPixmap)(void *header, QPixmap const &pixmap);
static void (*TextHeaderBase_setInvertImageOnly)(void *header, bool isImageOnly);
// DecorationPosition is an enum passed in a register; 3 is what NickelMenu passes.
static void (*NickelTouchMenu_construct)(void *menu, QWidget *parent, int decorationPosition);
static void (*NickelTouchMenu_showDecoration)(void *menu, bool isShown);
static void (*MenuTextItem_construct)(void *item, QWidget *parent, bool isCheckable, bool isItalic);
static void (*MenuTextItem_setText)(void *item, QString const &text);
static void (*MenuTextItem_registerForTapGestures)(void *item);
// Data symbol: the Qt::GestureType that Nickel's TapGestureRecognizer registered
// (0 until Nickel registers it at startup, after our init).
static int *TapGestureRecognizer_gestureType;
// QPoint TapGesture::position() const: the touch point in screen coordinates
// (TouchPoint::scenePos, which is the screen position for widgets), set on
// touch begin; Nickel never sets the QGesture hot spot.
static QPoint (*TapGesture_position)(const QGesture *gesture);
// Original MoreView::MoreView(QWidget*), returned by the PLT hook.
static void (*MoreView_construct_original)(void *moreView, QWidget *parent);
}

// Signals connected by name (old-style SIGNAL(), no dlsym). check-symbols.sh
// verifies these too.
// signal-symbol: _ZN14IconButtonBase6tappedEv
// signal-symbol: _ZN18KeyboardController15commitRequestedEv
// signal-symbol: _ZN8N3Dialog11closeTappedEv
// signal-symbol: _ZN14TextHeaderBase6tappedEv
// signal-symbol: _ZN14TextHeaderBase9mouseDownEv
// signal-symbol: _ZN12MenuTextItem6tappedEb
// signal-symbol: _ZN19PlugWorkflowManager24usbPlugPermissionRequestEP17PermissionRequest
// signal-symbol: _ZN19PlugWorkflowManager36connectConfirmationPermissionRequestEP17PermissionRequestPb

static const char *const VERSION_FILE_PATH = "/mnt/onboard/.kobo/version";
static const int FIRMWARE_FIELD_INDEX = 2; // "serial,3.0.0,4.38.23828,..."
static const char *const SUPPORTED_FIRMWARE[] = {
    "4.38.23552", "4.38.23648", "4.38.23684", "4.38.23697", "4.38.23828",
};

static const int TOAST_DURATION_MILLISECONDS = 3000;
static const bool PUSH_WITHOUT_ANIMATION = false;
// IconLeftButton is 0x58 bytes on 23828; allocate generously and zeroed (as
// NickelMenu does for MainNavButton) so a slightly larger class still fits.
static const size_t ICON_LEFT_BUTTON_ALLOCATION = 256;
// KeyboardReceiver's size is unknown; NickelHardcover uses 128 bytes, twice that leaves room.
static const size_t KEYBOARD_RECEIVER_ALLOCATION = 256;
// NickelMenu's sizes for its NickelTouchMenu and MenuTextItem blocks (about 3x the 15505 classes).
static const size_t TOUCH_MENU_ALLOCATION = 512;
static const size_t MENU_TEXT_ITEM_ALLOCATION = 256;
static const int TOUCH_MENU_DECORATION_POSITION = 3;
static const bool MENU_ITEM_CHECKABLE = false;
static const bool MENU_ITEM_ITALIC = true; // NickelMenu's arguments; Nickel's QSS decides the look
static const int DEFAULT_KEYBOARD_SCRIPT = 0;
static const bool WITH_KEYBOARD = true;
static const bool WITHOUT_PAGE_MENU = false; // N3Dialog's bool: GoToPageMenuController on its "pages" label
// NickelMenu's "autoconnect" arguments: shows Nickel's connect UI when needed.
static const bool CONNECT_WIRELESS_FIRST_ARGUMENT = true;
static const bool CONNECT_WIRELESS_SECOND_ARGUMENT = false;
static const int FULL_RESCAN_UNPLUG_DELAY_MILLISECONDS = 1000;
// Kobo's patched Qt reads custom widget attributes as e-ink hints; 0x80 asks
// for a flashing full update (re/notes/G2-findings.md Q4).
static const Qt::WidgetAttribute FULL_REFRESH_ATTRIBUTE = static_cast<Qt::WidgetAttribute>(0x80);

static bool isBridgeActive = false;
static char firmwareVersion[32] = "unknown";
static NickelBridge::MoreViewCreatedHandler moreViewCreatedHandler = NULL;

// ---- NickelHook tables ----

namespace NickelBridge {

struct nh_hook hookTable[] = {
    {.sym = "_ZN8MoreViewC1EP7QWidget",
     .sym_new = "nickeldrawer_MoreView_construct",
     .lib = "libnickel.so.1.0.0",
     .out = nh_symoutptr(MoreView_construct_original),
     .desc = "MoreView::MoreView(QWidget*) (Apps row)",
     .optional = true},
    {0},
};

#define NICKEL_SYMBOL(mangled, variable, description)                                                    \
    {.name = mangled, .out = nh_symoutptr(variable), .desc = description, .optional = true}

struct nh_dlsym symbolTable[] = {
    NICKEL_SYMBOL("_ZN20MainWindowController14sharedInstanceEv", MainWindowController_sharedInstance, "window"),
    NICKEL_SYMBOL("_ZN20MainWindowController5toastERK7QStringS2_i", MainWindowController_toast, "toast"),
    NICKEL_SYMBOL("_ZN20MainWindowController4pushEP18AbstractControllerb", MainWindowController_push, "route A"),
    NICKEL_SYMBOL("_ZN20MainWindowController3popEP18AbstractController", MainWindowController_pop, "route A"),
    NICKEL_SYMBOL("_ZN20MainWindowController8pushViewEP7QWidget", MainWindowController_pushView, "route B"),
    NICKEL_SYMBOL("_ZN20MainWindowController7popViewEP7QWidget", MainWindowController_popView, "route B"),
    NICKEL_SYMBOL("_ZN14IconLeftButtonC1EP7QWidget", IconLeftButton_construct, "Apps row"),
    NICKEL_SYMBOL("_ZN14IconLeftButton9setPixmapERK7QPixmap", IconLeftButton_setPixmap, "Apps row"),
    NICKEL_SYMBOL("_ZN14IconLeftButton7setTextERK7QString", IconLeftButton_setText, "Apps row"),
    NICKEL_SYMBOL("_ZN25ConfirmationDialogFactory12showOKDialogERK7QStringS2_",
                  ConfirmationDialogFactory_showOKDialog, "result dialog"),
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
    NICKEL_SYMBOL("_ZN15NickelTouchMenuC1EP7QWidget18DecorationPosition", NickelTouchMenu_construct, "row menu"),
    NICKEL_SYMBOL("_ZN15NickelTouchMenu14showDecorationEb", NickelTouchMenu_showDecoration, "row menu"),
    NICKEL_SYMBOL("_ZN12MenuTextItemC1EP7QWidgetbb", MenuTextItem_construct, "row menu"),
    NICKEL_SYMBOL("_ZN12MenuTextItem7setTextERK7QString", MenuTextItem_setText, "row menu"),
    NICKEL_SYMBOL("_ZN12MenuTextItem22registerForTapGesturesEv", MenuTextItem_registerForTapGestures, "row menu"),
    NICKEL_SYMBOL("_ZN23WirelessWorkflowManager14sharedInstanceEv", WirelessWorkflowManager_sharedInstance, "Wi-Fi"),
    NICKEL_SYMBOL("_ZN23WirelessWorkflowManager20isInternetAccessibleEv", WirelessWorkflowManager_isInternetAccessible,
                  "Wi-Fi"),
    NICKEL_SYMBOL("_ZN23WirelessWorkflowManager15connectWirelessEbb", WirelessWorkflowManager_connectWireless, "Wi-Fi"),
    NICKEL_SYMBOL("_ZN19PlugWorkflowManager14sharedInstanceEv", PlugWorkflowManager_sharedInstance, "rescan"),
    NICKEL_SYMBOL("_ZN19PlugWorkflowManager4syncEv", PlugWorkflowManager_sync, "quick rescan"),
    NICKEL_SYMBOL("_ZN19PlugWorkflowManager18onCancelAndConnectEv", PlugWorkflowManager_onCancelAndConnect,
                  "full rescan"),
    NICKEL_SYMBOL("_ZN19PlugWorkflowManager9unpluggedEv", PlugWorkflowManager_unplugged, "full rescan"),
    NICKEL_SYMBOL("_ZN15N3DialogFactory9getDialogEP7QWidgetb", N3DialogFactory_getDialog, "field keyboard"),
    NICKEL_SYMBOL("_ZN8N3Dialog8setTitleERK7QString", N3Dialog_setTitle, "field keyboard"),
    NICKEL_SYMBOL("_ZN8N3Dialog13keyboardFrameEv", N3Dialog_keyboardFrame, "field keyboard"),
    NICKEL_SYMBOL("_ZN8N3Dialog12showKeyboardEv", N3Dialog_showKeyboard, "field keyboard"),
    NICKEL_SYMBOL("_ZN8N3Dialog26setLargeTitleLayoutVisibleEb", N3Dialog_setLargeTitleLayoutVisible,
                  "tweaks modal title"),
    NICKEL_SYMBOL("_ZN22N3PowerWorkflowManager14sharedInstanceEv", N3PowerWorkflowManager_sharedInstance,
                  "sleep/reboot"),
    NICKEL_SYMBOL("_ZN22N3PowerWorkflowManager12requestSleepEv", N3PowerWorkflowManager_requestSleep, "sleep"),
    NICKEL_SYMBOL("_ZN22N3PowerWorkflowManager6rebootEv", N3PowerWorkflowManager_reboot, "reboot"),
    NICKEL_SYMBOL("_ZN13KeyboardFrame14createKeyboardE14KeyboardScriptRK7QLocale", KeyboardFrame_createKeyboard,
                  "field keyboard"),
    NICKEL_SYMBOL("_ZN16KeyboardReceiverC1EP9QLineEditb", KeyboardReceiver_construct, "field keyboard"),
    NICKEL_SYMBOL("_ZN24SearchKeyboardController11setReceiverEP16KeyboardReceiverb",
                  SearchKeyboardController_setReceiver, "field keyboard"),
    NICKEL_SYMBOL("_ZN24SearchKeyboardController9setGoTextERK7QString", SearchKeyboardController_setGoText,
                  "field keyboard"),
    NICKEL_SYMBOL("_ZN20TapGestureRecognizer12_gestureTypeE", TapGestureRecognizer_gestureType, "taps"),
    NICKEL_SYMBOL("_ZNK10TapGesture8positionEv", TapGesture_position, "tap point"),
    NICKEL_SYMBOL("_ZN10TextHeaderC1EP7QWidget", TextHeader_construct, "page header"),
    NICKEL_SYMBOL("_ZN10TextHeader7setTextERK7QString", TextHeader_setText, "page header"),
    NICKEL_SYMBOL("_ZN14TextHeaderBase19setDeselectedPixmapERK7QPixmap", TextHeaderBase_setDeselectedPixmap,
                  "page header"),
    NICKEL_SYMBOL("_ZN14TextHeaderBase17setSelectedPixmapERK7QPixmap", TextHeaderBase_setSelectedPixmap,
                  "page header"),
    NICKEL_SYMBOL("_ZN14TextHeaderBase18setInvertImageOnlyEb", TextHeaderBase_setInvertImageOnly, "page header"),
    {0},
};

#undef NICKEL_SYMBOL

} // namespace NickelBridge

// ---- Firmware check ----

static void readFirmwareVersion()
{
    char line[128];
    FILE *file = fopen(VERSION_FILE_PATH, "r");
    if (!file) {
        nh_log("bridge: cannot open %s", VERSION_FILE_PATH);
        return;
    }
    char *got = fgets(line, sizeof(line), file);
    fclose(file); // never keep files open on /mnt/onboard
    if (!got) {
        nh_log("bridge: %s is empty", VERSION_FILE_PATH);
        return;
    }
    char *cursor = line;
    for (int field = 0; field < FIRMWARE_FIELD_INDEX; field++) {
        cursor = strchr(cursor, ',');
        if (!cursor) {
            return;
        }
        cursor++;
    }
    cursor[strcspn(cursor, ",\r\n")] = '\0';
    if (*cursor) {
        snprintf(firmwareVersion, sizeof(firmwareVersion), "%s", cursor);
    }
}

static bool isSupportedFirmware(const char *firmware)
{
    for (size_t i = 0; i < sizeof(SUPPORTED_FIRMWARE) / sizeof(SUPPORTED_FIRMWARE[0]); i++) {
        if (!strcmp(firmware, SUPPORTED_FIRMWARE[i])) {
            return true;
        }
    }
    return false;
}

static void *mainWindow()
{
    void *window = MainWindowController_sharedInstance ? MainWindowController_sharedInstance() : NULL;
    if (!window) {
        nh_log("bridge: MainWindowController::sharedInstance unavailable");
    }
    return window;
}

namespace NickelBridge {

bool activate()
{
    readFirmwareVersion();
    isBridgeActive = isSupportedFirmware(firmwareVersion);
    if (!isBridgeActive) {
        nh_log("bridge: firmware %s not in the tested list, drawer disabled (NickelMenu stays)", firmwareVersion);
    }
    return isBridgeActive;
}

bool isActive()
{
    return isBridgeActive;
}

const char *firmware()
{
    return firmwareVersion;
}

void setMoreViewCreatedHandler(MoreViewCreatedHandler handler)
{
    moreViewCreatedHandler = handler;
}

int tapGestureType()
{
    return isBridgeActive && TapGestureRecognizer_gestureType ? *TapGestureRecognizer_gestureType : 0;
}

bool tapGesturePosition(QGesture *gesture, QPoint *screenPosition)
{
    // inherits(), not a C++ cast: the class is Nickel's (re/notes/G2-findings.md, RTTI).
    if (!isBridgeActive || !TapGesture_position || !gesture || !gesture->inherits("TapGesture")) {
        return false;
    }
    *screenPosition = TapGesture_position(gesture);
    return true;
}

QObject *plugWorkflowManager()
{
    bool hasManager = isBridgeActive && PlugWorkflowManager_sharedInstance;
    // Its QObject base is at offset 0 (its signals activate on this pointer).
    QObject *manager = hasManager ? static_cast<QObject *>(PlugWorkflowManager_sharedInstance()) : NULL;
    if (!manager) {
        DebugLog::write("bridge: PlugWorkflowManager unavailable, USB plug not watched");
    }
    return manager;
}

NickelTextHeaderApi textHeaderApi()
{
    NickelTextHeaderApi api = {NULL, NULL, NULL, NULL, NULL};
    if (isBridgeActive) {
        api.construct = TextHeader_construct;
        api.setText = TextHeader_setText;
        api.setDeselectedPixmap = TextHeaderBase_setDeselectedPixmap;
        api.setSelectedPixmap = TextHeaderBase_setSelectedPixmap;
        api.setInvertImageOnly = TextHeaderBase_setInvertImageOnly;
    }
    return api;
}

// ---- More tab ----

bool hasIconLeftButton()
{
    return isBridgeActive && MoreView_construct_original && IconLeftButton_construct && IconLeftButton_setPixmap &&
           IconLeftButton_setText;
}

QWidget *createIconLeftButton(QWidget *parent)
{
    if (!hasIconLeftButton()) {
        nh_log("bridge: IconLeftButton unavailable");
        return NULL;
    }
    void *memory = calloc(1, ICON_LEFT_BUTTON_ALLOCATION);
    if (!memory) {
        nh_log("bridge: out of memory for IconLeftButton");
        return NULL;
    }
    DebugLog::write("bridge: constructing IconLeftButton at %p, parent %p", memory, (void *)parent);
    IconLeftButton_construct(memory, parent);
    QWidget *button = reinterpret_cast<QWidget *>(memory);
    DebugLog::write("bridge: constructed %s", button->metaObject()->className());
    return button;
}

void setIconLeftButtonPixmap(QWidget *button, const QPixmap &pixmap)
{
    if (hasIconLeftButton() && button) {
        IconLeftButton_setPixmap(button, pixmap);
    }
}

void setIconLeftButtonText(QWidget *button, const QString &text)
{
    if (hasIconLeftButton() && button) {
        IconLeftButton_setText(button, text);
    }
}

// ---- Window stack ----

bool hasControllerPush()
{
    return isBridgeActive && MainWindowController_sharedInstance && MainWindowController_push &&
           MainWindowController_pop;
}

bool hasViewPush()
{
    return isBridgeActive && MainWindowController_sharedInstance && MainWindowController_pushView &&
           MainWindowController_popView;
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

void pushView(QWidget *view)
{
    void *window = hasViewPush() ? mainWindow() : NULL;
    if (window && view) {
        MainWindowController_pushView(window, view);
    }
}

void popView(QWidget *view)
{
    void *window = hasViewPush() ? mainWindow() : NULL;
    if (window && view) {
        MainWindowController_popView(window, view);
    }
}

// ---- Feedback ----

bool hasToast()
{
    return isBridgeActive && MainWindowController_sharedInstance && MainWindowController_toast;
}

void showToast(const QString &text)
{
    void *window = hasToast() ? mainWindow() : NULL;
    if (!window) {
        nh_log("bridge: no toast, text was: %s", qPrintable(text));
        return;
    }
    DebugLog::write("bridge: toast on window %p: %s", window, qPrintable(text));
    // Empty (not null) secondary text: the form proven on device by ndhello.
    MainWindowController_toast(window, text, QStringLiteral(""), TOAST_DURATION_MILLISECONDS);
    DebugLog::write("bridge: toast queued");
}

bool hasMessageDialog()
{
    return isBridgeActive && ConfirmationDialogFactory_showOKDialog;
}

// Nickel's ConfirmationDialog::tapGesture rejects a tap outside its frame
// when the flag is set; without the symbol the dialog keeps Nickel's default.
static void setRejectOnOutsideTap(QDialog *dialog, bool isRejecting)
{
    if (!ConfirmationDialog_setRejectOnOutsideTap) {
        DebugLog::write("bridge: setRejectOnOutsideTap missing, dialog %p closes by its buttons only",
                        static_cast<void *>(dialog));
        return;
    }
    ConfirmationDialog_setRejectOnOutsideTap(dialog, isRejecting);
}

QDialog *showMessageDialog(const QString &title, const QString &text)
{
    if (!hasMessageDialog()) {
        return NULL;
    }
    // ConfirmationDialog renders the body as rich text; keep line breaks.
    QString body = text.toHtmlEscaped().replace(QLatin1Char('\n'), QStringLiteral("<br>"));
    QDialog *dialog = ConfirmationDialogFactory_showOKDialog(title, body);
    if (!dialog) {
        nh_log("bridge: showOKDialog returned null");
        return NULL;
    }
    setRejectOnOutsideTap(dialog, true);
    return dialog;
}

bool hasConfirmDialog()
{
    return isBridgeActive && ConfirmationDialogFactory_getConfirmationDialog && ConfirmationDialog_setTitle &&
           ConfirmationDialog_setText && ConfirmationDialog_setAcceptButtonText &&
           ConfirmationDialog_setRejectButtonText && ConfirmationDialog_showCloseButton;
}

QDialog *createConfirmDialog(const QString &title, const QString &text, const QString &acceptLabel,
                             const QString &rejectLabel, bool rejectsOnOutsideTap)
{
    if (!hasConfirmDialog()) {
        return NULL;
    }
    QDialog *dialog = ConfirmationDialogFactory_getConfirmationDialog(NULL);
    if (!dialog) {
        nh_log("bridge: getConfirmationDialog returned null");
        return NULL;
    }
    ConfirmationDialog_setTitle(dialog, title);
    ConfirmationDialog_setText(dialog, text.toHtmlEscaped());
    ConfirmationDialog_setAcceptButtonText(dialog, acceptLabel);
    ConfirmationDialog_setRejectButtonText(dialog, rejectLabel);
    ConfirmationDialog_showCloseButton(dialog, false);
    setRejectOnOutsideTap(dialog, rejectsOnOutsideTap);
    return dialog;
}

// ---- Touch menu (NickelMenu's proven recipe, nickelmenu.cc) ----

bool hasTouchMenu()
{
    return isBridgeActive && NickelTouchMenu_construct && MenuTextItem_construct && MenuTextItem_setText &&
           MenuTextItem_registerForTapGestures;
}

QMenu *createTouchMenu()
{
    if (!hasTouchMenu()) {
        return NULL;
    }
    void *memory = calloc(1, TOUCH_MENU_ALLOCATION);
    if (!memory) {
        nh_log("bridge: out of memory for NickelTouchMenu");
        return NULL;
    }
    NickelTouchMenu_construct(memory, NULL, TOUCH_MENU_DECORATION_POSITION);
    QMenu *menu = reinterpret_cast<QMenu *>(memory);
    // A plain bordered box: the decoration arrow would point at nothing.
    if (NickelTouchMenu_showDecoration) {
        NickelTouchMenu_showDecoration(menu, false);
    }
    DebugLog::write("bridge: NickelTouchMenu %p (%s)", memory, menu->metaObject()->className());
    return menu;
}

// One MenuTextItem row inside a QWidgetAction, separated from the previous
// row by the menu's own divider (TouchMenu::separator). A tappable row's
// tapped(bool) goes to receiver's slot; the title row ignores taps.
QWidget *addTouchMenuItem(QMenu *menu, const QString &text, bool isFirst, QObject *receiver, const char *tapSlot)
{
    void *memory = calloc(1, MENU_TEXT_ITEM_ALLOCATION);
    if (!memory) {
        nh_log("bridge: out of memory for MenuTextItem");
        return NULL;
    }
    MenuTextItem_construct(memory, menu, MENU_ITEM_CHECKABLE, MENU_ITEM_ITALIC);
    QWidget *item = reinterpret_cast<QWidget *>(memory);
    MenuTextItem_setText(item, text);
    if (receiver) {
        MenuTextItem_registerForTapGestures(item); // Nickel's pressed row: inverted while the finger is down
        if (!QObject::connect(item, SIGNAL(tapped(bool)), receiver, tapSlot)) {
            DebugLog::write("bridge: cannot connect %s::tapped(bool)", item->metaObject()->className());
            delete item;
            return NULL;
        }
    }
    if (!isFirst) {
        menu->addSeparator();
    }
    QWidgetAction *action = new QWidgetAction(menu);
    action->setDefaultWidget(item);
    action->setEnabled(true); // a disabled title row would be drawn greyed
    menu->addAction(action);
    return item;
}

// ---- Keyboard ----

bool hasKeyboardDialog()
{
    return hasViewPush() && N3DialogFactory_getDialog && N3Dialog_setTitle && N3Dialog_keyboardFrame &&
           N3Dialog_showKeyboard && KeyboardFrame_createKeyboard && KeyboardReceiver_construct &&
           SearchKeyboardController_setReceiver && SearchKeyboardController_setGoText;
}

// Nickel's KeyboardReceiver for lineEdit. It is never freed (NickelHardcover
// does the same): its ownership inside Nickel is unknown and it is small.
static void *createKeyboardReceiver(QLineEdit *lineEdit)
{
    void *receiver = calloc(1, KEYBOARD_RECEIVER_ALLOCATION);
    if (!receiver) {
        nh_log("bridge: out of memory for KeyboardReceiver");
        return NULL;
    }
    KeyboardReceiver_construct(receiver, lineEdit, false);
    return receiver;
}

static bool attachKeyboard(QWidget *dialog, QLineEdit *lineEdit, const QString &goText, QObject *receiver,
                           const char *commitSlot)
{
    QWidget *frame = N3Dialog_keyboardFrame(dialog);
    QObject *controller = frame ? KeyboardFrame_createKeyboard(frame, DEFAULT_KEYBOARD_SCRIPT, QLocale()) : NULL;
    void *keyboardReceiver = controller ? createKeyboardReceiver(lineEdit) : NULL;
    if (!keyboardReceiver) {
        nh_log("bridge: keyboard frame %p controller %p", (void *)frame, (void *)controller);
        return false;
    }
    SearchKeyboardController_setReceiver(controller, keyboardReceiver, false);
    SearchKeyboardController_setGoText(controller, goText);
    if (!QObject::connect(controller, SIGNAL(commitRequested()), receiver, commitSlot)) {
        nh_log("bridge: cannot connect commitRequested()");
        return false;
    }
    return true;
}

QWidget *openKeyboardDialog(QWidget *content, QLineEdit *lineEdit, const QString &title, const QString &goText,
                            QObject *receiver, const char *commitSlot, const char *closeSlot)
{
    void *window = hasKeyboardDialog() ? mainWindow() : NULL;
    if (!window || !content || !lineEdit || !receiver) {
        nh_log("bridge: keyboard dialog unavailable");
        return NULL;
    }
    QWidget *dialog = N3DialogFactory_getDialog(content, WITH_KEYBOARD);
    if (!dialog) {
        nh_log("bridge: N3DialogFactory::getDialog returned null");
        return NULL;
    }
    DebugLog::write("bridge: keyboard dialog %p (%s)", (void *)dialog, dialog->metaObject()->className());
    N3Dialog_setTitle(dialog, title);
    dialog->setProperty(OVERLAY_PROPERTY, true);
    QObject::connect(dialog, SIGNAL(closeTapped()), receiver, closeSlot);
    MainWindowController_pushView(window, dialog);
    dialog->show();
    if (!attachKeyboard(dialog, lineEdit, goText, receiver, commitSlot)) {
        MainWindowController_popView(window, dialog);
        return NULL;
    }
    N3Dialog_showKeyboard(dialog);
    lineEdit->setFocus(Qt::OtherFocusReason);
    return dialog;
}

void closeKeyboardDialog(QWidget *dialog)
{
    popView(dialog);
}

// ---- Settings-style modal ----

bool hasSettingsDialog()
{
    return isBridgeActive && N3DialogFactory_getDialog && N3Dialog_setTitle;
}

QWidget *createSettingsDialog(QWidget *content, const QString &title)
{
    if (!hasSettingsDialog() || !content) {
        nh_log("bridge: N3Dialog unavailable");
        return NULL;
    }
    QWidget *dialog = N3DialogFactory_getDialog(content, WITHOUT_PAGE_MENU);
    if (!dialog) {
        nh_log("bridge: N3DialogFactory::getDialog returned null");
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

// ---- Power ----

bool hasPowerAction(PowerAction action)
{
    bool hasManager = isBridgeActive && N3PowerWorkflowManager_sharedInstance;
    return hasManager && (action == POWER_SLEEP ? N3PowerWorkflowManager_requestSleep != NULL
                                                : N3PowerWorkflowManager_reboot != NULL);
}

bool runPowerAction(PowerAction action)
{
    void *manager = hasPowerAction(action) ? N3PowerWorkflowManager_sharedInstance() : NULL;
    if (!manager) {
        nh_log("bridge: power action %s unavailable", action == POWER_SLEEP ? "sleep" : "reboot");
        return false;
    }
    nh_log("bridge: power action %s", action == POWER_SLEEP ? "sleep" : "reboot");
    if (action == POWER_SLEEP) {
        N3PowerWorkflowManager_requestSleep(manager);
    } else {
        N3PowerWorkflowManager_reboot(manager);
    }
    return true;
}

// ---- Wi-Fi and library ----

bool hasWifi()
{
    return isBridgeActive && WirelessWorkflowManager_sharedInstance && WirelessWorkflowManager_isInternetAccessible &&
           WirelessWorkflowManager_connectWireless;
}

static void *wirelessManager()
{
    void *manager = hasWifi() ? WirelessWorkflowManager_sharedInstance() : NULL;
    if (!manager) {
        nh_log("bridge: WirelessWorkflowManager unavailable");
    }
    return manager;
}

bool isInternetAccessible()
{
    void *manager = wirelessManager();
    return manager && WirelessWorkflowManager_isInternetAccessible(manager);
}

bool requestWifiConnect()
{
    void *manager = wirelessManager();
    if (!manager) {
        return false;
    }
    WirelessWorkflowManager_connectWireless(manager, CONNECT_WIRELESS_FIRST_ARGUMENT,
                                            CONNECT_WIRELESS_SECOND_ARGUMENT);
    return true;
}

bool hasRescan(RescanKind kind)
{
    bool hasManager = isBridgeActive && PlugWorkflowManager_sharedInstance;
    if (kind == RESCAN_FULL) {
        return hasManager && PlugWorkflowManager_onCancelAndConnect && PlugWorkflowManager_unplugged;
    }
    return hasManager && PlugWorkflowManager_sync;
}

bool rescanLibrary(RescanKind kind)
{
    void *manager = hasRescan(kind) ? PlugWorkflowManager_sharedInstance() : NULL;
    if (!manager) {
        nh_log("bridge: %s rescan unavailable", kind == RESCAN_FULL ? "full" : "quick");
        return false;
    }
    if (kind == RESCAN_QUICK) {
        PlugWorkflowManager_sync(manager);
        return true;
    }
    // Same sequence as NickelMenu's rescan_books_full, with the 1 s pause on
    // a timer. The finisher deletes itself.
    PlugWorkflowManager_onCancelAndConnect(manager);
    QTimer::singleShot(FULL_RESCAN_UNPLUG_DELAY_MILLISECONDS, new FullRescanFinisher(manager), SLOT(finish()));
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

FullRescanFinisher::FullRescanFinisher(void *plugWorkflowManager) : QObject(NULL), manager(plugWorkflowManager) {}

void FullRescanFinisher::finish()
{
    if (PlugWorkflowManager_unplugged) {
        PlugWorkflowManager_unplugged(manager);
    }
    deleteLater();
}

// NickelHook PLT hook target for MoreView::MoreView(QWidget*). Always calls
// the original first; adds our row only while the bridge is active.
extern "C" __attribute__((visibility("default"))) void nickeldrawer_MoreView_construct(void *moreView, QWidget *parent)
{
    MoreView_construct_original(moreView, parent);
    DebugLog::write("bridge: MoreView constructed %p", moreView);
    if (isBridgeActive && moreViewCreatedHandler) {
        // MoreView derives from QWidget as its first base.
        moreViewCreatedHandler(reinterpret_cast<QWidget *>(moreView));
    }
}
