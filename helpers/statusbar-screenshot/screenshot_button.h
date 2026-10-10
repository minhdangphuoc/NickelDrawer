#ifndef STATUSBARSHOT_SCREENSHOT_BUTTON_H
#define STATUSBARSHOT_SCREENSHOT_BUTTON_H

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
// The button is on or off by the flag file (helper_flags.h), checked when the bar is built, whenever
// the bar is shown and on every tap, and by the dynamic property "nhScreenshotButton" that NickelDrawer's
// Tweaks sets on the bar for an immediate update. If the layout is not the expected one, nothing is added.
class ScreenshotButton : public QObject {
    Q_OBJECT
public:
    // The name of the property NickelDrawer sets (docs/spec/tweaks.md, "statusbar-screenshot").
    static const char SWITCH_PROPERTY[];

    // Adds a controller (and, when enabled, the button) to a status bar that was just built.
    static void attachTo(QWidget *statusBar);

protected:
    bool eventFilter(QObject *watched, QEvent *event);

private slots:
    void onTapped();
    void finishCapture();

private:
    explicit ScreenshotButton(QWidget *statusBar);

    void updateFromFlagFile();
    void showButton(bool isShown);
    void addButton();
    void removeButton();
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
    QString nameBeforeCapture;
};

#endif
