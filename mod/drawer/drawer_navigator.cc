#include "drawer_navigator.h"

#include <QStackedWidget>
#include <QVariant>

#include "debug_log.h"

#include "drawer_controller.h"
#include "drawer_session.h"
#include "drawer_view.h"
#include "fallback_strings.h"
#include "mod_switches.h"
#include "navigation.h"
#include "nickel_bridge.h"
#include "status_bar_inset.h"

DrawerNavigator::DrawerNavigator() : QObject(NULL) {}

DrawerNavigator *DrawerNavigator::instance()
{
    // Created on first use, lives as long as Nickel (one small QObject).
    static DrawerNavigator *navigator = new DrawerNavigator();
    return navigator;
}

void DrawerNavigator::open()
{
    DebugLog::write("navigator: open requested (route %s)", ModSwitches::usesViewPushRoute() ? "B" : "A");
    if (isOpen()) {
        DebugLog::write("navigator: already open, ignoring tap");
        return;
    }
    FallbackStrings::reloadLocale();
    if (!hasRouteApi()) {
        DebugLog::write("navigator: libnickel API for route %s missing", ModSwitches::usesViewPushRoute() ? "B" : "A");
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
        recordOpenedDrawer();
        session->start();
    }
}

// The stack starts at More, where the row was tapped: back from the drawer returns there.
void DrawerNavigator::recordOpenedDrawer()
{
    Navigation::clear("drawer opened from More");
    Navigation::ViewState none = Navigation::stateOf(0, 0);
    Navigation::push(Navigation::LEVEL_MORE, QString(), none);
    Navigation::push(Navigation::LEVEL_DRAWER, QString(), none);
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
        DebugLog::write("navigator: route A: unexpected AbstractController offset, not pushing");
        delete controller;
        discardUnpushedView();
        return;
    }
    DebugLog::write("navigator: route A: push controller");
    NickelBridge::pushController(controller->asNickelController());
    DebugLog::write("navigator: route A: push returned, view %p parent %p", static_cast<void *>(view.data()),
                    view ? static_cast<void *>(view->parentWidget()) : NULL);

    if (!view || !view->parentWidget()) {
        DebugLog::write("navigator: route A: push did not add the view to the window stack, cleaning up");
        delete controller;
        discardUnpushedView();
        return;
    }
    if (controller && controller->parent() != view) {
        // Nickel normally parents the controller to the view; without that the
        // controller would outlive the page.
        DebugLog::write("navigator: route A: controller not parented to the view by Nickel, fixing");
        controller->setParent(view);
    }
    StatusBarInset::logGeometry("after push", view, NULL);
    DebugLog::write("navigator: route A: open");
}

void DrawerNavigator::openWithViewPush()
{
    DebugLog::write("navigator: route B: pushView");
    NickelBridge::pushView(view);
    watchedStack = view ? qobject_cast<QStackedWidget *>(view->parentWidget()) : NULL;
    if (!watchedStack) {
        DebugLog::write("navigator: route B: view not in a QStackedWidget after pushView, cleaning up");
        discardUnpushedView();
        return;
    }
    connect(watchedStack, SIGNAL(currentChanged(int)), this, SLOT(onStackCurrentChanged(int)));
    DebugLog::write("navigator: route B: open");
}

void DrawerNavigator::discardUnpushedView()
{
    delete view;
    NickelBridge::showToast(FallbackStrings::text(FallbackStrings::DRAWER_UNAVAILABLE));
}

void DrawerNavigator::onBackRequested()
{
    DebugLog::write("navigator: back");
    Navigation::pop(NULL); // drawer → More
    if (ModSwitches::usesViewPushRoute()) {
        closeViewPush();
        return;
    }
    if (!controller) {
        DebugLog::write("navigator: route A: cannot pop, controller gone");
        return;
    }
    // pop() deletes the view later (deleteLater), and the controller with it.
    DebugLog::write("navigator: route A: pop controller %p", static_cast<void *>(controller.data()));
    NickelBridge::popController(controller->asNickelController());
    DebugLog::write("navigator: route A: pop returned");
}

void DrawerNavigator::onStackCurrentChanged(int index)
{
    if (!view || !watchedStack || watchedStack->currentWidget() == view) {
        return;
    }
    QWidget *current = watchedStack->currentWidget();
    if (current && current->property(NickelBridge::OVERLAY_PROPERTY).toBool()) {
        DebugLog::write("navigator: route B: drawer overlay %d is current, drawer stays", index);
        return;
    }
    DebugLog::write("navigator: route B: view %d became current, closing the drawer", index);
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
    DebugLog::write("navigator: drawer closed");
    // After back only More is left; otherwise Nickel closed the page (tab tap, Home)
    // and with it everything stacked on it.
    const Navigation::Entry *top = Navigation::top();
    bool isClosedByNickel = top && top->level != Navigation::LEVEL_MORE;
    Navigation::clear(isClosedByNickel ? "drawer page closed by Nickel (tab or Home)" : "back at More");
    emit drawerClosed();
}
