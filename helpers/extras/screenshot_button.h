#ifndef EXTRAS_SCREENSHOT_BUTTON_H
#define EXTRAS_SCREENSHOT_BUTTON_H

#include <QObject>
#include <QPointer>

#include "icon_pixmaps.h"

class QBoxLayout;
class QEvent;
class QWidget;

// The screenshot button of one Nickel status bar: a StatusBarIcon placed just before the brightness
// icon, and the tap that takes the screenshot. It is a child of the status bar, so Nickel deleting
// the bar (it rebuilds it after setup, language change and main-window reset) deletes it too.
//
// The button looks and sits like Nickel's own icons: it copies the brightness icon's layout alignment,
// size policy, minimum/maximum size, margins and label alignment, its images are the exact size of
// the brightness icon's pixmap, and nothing is fixed or computed by hand (see icon_metrics.h for why
// a fixed size sat too high). It is on or off by the flag file (helper_flags.h), checked when the bar is built, whenever
// the bar is shown, on every tap and when the Tweaks panel changed it (refreshAll). If the layout
// is not the expected one, nothing is added.
class ScreenshotButton : public QObject {
    Q_OBJECT
public:
    // Adds a controller (and, when enabled, the button) to a status bar that was just built.
    static void attachTo(QWidget *statusBar);
    // Re-reads the flag file for every live status bar, so the Tweaks panel's switch applies at once.
    static void refreshAll();

protected:
    bool eventFilter(QObject *watched, QEvent *event);

private slots:
    void onTapped();
    void finishCapture();
    void logGeometry();

private:
    explicit ScreenshotButton(QWidget *statusBar);

    void updateFromFlagFile();
    void showButton(bool isShown);
    void addButton();
    void removeButton();
    void matchNickelIcon(QWidget *icon, QWidget *light) const;
    QWidget *referenceIcon(QBoxLayout *layout) const;
    void scheduleGeometryLog();
    void describeIcon(const char *role, QWidget *icon, QBoxLayout *layout) const;
    QWidget *lightIcon() const;
    QBoxLayout *iconLayout() const;
    void blankButton();
    void restoreButton();
    void reportCapture(bool isSaved, const QString &fileName);

    QWidget *statusBar;
    QPointer<QWidget> button;
    IconPixmaps::Set pixmaps;
    bool isWanted;
    bool isCapturing;
    bool isGeometryLogPending;
    QString nameBeforeCapture;
};

#endif
