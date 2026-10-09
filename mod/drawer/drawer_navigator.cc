#include "drawer_navigator.h"

#include <QStackedWidget>
#include <QVariant>

#include <NickelHook.h>

#include "drawer_controller.h"
#include "drawer_session.h"
#include "drawer_view.h"
#include "fallback_strings.h"
#include "mod_switches.h"
#include "nickel_bridge.h"

DrawerNavigator::DrawerNavigator() : QObject(NULL) {}

DrawerNavigator *DrawerNavigator::instance()
{
    // Created on first use, lives as long as Nickel (one small QObject).
    static DrawerNavigator *navigator = new DrawerNavigator();
    return navigator;
}

void DrawerNavigator::open()
{
    if (isOpen()) {
        nh_log("navigator: already open, ignoring tap");
        return;
    }
    FallbackStrings::reloadLocale();
    if (!hasRouteApi()) {
        nh_log("navigator: libnickel API for route %s missing", ModSwitches::usesViewPushRoute() ? "B" : "A");
        NickelBridge::showToast(FallbackStrings::text(FallbackStrings::DRAWER_UNAVAILABLE));
        return;
    }

    view = new DrawerView();
    connect(view, SIGNAL(backRequested()), this, SLOT(onBackRequested()));
    connect(view, SIGNAL(destroyed()), this, SLOT(onViewDestroyed()));
    DrawerSession *session = new DrawerSession(view);
    if (ModSwitches::usesViewPushRoute()) {
        openWithViewPush();
    } else {
        openWithController();
    }
    if (view) {
        session->start();
    }
}

bool DrawerNavigator::isOpen() const
{
    return !view.isNull();
}

bool DrawerNavigator::hasRouteApi() const
{
    return ModSwitches::usesViewPushRoute() ? NickelBridge::hasViewPush() : NickelBridge::hasControllerPush();
}

void DrawerNavigator::openWithController()
{
    controller = new DrawerController(view);
    if (!controller->hasNickelLayout()) {
        nh_log("navigator: route A: unexpected AbstractController offset, not pushing");
        delete controller;
        discardUnpushedView();
        return;
    }
    nh_log("navigator: route A: push controller");
    NickelBridge::pushController(controller->asNickelController());

    if (!view || !view->parentWidget()) {
        nh_log("navigator: route A: push did not add the view to the window stack, cleaning up");
        delete controller;
        discardUnpushedView();
        return;
    }
    if (controller && controller->parent() != view) {
        // Nickel normally parents the controller to the view; without that the
        // controller would outlive the page.
        nh_log("navigator: route A: controller not parented to the view by Nickel, fixing");
        controller->setParent(view);
    }
    nh_log("navigator: route A: open");
}

void DrawerNavigator::openWithViewPush()
{
    nh_log("navigator: route B: pushView");
    NickelBridge::pushView(view);
    watchedStack = view ? qobject_cast<QStackedWidget *>(view->parentWidget()) : NULL;
    if (!watchedStack) {
        nh_log("navigator: route B: view not in a QStackedWidget after pushView, cleaning up");
        discardUnpushedView();
        return;
    }
    connect(watchedStack, SIGNAL(currentChanged(int)), this, SLOT(onStackCurrentChanged(int)));
    nh_log("navigator: route B: open");
}

void DrawerNavigator::discardUnpushedView()
{
    delete view;
    NickelBridge::showToast(FallbackStrings::text(FallbackStrings::DRAWER_UNAVAILABLE));
}

void DrawerNavigator::onBackRequested()
{
    nh_log("navigator: back");
    if (ModSwitches::usesViewPushRoute()) {
        closeViewPush();
        return;
    }
    if (!controller) {
        nh_log("navigator: route A: cannot pop, controller gone");
        return;
    }
    // pop() deletes the view later (deleteLater), and the controller with it.
    NickelBridge::popController(controller->asNickelController());
}

void DrawerNavigator::onStackCurrentChanged(int index)
{
    if (!view || !watchedStack || watchedStack->currentWidget() == view) {
        return;
    }
    QWidget *current = watchedStack->currentWidget();
    if (current && current->property(NickelBridge::OVERLAY_PROPERTY).toBool()) {
        nh_log("navigator: route B: drawer overlay %d is current, drawer stays", index);
        return;
    }
    nh_log("navigator: route B: view %d became current, closing the drawer", index);
    closeViewPush();
}

void DrawerNavigator::closeViewPush()
{
    if (watchedStack) {
        disconnect(watchedStack, SIGNAL(currentChanged(int)), this, SLOT(onStackCurrentChanged(int)));
        watchedStack = NULL;
    }
    if (view) {
        NickelBridge::popView(view);
    }
}

void DrawerNavigator::onViewDestroyed()
{
    nh_log("navigator: drawer closed");
    emit drawerClosed();
}
