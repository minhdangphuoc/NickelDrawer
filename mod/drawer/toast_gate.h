#ifndef NICKELDRAWER_TOAST_GATE_H
#define NICKELDRAWER_TOAST_GATE_H

#include <QString>

// Every drawer toast goes through here: Nickel's toast, at most one per app
// every 2 s (ToastLimiter, ui-kit §3.7). A dropped toast is logged.
namespace ToastGate {

// Keys of toasts that belong to no app.
static const char HOST_KEY[] = "_drawer";
static const char TWEAKS_KEY[] = "_tweaks";

void show(const QString &appKey, const QString &text);
// Host toasts (HOST_KEY, TWEAKS_KEY).
void show(const char *hostKey, const QString &text);

} // namespace ToastGate

#endif
