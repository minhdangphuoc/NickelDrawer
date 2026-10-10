#include "host_request_runner.h"

#include <NickelHook.h>

#include "dialogs.h"
#include "fallback_strings.h"
#include "nickel_bridge.h"
#include "wifi_connector.h"

HostRequests::Plan HostRequestRunner::planFor(const QList<HostRequest> &requests, const QString &manifestRescan)
{
    HostRequests::Plan plan;
    if (HostRequests::addManifestRescan(&plan, qPrintable(manifestRescan)) != HostRequests::REQUEST_ADDED) {
        nh_log("requests: unknown rescan mode %s dropped", qPrintable(manifestRescan));
    }
    for (int i = 0; i < requests.size(); i++) {
        HostRequests::Verdict verdict =
            HostRequests::add(&plan, qPrintable(requests[i].request), qPrintable(requests[i].mode));
        if (verdict != HostRequests::REQUEST_ADDED) {
            nh_log("requests: %s (mode %s) dropped", qPrintable(requests[i].request), qPrintable(requests[i].mode));
        }
    }
    return plan;
}

HostRequestRunner::HostRequestRunner(QObject *parent) : QObject(parent), isWaitingForWifi(false) {}

void HostRequestRunner::perform(const HostRequests::Plan &plan, const QString &title)
{
    if (HostRequests::isEmpty(plan)) {
        return;
    }
    HostRequests::merge(&pending, plan);
    if (isWaitingForWifi) {
        return; // joins the flow that is already open
    }
    dialogTitle = title;
    if (pending.needsWifi && !NickelBridge::isInternetAccessible()) {
        isWaitingForWifi = true;
        askForWifi();
        return;
    }
    rescan();
}

void HostRequestRunner::askForWifi()
{
    if (!NickelBridge::hasWifi()) {
        nh_log("requests: Wi-Fi symbols missing, skipping the connect");
        onWifiFailed();
        return;
    }
    ConfirmQuestion *question =
        new ConfirmQuestion(dialogTitle, FallbackStrings::text(FallbackStrings::WIFI_OFF_TEXT),
                            FallbackStrings::text(FallbackStrings::WIFI_TURN_ON), FallbackStrings::text(FallbackStrings::CANCEL),
                            ConfirmQuestion::NOT_DESTRUCTIVE);
    connect(question, SIGNAL(answered(bool)), this, SLOT(onWifiAnswer(bool)));
    question->ask();
}

void HostRequestRunner::onWifiAnswer(bool isAccepted)
{
    if (!isAccepted) {
        isWaitingForWifi = false;
        rescan(); // a rescan is local: it does not need the network
        return;
    }
    WifiConnector *connector = new WifiConnector(this);
    connect(connector, SIGNAL(connected()), this, SLOT(onWifiConnected()));
    connect(connector, SIGNAL(failed()), this, SLOT(onWifiFailed()));
    connector->start();
}

void HostRequestRunner::onWifiConnected()
{
    isWaitingForWifi = false;
    rescan();
}

void HostRequestRunner::onWifiFailed()
{
    isWaitingForWifi = false;
    Dialogs::showMessage(dialogTitle, FallbackStrings::text(FallbackStrings::WIFI_FAILED));
    rescan();
}

// Takes the plan: the next reply starts from nothing.
void HostRequestRunner::rescan()
{
    HostRequests::RescanChoice choice = HostRequests::rescanToRun(pending);
    pending = HostRequests::Plan();
    if (choice == HostRequests::RESCAN_FULL && !NickelBridge::hasRescan(NickelBridge::RESCAN_FULL)) {
        nh_log("requests: full rescan unavailable, using a quick one");
        choice = HostRequests::RESCAN_QUICK;
    }
    if (choice == HostRequests::RESCAN_FULL) {
        NickelBridge::rescanLibrary(NickelBridge::RESCAN_FULL);
    } else if (choice == HostRequests::RESCAN_QUICK) {
        NickelBridge::rescanLibrary(NickelBridge::RESCAN_QUICK);
    }
}
