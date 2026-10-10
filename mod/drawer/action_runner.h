#ifndef NICKELDRAWER_ACTION_RUNNER_H
#define NICKELDRAWER_ACTION_RUNNER_H

#include <QObject>
#include <QPoint>

#include "drawer_model.h"

class HostRequestRunner;
class ModelClient;

// What a tap or long-press on a drawer item does (docs/spec/drawer-protocol.md,
// ui-kit §3.15–3.18, §5.4):
//   tap        → run the first action; needs_wifi → ask, connect, retry once;
//                show the result (dialog for "show", toast for "background");
//                perform the reply's `requests` and `rescan` (Wi-Fi first, one rescan).
//   long-press → menu at the pressed item: Info, the other actions, Remove…
//                last (with confirm).
//   screen tap → `drawer event` with the same Wi-Fi, result and rescan rules;
//                a toast or dialog declared on the screen is shown as such.
// One thing at a time: while busy, the view ignores input.
class ActionRunner : public QObject {
    Q_OBJECT
public:
    ActionRunner(ModelClient *client, QObject *parent);

    void runPrimary(const DrawerItem &item);
    // anchor: the pressed item's centre, global; null: the screen centre.
    void openMenu(const DrawerItem &item, const QPoint &anchor);
    void runScreenEvent(const DrawerItem &app, const ScreenEvent &event);
    bool isRunning() const { return isBusy; }
    // Wi-Fi then rescan, as a screen reply's `requests` ask (title: Wi-Fi dialog heading).
    void performScreenRequests(const QList<HostRequest> &requests, const QString &title);

signals:
    void busyChanged(bool isBusy);
    // Something may have changed; reload the model.
    void modelMayHaveChanged();

private slots:
    void onActionFinished(const ActionResult &result);
    void onWifiAnswer(bool isAccepted);
    void onWifiConnected();
    void onWifiFailed();
    void onMenuChoice(const QString &choiceId);
    void onInfoLoaded(bool isLoaded, const AppInfo &info, const QString &failure);
    void onRemoveAnswer(bool isAccepted);
    void onRemoveFinished(const ActionResult &result);
    void onDeclaredDialogAnswer(bool isAccepted);
    void onActionConfirmAnswer(bool isAccepted);

private:
    void setBusy(bool isBusy);
    void showProblemText(const DrawerItem &item);
    void runAction(const DrawerAction &action);
    void askToConfirmAction(const DrawerAction &action);
    void runCurrent();
    bool presentScreenSignal(const ActionResult &result);
    void presentScreenEventOutcome(const ActionResult &result);
    void askForWifi();
    void presentResult(const ActionResult &result);
    void performRequestsOf(const ActionResult &result);
    void finishAndReload();
    void askToRemove();
    QString formatInfo(const AppInfo &info) const;

    ModelClient *client;
    HostRequestRunner *hostRequests;
    DrawerItem currentItem;
    DrawerAction currentAction;
    bool isScreenEvent;
    ScreenEvent currentEvent;
    DeclaredDialog pendingDialog;
    bool hasRetriedAfterWifi;
    bool isBusy;
};

#endif
