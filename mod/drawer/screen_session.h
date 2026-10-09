#ifndef NICKELDRAWER_SCREEN_SESSION_H
#define NICKELDRAWER_SCREEN_SESSION_H

#include <QObject>
#include <QStringList>

#include "drawer_model.h"
#include "screen_model.h"

class ActionRunner;
class ModelClient;
class QTimer;
class ScreenView;

// One open app screen (Phase 4, flag .adds/nickeldrawer/screens): loads it
// with `appsctl drawer screen`, shows it in a ScreenView over the drawer page
// and turns taps into `drawer event` runs through the drawer's ActionRunner
// (same Wi-Fi, dialog and rescan rules as a drawer tap).
//   button / row / empty → its action (button confirm asked first)
//   toggle / radio       → shown at once, then the action named like the
//                          variable with NICKELDRAWER_VALUE
//   field                → Nickel's keyboard, else an "edit on your phone" notice
//   list row (tap)       → the list's tap action with the row's primary text
//   header action        → push another screen, or run an action
// Back pops pushed screens, then closes. A live screen (state action) is
// re-read every 5 s while idle. Child of its view: it ends with the view.
class ScreenSession : public QObject {
    Q_OBJECT
public:
    ScreenSession(const DrawerItem &app, QWidget *drawerView, ActionRunner *runner);

    void start();

signals:
    // The screen view is gone; the drawer underneath is visible again.
    void closed();

private slots:
    void reload();
    void onScreenLoaded(bool isLoaded, const ScreenModel &screen, const QString &failure);
    void onBackRequested();
    void onHeaderActionTapped();
    void onComponentTapped(int component, const QString &value);
    void onConfirmAnswer(bool isAccepted);
    void onFieldCommitted(const QString &value);
    void onFieldFinished();
    void onBusyChanged(bool isBusy);
    void onLiveTimeout();

private:
    void load(const QString &screenId);
    void showLoadError(const QString &failure);
    void runAction(const QString &actionId);
    void runWithValue(const QString &actionId, const QString &value);
    void runEvent(const ScreenEvent &event);
    void tapButton(const ScreenComponent &button);
    void tapToggle(int index, const ScreenComponent &toggle);
    void tapRadio(int index, const ScreenComponent &radio, const QString &option);
    void tapField(const ScreenComponent &field);
    void close();
    QString currentScreenId() const;

    DrawerItem app;
    ActionRunner *runner;
    ModelClient *client;
    ScreenView *view;
    QTimer *liveTimer;
    QStringList screenStack; // pushed screen ids; empty entry = the start screen
    bool isLoading;
    bool isEditingField;
    bool isConfirmOpen;
    QString pendingConfirmAction;
    QString editedFieldAction;
};

#endif
