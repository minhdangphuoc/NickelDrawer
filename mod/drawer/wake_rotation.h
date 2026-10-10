#ifndef NICKELDRAWER_WAKE_ROTATION_H
#define NICKELDRAWER_WAKE_ROTATION_H

#include <QObject>

// Random wallpaper rotation (docs/spec/wallpaper.md "Random rotation"): after
// each wake from sleep, when the Wallpaper app left its rotate-on-wake flag,
// starts `wallpaper next` detached so the next sleep shows another picture.
// We choose the picture, not Nickel (its qrand() pick repeated pictures).
//
// Rotating after the wake, not before sleep, never races Nickel reading
// .kobo/screensaver while it builds the sleep screen. Fails soft: without the
// power manager or its resumedFromSleep() signal it only logs, and the
// wallpaper simply stays the current picture.
class WakeRotation : public QObject {
    Q_OBJECT
public:
    // From init(): tiny. Queues the connection to the GUI thread's event loop
    // and makes it a while after Nickel started, so init never touches Nickel.
    static void scheduleInstall();
    // Connects once (later calls do nothing). Also called when the More tab
    // is built, in case init could not queue it. GUI thread only.
    static void install();

private slots:
    void startInstallDelay();
    void installNow();
    void onResumedFromSleep();

private:
    WakeRotation();
    bool isDebounced() const;
    void startRotation();

    bool isInstalled;
    qint64 lastStartMilliseconds;
};

#endif
