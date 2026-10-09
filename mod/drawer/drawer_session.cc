#include "drawer_session.h"

#include "action_runner.h"
#include "drawer_view.h"
#include "model_client.h"

DrawerSession::DrawerSession(DrawerView *drawerView)
    : QObject(drawerView), view(drawerView), client(new ModelClient(this)), runner(new ActionRunner(client, this)),
      isLoading(false)
{
    connect(client, SIGNAL(modelLoaded(DrawerModel)), this, SLOT(onModelLoaded(DrawerModel)));
    connect(view, SIGNAL(itemTapped(DrawerItem)), this, SLOT(onItemTapped(DrawerItem)));
    connect(view, SIGNAL(itemLongPressed(DrawerItem)), this, SLOT(onItemLongPressed(DrawerItem)));
    connect(runner, SIGNAL(busyChanged(bool)), this, SLOT(onBusyChanged(bool)));
    connect(runner, SIGNAL(modelMayHaveChanged()), this, SLOT(reload()));
}

void DrawerSession::start()
{
    reload();
}

void DrawerSession::reload()
{
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
    runner->runPrimary(item);
}

void DrawerSession::onItemLongPressed(const DrawerItem &item)
{
    runner->openMenu(item);
}

void DrawerSession::onBusyChanged(bool isBusy)
{
    view->setInputBlocked(isBusy);
}
