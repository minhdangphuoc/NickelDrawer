#include "toast_gate.h"

#include <QElapsedTimer>

#include "debug_log.h"
#include "nickel_bridge.h"
#include "toast_limiter.h"

static ToastLimiter::State limiterState;
static QElapsedTimer toastClock;
static bool hasStarted = false;

static long nowMilliseconds()
{
    if (!hasStarted) {
        ToastLimiter::reset(&limiterState);
        toastClock.start();
        hasStarted = true;
    }
    return static_cast<long>(toastClock.elapsed());
}

namespace ToastGate {

void show(const QString &appKey, const QString &text)
{
    QByteArray key = appKey.toUtf8();
    if (!ToastLimiter::allow(&limiterState, key.constData(), nowMilliseconds())) {
        DebugLog::write("toast: dropped for %s (one per 2 s): %s", key.constData(), qPrintable(text));
        return;
    }
    NickelBridge::showToast(text);
}

void show(const char *hostKey, const QString &text)
{
    show(QString::fromLatin1(hostKey), text);
}

} // namespace ToastGate
