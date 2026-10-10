#ifndef NICKELDRAWER_HOST_REQUEST_RUNNER_H
#define NICKELDRAWER_HOST_REQUEST_RUNNER_H

#include <QList>
#include <QObject>
#include <QString>

#include "drawer_model.h"
#include "host_requests.h"

// Performs what a reply asked for (docs/spec/drawer-protocol.md "Requests"):
// Wi-Fi first (the same ask-then-connect flow as needs_wifi, skipped when the
// internet is already reachable), then one library rescan. A plan arriving
// while the Wi-Fi flow is open joins it. Nothing is remembered afterwards.
class HostRequestRunner : public QObject {
    Q_OBJECT
public:
    explicit HostRequestRunner(QObject *parent);

    // requests: a reply's `requests`; manifestRescan: its `rescan` field.
    // Unknown entries are logged and dropped. title: heading of the Wi-Fi dialogs.
    static HostRequests::Plan planFor(const QList<HostRequest> &requests, const QString &manifestRescan);

    void perform(const HostRequests::Plan &plan, const QString &title);

private slots:
    void onWifiAnswer(bool isAccepted);
    void onWifiConnected();
    void onWifiFailed();

private:
    void askForWifi();
    void rescan();

    HostRequests::Plan pending;
    QString dialogTitle;
    bool isWaitingForWifi;
};

#endif
