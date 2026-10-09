#ifndef NICKELDRAWER_DRAWER_SESSION_H
#define NICKELDRAWER_DRAWER_SESSION_H

#include <QObject>
#include <QPointer>

#include "drawer_model.h"

class ActionRunner;
class DrawerView;
class ModelClient;
class ScreenSession;

// One open drawer: loads the model into the view and routes the view's taps
// to the action runner. Child of the view, so it ends when the view does.
class DrawerSession : public QObject {
    Q_OBJECT
public:
    explicit DrawerSession(DrawerView *view);

    void start();

private slots:
    void reload();
    void onModelLoaded(const DrawerModel &model);
    void onItemTapped(const DrawerItem &item);
    void onItemLongPressed(const DrawerItem &item);
    void onBusyChanged(bool isBusy);
    void onScreenClosed();
    void flashDrawer();

private:
    void openScreen(const DrawerItem &item);
    bool isScreenOpen() const;

    DrawerView *view;
    ModelClient *client;
    ActionRunner *runner;
    bool isLoading;
    bool needsReloadAfterScreen;
    QPointer<ScreenSession> screenSession;
};

#endif
