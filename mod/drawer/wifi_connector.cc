#include "wifi_connector.h"

#include <QTimer>

#include <NickelHook.h>

#include "drawer_logic.h"
#include "nickel_bridge.h"

WifiConnector::WifiConnector(QObject *parent) : QObject(parent), pollTimer(new QTimer(this)), attempts(0)
{
    pollTimer->setInterval(DrawerLogic::WIFI_POLL_INTERVAL_MILLISECONDS);
    connect(pollTimer, SIGNAL(timeout()), this, SLOT(poll()));
}

void WifiConnector::start()
{
    if (NickelBridge::isInternetAccessible()) {
        // Deliver from the event loop so the caller's connections are in place.
        QTimer::singleShot(0, this, SLOT(poll()));
        return;
    }
    if (!NickelBridge::requestWifiConnect()) {
        nh_log("wifi: connect unavailable");
        attempts = DrawerLogic::WIFI_POLL_ATTEMPTS; // the next poll reports failure
        QTimer::singleShot(0, this, SLOT(poll()));
        return;
    }
    pollTimer->start();
}

void WifiConnector::poll()
{
    if (NickelBridge::isInternetAccessible()) {
        finish(true);
        return;
    }
    attempts++;
    if (attempts >= DrawerLogic::WIFI_POLL_ATTEMPTS) {
        nh_log("wifi: no connection after %d checks", attempts);
        finish(false);
    }
}

void WifiConnector::finish(bool isConnected)
{
    pollTimer->stop();
    if (isConnected) {
        emit connected();
    } else {
        emit failed();
    }
    deleteLater();
}
