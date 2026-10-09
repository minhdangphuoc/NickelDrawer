#include "action_runner.h"

#include <NickelHook.h>

#include "dialogs.h"
#include "drawer_logic.h"
#include "fallback_strings.h"
#include "model_client.h"
#include "nickel_bridge.h"
#include "wifi_connector.h"

// Menu ids of the drawer's own entries; '#' cannot start an app action id.
static const char *const MENU_INFO_ID = "#info";
static const char *const MENU_REMOVE_ID = "#remove";
// The app's status action feeds the model's status line; it is not a menu entry.
static const char *const STATUS_ACTION_ID = "status";
static const char *const RESCAN_QUICK = "quick";
static const char *const RESCAN_FULL = "full";

static DialogChoice choice(const QString &id, const QString &label)
{
    DialogChoice entry = {id, label};
    return entry;
}

ActionRunner::ActionRunner(ModelClient *modelClient, QObject *parent)
    : QObject(parent), client(modelClient), hasRetriedAfterWifi(false), isBusy(false)
{
    connect(client, SIGNAL(actionFinished(ActionResult)), this, SLOT(onActionFinished(ActionResult)));
    connect(client, SIGNAL(infoLoaded(bool, AppInfo, QString)), this, SLOT(onInfoLoaded(bool, AppInfo, QString)));
    connect(client, SIGNAL(removeFinished(ActionResult)), this, SLOT(onRemoveFinished(ActionResult)));
}

void ActionRunner::setBusy(bool busy)
{
    if (isBusy != busy) {
        isBusy = busy;
        emit busyChanged(busy);
    }
}

// ---- Tap ----

void ActionRunner::runPrimary(const DrawerItem &item)
{
    if (isBusy || item.isProblem) {
        return;
    }
    if (!item.hasPrimaryAction()) {
        openMenu(item);
        return;
    }
    currentItem = item;
    hasRetriedAfterWifi = false;
    setBusy(true);
    runAction(item.actions.first());
}

void ActionRunner::runAction(const DrawerAction &action)
{
    currentAction = action;
    nh_log("action: %s %s", qPrintable(currentItem.id), qPrintable(action.id));
    client->requestRun(currentItem.id, action);
}

void ActionRunner::onActionFinished(const ActionResult &result)
{
    switch (DrawerLogic::nextStepAfterRun(result.needsWifi, hasRetriedAfterWifi)) {
    case DrawerLogic::RUN_ASK_FOR_WIFI:
        askForWifi();
        return;
    case DrawerLogic::RUN_GIVE_UP_WIFI:
        Dialogs::showMessage(currentItem.name, FallbackStrings::text(FallbackStrings::WIFI_FAILED));
        finishAndReload();
        return;
    case DrawerLogic::RUN_SHOW_RESULT:
        presentResult(result);
        rescanIfAsked(result);
        finishAndReload();
        return;
    }
}

void ActionRunner::askForWifi()
{
    ConfirmQuestion *question =
        new ConfirmQuestion(currentItem.name, FallbackStrings::text(FallbackStrings::WIFI_OFF_TEXT),
                            FallbackStrings::text(FallbackStrings::WIFI_TURN_ON), FallbackStrings::text(FallbackStrings::CANCEL));
    connect(question, SIGNAL(answered(bool)), this, SLOT(onWifiAnswer(bool)));
    question->ask();
}

void ActionRunner::onWifiAnswer(bool isAccepted)
{
    if (!isAccepted) {
        setBusy(false); // cancel returns to the drawer unchanged
        return;
    }
    WifiConnector *connector = new WifiConnector(this);
    connect(connector, SIGNAL(connected()), this, SLOT(onWifiConnected()));
    connect(connector, SIGNAL(failed()), this, SLOT(onWifiFailed()));
    connector->start();
}

void ActionRunner::onWifiConnected()
{
    hasRetriedAfterWifi = true;
    runAction(currentAction);
}

void ActionRunner::onWifiFailed()
{
    Dialogs::showMessage(currentItem.name, FallbackStrings::text(FallbackStrings::WIFI_FAILED));
    setBusy(false);
}

void ActionRunner::presentResult(const ActionResult &result)
{
    bool isToast = currentAction.isBackground && result.isOk;
    if (isToast && NickelBridge::hasToast()) {
        NickelBridge::showToast(result.text.isEmpty() ? currentAction.label : result.text);
        return;
    }
    if (!result.text.isEmpty()) {
        Dialogs::showMessage(currentItem.name, result.text);
    }
}

void ActionRunner::rescanIfAsked(const ActionResult &result)
{
    if (!result.isOk) {
        return;
    }
    if (result.rescan == QLatin1String(RESCAN_QUICK)) {
        NickelBridge::rescanLibrary(NickelBridge::RESCAN_QUICK);
    } else if (result.rescan == QLatin1String(RESCAN_FULL)) {
        NickelBridge::rescanLibrary(NickelBridge::RESCAN_FULL);
    }
}

void ActionRunner::finishAndReload()
{
    setBusy(false);
    emit modelMayHaveChanged();
}

// ---- Long-press ----

void ActionRunner::openMenu(const DrawerItem &item)
{
    if (isBusy) {
        return;
    }
    // Items that do not apply are hidden, not greyed (ui-kit §3.16).
    QList<DialogChoice> choices;
    bool isApp = !item.isHostItem() && !item.id.isEmpty();
    if (isApp && !item.isProblem) {
        choices << choice(QString::fromLatin1(MENU_INFO_ID), FallbackStrings::text(FallbackStrings::MENU_INFO));
    }
    for (int i = 1; i < item.actions.size(); i++) {
        if (item.actions[i].id != QLatin1String(STATUS_ACTION_ID)) {
            choices << choice(item.actions[i].id, item.actions[i].label);
        }
    }
    if (isApp) {
        choices << choice(QString::fromLatin1(MENU_REMOVE_ID), FallbackStrings::text(FallbackStrings::MENU_REMOVE));
    }
    if (choices.isEmpty()) {
        return;
    }
    currentItem = item;
    hasRetriedAfterWifi = false;
    setBusy(true);
    PlainDialog *menu = createRowMenu(item.name, choices);
    connect(menu, SIGNAL(chosen(QString)), this, SLOT(onMenuChoice(QString)));
    menu->present();
}

void ActionRunner::onMenuChoice(const QString &choiceId)
{
    if (choiceId.isEmpty()) {
        setBusy(false);
    } else if (choiceId == QLatin1String(MENU_INFO_ID)) {
        client->requestInfo(currentItem.id);
    } else if (choiceId == QLatin1String(MENU_REMOVE_ID)) {
        askToRemove();
    } else {
        for (int i = 0; i < currentItem.actions.size(); i++) {
            if (currentItem.actions[i].id == choiceId) {
                runAction(currentItem.actions[i]);
                return;
            }
        }
        nh_log("action: menu choice %s no longer exists", qPrintable(choiceId));
        setBusy(false);
    }
}

void ActionRunner::onInfoLoaded(bool isLoaded, const AppInfo &info, const QString &failure)
{
    if (isLoaded) {
        Dialogs::showMessage(info.name.isEmpty() ? currentItem.name : info.name, formatInfo(info));
    } else {
        Dialogs::showMessage(FallbackStrings::text(FallbackStrings::ERROR_TITLE), failure);
    }
    setBusy(false);
}

QString ActionRunner::formatInfo(const AppInfo &info) const
{
    using FallbackStrings::text;
    QString separator = QStringLiteral(", ");
    QStringList lines;
    if (!info.description.isEmpty()) {
        lines << info.description << QString();
    }
    lines << text(FallbackStrings::INFO_VERSION) + QStringLiteral(": ") + info.version;
    if (!info.reads.isEmpty()) {
        lines << text(FallbackStrings::INFO_READS) + QStringLiteral(": ") + info.reads.join(separator);
    }
    if (!info.writes.isEmpty()) {
        lines << text(FallbackStrings::INFO_WRITES) + QStringLiteral(": ") + info.writes.join(separator);
    }
    lines << text(FallbackStrings::INFO_NETWORK) + QStringLiteral(": ") +
                 text(info.usesNetwork ? FallbackStrings::YES : FallbackStrings::NO);
    lines << text(FallbackStrings::INFO_FOLDER) + QStringLiteral(": ") + info.path;
    return lines.join(QLatin1Char('\n'));
}

void ActionRunner::askToRemove()
{
    QString question = FallbackStrings::text(FallbackStrings::REMOVE_CONFIRM_TEXT);
    question.replace(QStringLiteral("{name}"), currentItem.name);
    ConfirmQuestion *confirm = new ConfirmQuestion(currentItem.name, question,
                                                   FallbackStrings::text(FallbackStrings::REMOVE_BUTTON),
                                                   FallbackStrings::text(FallbackStrings::CANCEL));
    connect(confirm, SIGNAL(answered(bool)), this, SLOT(onRemoveAnswer(bool)));
    confirm->ask();
}

void ActionRunner::onRemoveAnswer(bool isAccepted)
{
    if (!isAccepted) {
        setBusy(false);
        return;
    }
    client->requestRemove(currentItem.id);
}

void ActionRunner::onRemoveFinished(const ActionResult &result)
{
    if (!result.text.isEmpty()) {
        Dialogs::showMessage(currentItem.name, result.text);
    }
    finishAndReload();
}
