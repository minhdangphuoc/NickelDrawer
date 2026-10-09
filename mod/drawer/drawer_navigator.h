#ifndef NICKELDRAWER_DRAWER_NAVIGATOR_H
#define NICKELDRAWER_DRAWER_NAVIGATOR_H

#include <QObject>
#include <QPointer>

class DrawerController;
class DrawerView;
class QStackedWidget;

// Opens and closes the drawer page (re/notes/G2-findings.md Q2):
//   route A (default): push a DrawerController (navSection 6) with
//      MainWindowController::push; back = pop. Nickel owns the lifecycle
//      (the controller becomes a child of the view).
//   route B (flag .adds/nickeldrawer/pushview): pushView/popView the bare
//      widget, and pop it ourselves when another view becomes current.
class DrawerNavigator : public QObject {
    Q_OBJECT
public:
    static DrawerNavigator *instance();

    // From the More-tab row. Never crashes on missing API: logs, toasts, returns.
    void open();

signals:
    // The drawer page went away (back, tab switch, Home); the More row
    // refreshes its status line.
    void drawerClosed();

private slots:
    void onBackRequested();
    void onStackCurrentChanged(int index);
    void onViewDestroyed();

private:
    DrawerNavigator();

    bool isOpen() const;
    bool hasRouteApi() const;
    void openWithController();
    void openWithViewPush();
    void closeViewPush();
    void discardUnpushedView();

    QPointer<DrawerView> view;
    QPointer<DrawerController> controller;
    QPointer<QStackedWidget> watchedStack;
};

#endif
