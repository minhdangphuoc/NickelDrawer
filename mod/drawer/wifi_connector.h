#ifndef NICKELDRAWER_WIFI_CONNECTOR_H
#define NICKELDRAWER_WIFI_CONNECTOR_H

#include <QObject>

class QTimer;

// Starts Nickel's Wi-Fi connect flow and reports when the internet is
// reachable, polling once a second for up to 30 s (the "connected" signal is
// unreliable, as NickelHardcover notes). Emits one signal, then deletes itself.
class WifiConnector : public QObject {
    Q_OBJECT
public:
    explicit WifiConnector(QObject *parent);
    void start();

signals:
    void connected();
    void failed();

private slots:
    void poll();

private:
    void finish(bool isConnected);

    QTimer *pollTimer;
    int attempts;
};

#endif
