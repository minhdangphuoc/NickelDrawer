#include "wake_rotation.h"

#include <QCoreApplication>
#include <QDateTime>
#include <QFile>
#include <QProcess>
#include <QStringList>
#include <QTimer>

#include <NickelHook.h>

#include "debug_log.h"
#include "nickel_bridge.h"

static const char *const WALLPAPER_APP_DIRECTORY = "/mnt/onboard/.adds/apps/wallpaper";
static const char *const WALLPAPER_BINARY = "/mnt/onboard/.adds/apps/wallpaper/bin/wallpaper";
// Exists while the app's rotation is random (internal/wallpaper syncWakeFlag).
static const char *const ROTATE_ON_WAKE_FLAG = "/mnt/onboard/.adds/apps/wallpaper/data/rotate-on-wake";
// No child keeps a working directory on /mnt/onboard (it would block USB export).
static const char *const WORKING_DIRECTORY = "/";
// Nickel's startup (Nickel3Application::finishInitialization connects to the
// same power manager) is long done by then.
static const int INSTALL_DELAY_MILLISECONDS = 30000;
// resumedFromSleep() fires once per wake; this only guards against a double
// emission. Wall-clock time, so the sleep itself counts.
static const qint64 DEBOUNCE_MILLISECONDS = 10000;

static WakeRotation *sharedRotation = NULL;

WakeRotation::WakeRotation() : QObject(NULL), isInstalled(false), lastStartMilliseconds(0)
{
}

void WakeRotation::scheduleInstall()
{
    QCoreApplication *application = QCoreApplication::instance();
    if (!application) {
        nh_log("wake rotation: no application object at init, connecting when the More tab is built");
        return;
    }
    if (!sharedRotation) {
        sharedRotation = new WakeRotation();
    }
    // init may run on any thread; the object and its timer belong on the GUI thread.
    sharedRotation->moveToThread(application->thread());
    QMetaObject::invokeMethod(sharedRotation, "startInstallDelay", Qt::QueuedConnection);
}

void WakeRotation::install()
{
    if (!sharedRotation) {
        sharedRotation = new WakeRotation();
    }
    sharedRotation->installNow();
}

void WakeRotation::startInstallDelay()
{
    QTimer::singleShot(INSTALL_DELAY_MILLISECONDS, this, SLOT(installNow()));
}

void WakeRotation::installNow()
{
    if (isInstalled) {
        return;
    }
    isInstalled = true; // one attempt per boot: a missing signal stays missing
    QObject *manager = NickelBridge::powerWorkflowManager();
    if (!manager) {
        return;
    }
    if (!connect(manager, SIGNAL(resumedFromSleep()), this, SLOT(onResumedFromSleep()))) {
        nh_log("wake rotation: N3PowerWorkflowManager::resumedFromSleep() not found, random wallpaper stays put");
        return;
    }
    nh_log("wake rotation: watching resumedFromSleep()");
}

void WakeRotation::onResumedFromSleep()
{
    // A stat only: nothing is opened on /mnt/onboard.
    if (!QFile::exists(QString::fromLatin1(ROTATE_ON_WAKE_FLAG))) {
        return;
    }
    if (isDebounced()) {
        DebugLog::write("wake rotation: second wake signal within %lld ms, skipped",
                        static_cast<long long>(DEBOUNCE_MILLISECONDS));
        return;
    }
    startRotation();
}

bool WakeRotation::isDebounced() const
{
    qint64 elapsed = QDateTime::currentMSecsSinceEpoch() - lastStartMilliseconds;
    return lastStartMilliseconds != 0 && elapsed >= 0 && elapsed < DEBOUNCE_MILLISECONDS;
}

// Detached and never waited for: the GUI thread only forks.
void WakeRotation::startRotation()
{
    QString binary = QString::fromLatin1(WALLPAPER_BINARY);
    if (!QFile::exists(binary)) {
        nh_log("wake rotation: %s missing, reinstall the Wallpaper app", WALLPAPER_BINARY);
        return;
    }
    QStringList arguments;
    arguments << QString::fromLatin1("next") << QString::fromLatin1("--app-dir")
              << QString::fromLatin1(WALLPAPER_APP_DIRECTORY);
    lastStartMilliseconds = QDateTime::currentMSecsSinceEpoch();
    if (!QProcess::startDetached(binary, arguments, QString::fromLatin1(WORKING_DIRECTORY))) {
        nh_log("wake rotation: cannot start %s next", WALLPAPER_BINARY);
        return;
    }
    nh_log("wake rotation: started wallpaper next");
}
