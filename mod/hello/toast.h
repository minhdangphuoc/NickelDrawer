#ifndef NDHELLO_TOAST_H
#define NDHELLO_TOAST_H

#include <QObject>

// ToastOnce shows one toast through Nickel when show() is called. It exists
// only because Qt 5.2 has no QTimer::singleShot(msec, functor): the timer needs
// a QObject receiver with a slot. It runs on the thread that created it (the
// Nickel GUI thread when created from init()).
class ToastOnce : public QObject {
    Q_OBJECT
public:
    // Looks up the toast symbols; isAvailable() is false when they are missing.
    static void scheduleAfterMilliseconds(int milliseconds);

public slots:
    void show();
};

#endif
