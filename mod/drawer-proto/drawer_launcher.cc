#include "drawer_launcher.h"

#include <QStackedWidget>

#include <NickelHook.h>

#include "device_state.h"
#include "drawer_controller.h"
#include "drawer_view.h"
#include "nickel_api.h"

static const bool PUSH_WITHOUT_ANIMATION = false;

DrawerLauncher::DrawerLauncher() : QObject(NULL) {}

DrawerLauncher *DrawerLauncher::instance()
{
    // Created on first tap, lives as long as Nickel (one small QObject).
    static DrawerLauncher *launcher = new DrawerLauncher();
    return launcher;
}

void DrawerLauncher::open()
{
    DrawerLauncher *launcher = instance();
    if (launcher->isOpen()) {
        nh_log("drawer: already open, ignoring tap");
        return;
    }
    if (!launcher->hasRouteApi()) {
        nh_log("drawer: libnickel API for route %s missing, prototype C disabled",
               prototypeSwitches.usesViewPushRoute ? "B (pushView)" : "A (controller)");
        NickelApi::showToast(QStringLiteral("NickelDrawer: drawer unavailable (see log)"));
        return;
    }
    void *window = NickelApi::mainWindow();
    if (!window) {
        return;
    }

    launcher->view = new DrawerView(DeviceState::appsLabel());
    connect(launcher->view, SIGNAL(backRequested()), launcher, SLOT(onBackRequested()));
    if (prototypeSwitches.usesViewPushRoute) {
        launcher->openWithViewPush(window);
    } else {
        launcher->openWithController(window);
    }
}

bool DrawerLauncher::isOpen() const
{
    return !view.isNull();
}

bool DrawerLauncher::hasRouteApi() const
{
    return prototypeSwitches.usesViewPushRoute ? NickelApi::hasViewPush() : NickelApi::hasControllerPush();
}

void DrawerLauncher::openWithController(void *window)
{
    controller = new DrawerController(view);
    if (!controller->hasNickelLayout()) {
        nh_log("drawer: route A: unexpected AbstractController offset, not pushing");
        delete controller;
        discardUnpushedView();
        return;
    }

    nh_log("drawer: route A: push controller");
    MainWindowController_push(window, controller->asNickelController(), PUSH_WITHOUT_ANIMATION);

    if (!view || !view->parentWidget()) {
        nh_log("drawer: route A: push did not add the view to the window stack, cleaning up");
        delete controller;
        discardUnpushedView();
        return;
    }
    if (controller && controller->parent() != view) {
        // Nickel normally parents the controller to the view; without that the
        // controller would outlive the page. Do it ourselves and say so.
        nh_log("drawer: route A: controller not parented to the view by Nickel, fixing");
        controller->setParent(view);
    }
    nh_log("drawer: route A: open");
}

void DrawerLauncher::openWithViewPush(void *window)
{
    nh_log("drawer: route B: pushView");
    MainWindowController_pushView(window, view);

    watchedStack = view ? qobject_cast<QStackedWidget *>(view->parentWidget()) : NULL;
    if (!watchedStack) {
        nh_log("drawer: route B: view not in a QStackedWidget after pushView, cleaning up");
        discardUnpushedView();
        return;
    }
    connect(watchedStack, SIGNAL(currentChanged(int)), this, SLOT(onStackCurrentChanged(int)));
    nh_log("drawer: route B: open");
}

void DrawerLauncher::discardUnpushedView()
{
    delete view;
    NickelApi::showToast(QStringLiteral("NickelDrawer: could not open drawer (see log)"));
}

void DrawerLauncher::onBackRequested()
{
    nh_log("drawer: back tapped");
    if (prototypeSwitches.usesViewPushRoute) {
        closeViewPush();
        return;
    }
    void *window = NickelApi::mainWindow();
    if (!window || !controller) {
        nh_log("drawer: route A: cannot pop (window or controller gone)");
        return;
    }
    // pop() deletes the view later (deleteLater), and the controller with it.
    MainWindowController_pop(window, controller->asNickelController());
}

void DrawerLauncher::onStackCurrentChanged(int index)
{
    if (!view || !watchedStack || watchedStack->currentWidget() == view) {
        return;
    }
    nh_log("drawer: route B: another view became current (index %d), closing drawer", index);
    closeViewPush();
}

void DrawerLauncher::closeViewPush()
{
    if (watchedStack) {
        disconnect(watchedStack, SIGNAL(currentChanged(int)), this, SLOT(onStackCurrentChanged(int)));
        watchedStack = NULL;
    }
    void *window = NickelApi::mainWindow();
    if (!window || !view) {
        nh_log("drawer: route B: cannot popView (window or view gone)");
        return;
    }
    MainWindowController_popView(window, view);
}
