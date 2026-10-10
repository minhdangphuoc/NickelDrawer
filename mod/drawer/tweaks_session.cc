#include "tweaks_session.h"

#include <QDialog>
#include <QEvent>
#include <QKeyEvent>
#include <QTimer>
#include <QTouchEvent>

#include <NickelHook.h>

#include "debug_log.h"
#include "dialogs.h"
#include "drawer_controller.h"
#include "fallback_strings.h"
#include "model_client.h"
#include "nickel_bridge.h"
#include "open_popups.h"
#include "tweaks_overlay.h"
#include "tweaks_panel.h"

static const int RUN_FROM_EVENT_LOOP = 0;
// Long enough for Nickel to put the toast on the panel before the GUI thread
// stops (reboot) or the sleep screen covers it.
static const int POWER_ACTION_DELAY_MILLISECONDS = 1500;
// N3Dialog's white frame inside its #4d4d4d surround (Ui_N3Dialog, N3Dialog.qss).
static const char *const NICKEL_FRAME_NAME = "mainFrame";
// MainWindowController::showStatusBarAsNecessary reads these from the pushed
// view: all false hides Nickel's status bar, as for its own N3Dialog pages
// (the dialog has its own battery icon in its top row).
static const char *const STATUS_BAR_PROPERTIES[] = {"showTitle", "showIndicators", "showStatusBottomLine"};

static TweaksSession *openSession = NULL;

void TweaksSession::open()
{
    if (openSession) {
        DebugLog::write("tweaks: already open, ignoring tap");
        return;
    }
    FallbackStrings::reloadLocale();
    TweaksSession *session = new TweaksSession();
    if (!session->present()) {
        delete session;
        NickelBridge::showToast(FallbackStrings::text(FallbackStrings::DRAWER_UNAVAILABLE));
        return;
    }
    openSession = session;
    session->client->requestTweaks();
}

TweaksSession::TweaksSession()
    : QObject(NULL), client(new ModelClient(this)), isBusy(false), isCloseScheduled(false),
      hasTouchBegunOutside(false)
{
    connect(client, SIGNAL(tweaksLoaded(TweaksModel)), this, SLOT(onTweaksLoaded(TweaksModel)));
    connect(client, SIGNAL(tweakApplied(TweakResult)), this, SLOT(onTweakApplied(TweakResult)));
}

TweaksSession::~TweaksSession()
{
    if (openSession == this) {
        openSession = NULL;
    }
    DebugLog::write("tweaks: session closed");
}

bool TweaksSession::present()
{
    panel = new TweaksPanel(NULL);
    connect(panel, SIGNAL(tweakTapped(QString)), this, SLOT(onTweakTapped(QString)));
    if (presentInNickelDialog()) {
        return true;
    }
    if (!panel) { // a failed Nickel dialog took the panel with it
        panel = new TweaksPanel(NULL);
        connect(panel, SIGNAL(tweakTapped(QString)), this, SLOT(onTweakTapped(QString)));
    }
    presentInOwnOverlay();
    return ownOverlay != NULL;
}

// Settings' own recipe: N3Dialog as the view of a controller with no tab bar
// (N3DialogController::viewAsDialog + MainWindowController::push).
bool TweaksSession::presentInNickelDialog()
{
    if (!NickelBridge::hasSettingsDialog() || !NickelBridge::hasControllerPush()) {
        DebugLog::write("tweaks: Nickel's N3Dialog or controller push missing, own overlay");
        return false;
    }
    QWidget *dialog =
        NickelBridge::createSettingsDialog(panel, FallbackStrings::text(FallbackStrings::TWEAKS_TITLE));
    if (!dialog) {
        return false;
    }
    for (size_t i = 0; i < sizeof(STATUS_BAR_PROPERTIES) / sizeof(STATUS_BAR_PROPERTIES[0]); i++) {
        dialog->setProperty(STATUS_BAR_PROPERTIES[i], false);
    }
    DrawerController *dialogController = new DrawerController(dialog, DrawerController::NAV_SECTION_NONE);
    bool isClosable = connect(dialog, SIGNAL(closeTapped()), this, SLOT(onCloseRequested()));
    if (!dialogController->hasNickelLayout() || !isClosable) {
        DebugLog::write("tweaks: controller layout %s, closeTapped() %s; own overlay",
                        dialogController->hasNickelLayout() ? "ok" : "unexpected", isClosable ? "ok" : "missing");
        delete dialogController;
        delete dialog; // and the panel with it
        return false;
    }
    nickelDialog = dialog;
    controller = dialogController;
    watchModal(dialog);
    NickelBridge::pushController(dialogController->asNickelController());
    if (!dialog->parentWidget()) {
        DebugLog::write("tweaks: push did not add the dialog to the window stack, own overlay");
        disconnect(dialog, SIGNAL(destroyed()), this, SLOT(onModalDestroyed()));
        delete controller;
        delete dialog;
        return false;
    }
    if (controller && controller->parent() != dialog) {
        controller->setParent(dialog); // as Nickel does: the controller goes with its view
    }
    DebugLog::write("tweaks: Nickel N3Dialog %p pushed", static_cast<void *>(dialog));
    return true;
}

void TweaksSession::presentInOwnOverlay()
{
    TweaksOverlay *overlay = new TweaksOverlay(FallbackStrings::text(FallbackStrings::TWEAKS_TITLE), panel);
    ownOverlay = overlay;
    watchModal(overlay);
    overlay->present();
    DebugLog::write("tweaks: own overlay %p shown", static_cast<void *>(overlay));
}

void TweaksSession::watchModal(QWidget *modalWidget)
{
    connect(modalWidget, SIGNAL(destroyed()), this, SLOT(onModalDestroyed()));
    modalWidget->installEventFilter(this);
}

void TweaksSession::onTweaksLoaded(const TweaksModel &model)
{
    if (!panel) {
        return;
    }
    if (!model.isUsable) {
        DebugLog::write("tweaks: list unavailable: %s", qPrintable(model.failure));
        panel->showMessage(model.failure);
        return;
    }
    panel->showItems(model.items);
    NickelBridge::requestFullRefresh(panel);
}

void TweaksSession::onTweakTapped(const QString &tweakId)
{
    const TweakItem *item = panel ? panel->item(tweakId) : NULL;
    if (isBusy || !item || !item->isTappable()) {
        return;
    }
    if (item->kind == TweaksLogic::KIND_ACTION && !item->confirm.isEmpty()) {
        pendingTweakId = tweakId;
        setBusy(true);
        ConfirmQuestion *question = new ConfirmQuestion(
            item->label, item->confirm, FallbackStrings::text(FallbackStrings::TWEAK_CONFIRM_OK),
            FallbackStrings::text(FallbackStrings::CANCEL), ConfirmQuestion::NOT_DESTRUCTIVE);
        connect(question, SIGNAL(answered(bool)), this, SLOT(onConfirmAnswer(bool)));
        question->ask();
        return;
    }
    applyTweak(tweakId);
}

void TweaksSession::onConfirmAnswer(bool isAccepted)
{
    setBusy(false);
    DebugLog::write("tweaks: \"%s\" %s", qPrintable(pendingTweakId), isAccepted ? "confirmed" : "cancelled");
    if (isAccepted) {
        applyTweak(pendingTweakId);
    }
}

void TweaksSession::applyTweak(const QString &tweakId)
{
    pendingTweakId = tweakId;
    setBusy(true);
    client->requestTweak(tweakId);
}

void TweaksSession::setBusy(bool busy)
{
    isBusy = busy;
    if (panel) {
        panel->setInputEnabled(!busy);
    }
}

void TweaksSession::onTweakApplied(const TweakResult &result)
{
    setBusy(false);
    DebugLog::write("tweaks: \"%s\" %s", qPrintable(pendingTweakId), result.isOk ? "applied" : "failed");
    if (!result.isOk) {
        NickelBridge::showToast(result.error);
        return;
    }
    const TweakItem *item = panel ? panel->item(pendingTweakId) : NULL;
    if (item && result.hasValue && item->kind == TweaksLogic::KIND_TOGGLE) {
        TweakItem updated = *item;
        updated.isOn = result.value;
        panel->updateItem(updated);
    }
    showResult(result);
}

void TweaksSession::showResult(const TweakResult &result)
{
    if (!result.toast.isEmpty()) {
        NickelBridge::showToast(result.toast);
    }
    runNickelAction(result.nickel);
}

void TweaksSession::runNickelAction(TweaksLogic::NickelAction action)
{
    bool isAvailable = true;
    switch (action) {
    case TweaksLogic::NICKEL_NONE:
        return;
    case TweaksLogic::NICKEL_SLEEP:
    case TweaksLogic::NICKEL_REBOOT: {
        NickelBridge::PowerAction power =
            action == TweaksLogic::NICKEL_SLEEP ? NickelBridge::POWER_SLEEP : NickelBridge::POWER_REBOOT;
        isAvailable = NickelBridge::hasPowerAction(power);
        if (isAvailable) {
            DeferredPowerAction::start(power);
        }
        break;
    }
    case TweaksLogic::NICKEL_RESCAN_LIBRARY:
        // Quick: PlugWorkflowManager::sync, NickelMenu's rescan_books. The full
        // rescan fakes a USB session and clears the window stack.
        isAvailable = NickelBridge::rescanLibrary(NickelBridge::RESCAN_QUICK);
        break;
    case TweaksLogic::NICKEL_UNKNOWN:
        nh_log("tweaks: reply asks for an unknown Nickel action");
        isAvailable = false;
        break;
    }
    if (!isAvailable) {
        NickelBridge::showToast(FallbackStrings::text(FallbackStrings::NICKEL_ACTION_UNAVAILABLE));
    }
}

void TweaksSession::onCloseRequested()
{
    if (isCloseScheduled) {
        return;
    }
    isCloseScheduled = true;
    QTimer::singleShot(RUN_FROM_EVENT_LOOP, this, SLOT(closeModal()));
}

void TweaksSession::closeModal()
{
    isCloseScheduled = false;
    OpenPopups::closeAll("tweaks closed");
    if (ownOverlay) {
        ownOverlay->reject();
        return;
    }
    // Nickel may have removed the page meanwhile (popView hides it first).
    if (controller && nickelDialog && nickelDialog->isVisible()) {
        DebugLog::write("tweaks: closing, pop controller");
        NickelBridge::popController(controller->asNickelController());
    }
}

void TweaksSession::onModalDestroyed()
{
    DebugLog::write("tweaks: modal destroyed");
    deleteLater(); // pending reads are killed, a running tweak finishes on its own
}

bool TweaksSession::isOutsideNickelFrame(const QPoint &dialogPosition) const
{
    QWidget *frame = nickelDialog ? nickelDialog->findChild<QWidget *>(QString::fromLatin1(NICKEL_FRAME_NAME)) : NULL;
    if (!frame) {
        return false; // no frame known: no outside
    }
    QRect frameRect(frame->mapTo(nickelDialog.data(), QPoint(0, 0)), frame->size());
    return !frameRect.contains(dialogPosition);
}

// On Nickel's dialog: page keys page the panel (TactileButtonFilter sends them
// to the current view); a touch that starts and ends on the grey surround
// closes the modal. Touches inside the frame are left to Nickel and to us.
bool TweaksSession::eventFilter(QObject *watched, QEvent *event)
{
    if (watched != nickelDialog.data() || !panel) {
        return QObject::eventFilter(watched, event);
    }
    switch (event->type()) {
    case QEvent::KeyPress:
        return panel->handlePageKey(static_cast<QKeyEvent *>(event)->key());
    case QEvent::TouchBegin:
    case QEvent::TouchEnd: {
        QTouchEvent *touch = static_cast<QTouchEvent *>(event);
        if (touch->touchPoints().isEmpty()) {
            break;
        }
        bool isOutside = isOutsideNickelFrame(touch->touchPoints().first().pos().toPoint());
        if (event->type() == QEvent::TouchBegin) {
            hasTouchBegunOutside = isOutside;
        } else if (hasTouchBegunOutside && isOutside && !isBusy) {
            DebugLog::write("tweaks: tap outside the frame, closing");
            onCloseRequested();
        }
        break;
    }
    default:
        break;
    }
    return QObject::eventFilter(watched, event);
}

// ---- Deferred power action ----

void DeferredPowerAction::start(int powerAction)
{
    DeferredPowerAction *deferred = new DeferredPowerAction(powerAction);
    QTimer::singleShot(POWER_ACTION_DELAY_MILLISECONDS, deferred, SLOT(run()));
}

DeferredPowerAction::DeferredPowerAction(int powerAction) : QObject(NULL), action(powerAction) {}

void DeferredPowerAction::run()
{
    NickelBridge::runPowerAction(static_cast<NickelBridge::PowerAction>(action));
    deleteLater();
}
