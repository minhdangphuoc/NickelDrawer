#include "drawer_session.h"

#include <QTimer>

#include <NickelHook.h>

#include "action_runner.h"
#include "drawer_view.h"
#include "mod_switches.h"
#include "model_client.h"
#include "navigation.h"
#include "nickel_bridge.h"
#include "screen_session.h"

static const int RUN_FROM_EVENT_LOOP = 0;

DrawerSession::DrawerSession(DrawerView *drawerView)
    : QObject(drawerView), view(drawerView), client(new ModelClient(this)), runner(new ActionRunner(client, this)),
      isLoading(false), needsReloadAfterScreen(false)
{
    connect(client, SIGNAL(modelLoaded(DrawerModel)), this, SLOT(onModelLoaded(DrawerModel)));
    connect(view, SIGNAL(itemTapped(DrawerItem)), this, SLOT(onItemTapped(DrawerItem)));
    connect(view, SIGNAL(itemLongPressed(DrawerItem, QPoint)), this, SLOT(onItemLongPressed(DrawerItem, QPoint)));
    connect(runner, SIGNAL(busyChanged(bool)), this, SLOT(onBusyChanged(bool)));
    connect(runner, SIGNAL(modelMayHaveChanged()), this, SLOT(reload()));
}

void DrawerSession::start()
{
    reload();
}

void DrawerSession::reload()
{
    if (isScreenOpen()) {
        needsReloadAfterScreen = true; // no repaint behind the app screen
        return;
    }
    if (isLoading) {
        return;
    }
    isLoading = true;
    client->requestModel();
}

void DrawerSession::onModelLoaded(const DrawerModel &model)
{
    isLoading = false;
    view->setModel(model);
}

void DrawerSession::onItemTapped(const DrawerItem &item)
{
    if (item.id == QLatin1String(RETRY_ROW_ID)) {
        reload();
        return;
    }
    if (item.hasScreen && ModSwitches::usesScreens()) {
        openScreen(item);
        return;
    }
    runner->runPrimary(item);
}

bool DrawerSession::isScreenOpen() const
{
    return !screenSession.isNull();
}

void DrawerSession::openScreen(const DrawerItem &item)
{
    if (isScreenOpen() || runner->isRunning()) {
        return;
    }
    // Records the drawer's view mode and page; back from the screen restores them.
    Navigation::push(Navigation::LEVEL_APP_SCREEN, item.id,
                     Navigation::stateOf(view->currentViewMode(), view->currentPageIndex()));
    screenSession = new ScreenSession(item, view, runner);
    connect(screenSession, SIGNAL(closed()), this, SLOT(onScreenClosed()));
    screenSession->start();
}

void DrawerSession::onScreenClosed()
{
    screenSession = NULL;
    restoreDrawerState();
    // One full refresh when the drawer page shows again (ui-kit §3: push or pop flashes).
    QTimer::singleShot(RUN_FROM_EVENT_LOOP, this, SLOT(flashDrawer()));
    if (needsReloadAfterScreen) {
        needsReloadAfterScreen = false;
        reload();
    }
}

// The screen and any sub-screens it pushed are gone: the drawer as it was.
void DrawerSession::restoreDrawerState()
{
    Navigation::Entry drawer;
    if (Navigation::popTo(Navigation::LEVEL_DRAWER, &drawer)) {
        view->restoreState(static_cast<DrawerLogic::ViewMode>(drawer.state.viewMode), drawer.state.page);
    }
}

void DrawerSession::flashDrawer()
{
    NickelBridge::requestFullRefresh(view);
}

void DrawerSession::onItemLongPressed(const DrawerItem &item, const QPoint &anchor)
{
    runner->openMenu(item, anchor);
}

void DrawerSession::onBusyChanged(bool isBusy)
{
    view->setInputBlocked(isBusy);
}
