#include "tweaks_session.h"

#include <QApplication>
#include <QDialog>
#include <QEvent>
#include <QKeyEvent>
#include <QTimer>
#include <QTouchEvent>
#include <QVariant>
#include <QWidget>

#include <NickelHook.h>

#include "confirm_question.h"
#include "debug_log.h"
#include "helper_strings.h"
#include "nickel_bridge.h"
#include "page_controller.h"
#include "screenshot_button.h"
#include "tweaks_client.h"
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

static void showToast(const QString &text)
{
    NickelBridge::showToast(text, QStringLiteral(""));
}

void TweaksSession::open()
{
    if (openSession) {
        DebugLog::write("tweaks: already open, ignoring tap");
        return;
    }
    HelperStrings::reloadLocale();
    TweaksSession *session = new TweaksSession();
    if (!session->present()) {
        delete session;
        showToast(HelperStrings::text(HelperStrings::TWEAKS_UNAVAILABLE));
        return;
    }
    openSession = session;
    session->client->requestList();
}

TweaksSession::TweaksSession()
    : QObject(NULL), client(new TweaksClient(this)), isBusy(false), isCloseScheduled(false),
      hasTouchBegunOutside(false)
{
    connect(client, SIGNAL(listLoaded(TweaksModel)), this, SLOT(onTweaksLoaded(TweaksModel)));
    connect(client, SIGNAL(applied(TweakResult)), this, SLOT(onTweakApplied(TweakResult)));
}

TweaksSession::~TweaksSession()
{
    if (openSession == this) {
        openSession = NULL;
    }
    DebugLog::write("tweaks: session closed");
}

// Settings' own recipe: N3Dialog as the view of a controller with no tab bar
// (N3DialogController::viewAsDialog + MainWindowController::push). There is no stand-in window: when
// any part is missing the panel does not open and the user is told once.
bool TweaksSession::present()
{
    if (!NickelBridge::hasSettingsDialog() || !NickelBridge::hasControllerPush()) {
        DebugLog::write("tweaks: Nickel's N3Dialog or controller push missing, no panel");
        return false;
    }
    panel = new TweaksPanel(NULL);
    connect(panel, SIGNAL(tweakTapped(QString)), this, SLOT(onTweakTapped(QString)));
    QWidget *dialog = NickelBridge::createSettingsDialog(panel, HelperStrings::text(HelperStrings::TWEAKS_TITLE));
    if (!dialog) {
        delete panel.data();
        return false;
    }
    for (size_t i = 0; i < sizeof(STATUS_BAR_PROPERTIES) / sizeof(STATUS_BAR_PROPERTIES[0]); i++) {
        dialog->setProperty(STATUS_BAR_PROPERTIES[i], false);
    }
    PageController *dialogController = new PageController(dialog, PageController::NAV_SECTION_NONE);
    bool isClosable = connect(dialog, SIGNAL(closeTapped()), this, SLOT(onCloseRequested()));
    if (!dialogController->hasNickelLayout() || !isClosable) {
        DebugLog::write("tweaks: controller layout %s, closeTapped() %s; no panel",
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
        DebugLog::write("tweaks: push did not add the dialog to the window stack, no panel");
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
        askThenApply(*item);
        return;
    }
    applyTweak(tweakId);
}

// Without Nickel's confirmation dialog the action is not run: it was guarded for a reason.
void TweaksSession::askThenApply(const TweakItem &item)
{
    ConfirmQuestion *asked = new ConfirmQuestion(item.label, item.confirm,
                                                 HelperStrings::text(HelperStrings::CONFIRM_OK),
                                                 HelperStrings::text(HelperStrings::CANCEL));
    connect(asked, SIGNAL(answered(bool)), this, SLOT(onConfirmAnswer(bool)));
    pendingTweakId = item.id;
    setBusy(true);
    if (!asked->ask()) {
        delete asked;
        setBusy(false);
        showToast(HelperStrings::text(HelperStrings::NICKEL_UNAVAILABLE));
        return;
    }
    question = asked;
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
    client->requestApply(tweakId);
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
        showToast(result.error);
        return;
    }
    const TweakItem *item = panel ? panel->item(pendingTweakId) : NULL;
    if (item && result.hasValue && item->kind == TweaksLogic::KIND_TOGGLE) {
        TweakItem updated = *item;
        updated.isOn = result.value;
        panel->updateItem(updated);
    }
    ScreenshotButton::refreshAll(); // the "screenshot-button" switch applies to the status bar at once
    showResult(result);
}

void TweaksSession::showResult(const TweakResult &result)
{
    if (!result.toast.isEmpty()) {
        showToast(result.toast);
    }
    runNickelAction(result);
}

void TweaksSession::runNickelAction(const TweakResult &result)
{
    bool isAvailable = true;
    TweaksLogic::NickelAction action = result.nickel;
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
        isAvailable = NickelBridge::rescanLibrary();
        break;
    case TweaksLogic::NICKEL_UNKNOWN:
        nh_log("tweaks: reply asks for an unknown Nickel action");
        isAvailable = false;
        break;
    }
    if (!isAvailable) {
        showToast(HelperStrings::text(HelperStrings::NICKEL_UNAVAILABLE));
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
    if (question) {
        question->cancel();
    }
    // Nickel may have removed the page meanwhile (popView hides it first).
    if (controller && nickelDialog && nickelDialog->isVisible()) {
        DebugLog::write("tweaks: closing, pop controller");
        NickelBridge::popController(controller->asNickelController());
    }
}

void TweaksSession::onModalDestroyed()
{
    DebugLog::write("tweaks: dialog destroyed");
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
