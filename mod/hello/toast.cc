#include "toast.h"

#include <QString>
#include <QTimer>

#include <NickelHook.h>

// Filled by NickelHook before init(); NULL when libnickel lacks the symbol
// (both are optional so a missing one never triggers the failsafe).
extern "C" {
void *(*MainWindowController_sharedInstance)();
void (*MainWindowController_toast)(void *window, QString const &primary, QString const &secondary, int milliseconds);
}

static const char *const TOAST_TEXT = "NickelDrawer test mod loaded";
static const int TOAST_DURATION_MILLISECONDS = 3000;

static bool hasToastSymbols()
{
    return MainWindowController_sharedInstance && MainWindowController_toast;
}

void ToastOnce::scheduleAfterMilliseconds(int milliseconds)
{
    if (!hasToastSymbols()) {
        nh_log("toast symbols not found in libnickel, skipping toast");
        return;
    }
    // Intentionally never freed: it lives as long as Nickel and is tiny.
    ToastOnce *receiver = new ToastOnce();
    QTimer::singleShot(milliseconds, receiver, SLOT(show()));
}

void ToastOnce::show()
{
    void *window = MainWindowController_sharedInstance();
    if (!window) {
        nh_log("MainWindowController::sharedInstance returned null, skipping toast");
    } else {
        MainWindowController_toast(window, QString::fromUtf8(TOAST_TEXT), QStringLiteral(""), TOAST_DURATION_MILLISECONDS);
        nh_log("toast shown");
    }
    deleteLater();
}
