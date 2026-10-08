#ifndef NDDRAWER_DRAWER_LAUNCHER_H
#define NDDRAWER_DRAWER_LAUNCHER_H

#include <QObject>
#include <QPointer>

class DrawerController;
class DrawerView;
class QStackedWidget;

// Opens and closes Prototype C. Two routes (re/notes/G2-findings.md Q2):
//   A (default): push a DrawerController (navSection 6) with
//      MainWindowController::push; back = pop. Nickel owns the lifecycle.
//   B (flag proto-c-pushview): pushView/popView the bare widget, and pop it
//      ourselves when something else becomes the current view.
class DrawerLauncher : public QObject {
    Q_OBJECT
public:
    // Called from the More-tab row. Never throws or crashes on missing API:
    // logs, toasts and returns.
    static void open();

private slots:
    void onBackRequested();
    void onStackCurrentChanged(int index);

private:
    DrawerLauncher();
    static DrawerLauncher *instance();

    bool isOpen() const;
    bool hasRouteApi() const;
    void openWithController(void *window);
    void openWithViewPush(void *window);
    void closeViewPush();
    void discardUnpushedView();

    QPointer<DrawerView> view;
    QPointer<DrawerController> controller;
    QPointer<QStackedWidget> watchedStack;
};

#endif
