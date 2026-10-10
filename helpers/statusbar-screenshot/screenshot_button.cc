#include "screenshot_button.h"

#include <QBoxLayout>
#include <QEvent>
#include <QFileInfo>
#include <QTimer>
#include <QWidget>

#include <NickelHook.h>

#include "capture_check.h"
#include "helper_flags.h"
#include "helper_log.h"
#include "helper_strings.h"
#include "nickel_bridge.h"

static const char BUTTON_OBJECT_NAME[] = "nhScreenshotIcon";
static const char ICON_CONTAINER_NAME[] = "iconContainer";
static const char LIGHT_ICON_NAME[] = "lightIcon";
static const char LIGHT_ICON_CLASS[] = "StatusBarLightIcon";
static const int FALLBACK_ICON_SIZE = 64;
// Time for Qt to repaint the bar without the button and compose it into the screen image that
// saveCurrentScreen() copies. hide() or update() alone only schedule a paint, so the capture waits a
// few event-loop passes. Tune on the device (re/notes/statusbar-screenshot.md, section 3).
static const int CAPTURE_DELAY_MILLISECONDS = 250;

ScreenshotButton::ScreenshotButton(QWidget *bar)
    : QObject(bar), statusBar(bar), isWanted(true), isCapturing(false)
{
}

void ScreenshotButton::attachTo(QWidget *statusBar)
{
    if (statusBar->findChild<ScreenshotButton *>()) {
        return; // a second construction hook for the same bar
    }
    ScreenshotButton *controller = new ScreenshotButton(statusBar);
    statusBar->installEventFilter(controller);
    controller->updateFromFlagFile();
}

// ---- Layout ----

QWidget *ScreenshotButton::lightIcon() const
{
    QWidget *byName = statusBar->findChild<QWidget *>(QString::fromLatin1(LIGHT_ICON_NAME));
    if (byName) {
        return byName;
    }
    QList<QWidget *> widgets = statusBar->findChildren<QWidget *>();
    for (int i = 0; i < widgets.size(); i++) {
        if (widgets[i]->inherits(LIGHT_ICON_CLASS)) {
            return widgets[i];
        }
    }
    return NULL;
}

QBoxLayout *ScreenshotButton::iconLayout() const
{
    QWidget *container = statusBar->findChild<QWidget *>(QString::fromLatin1(ICON_CONTAINER_NAME));
    return container ? qobject_cast<QBoxLayout *>(container->layout()) : NULL;
}

// ---- On and off ----

void ScreenshotButton::updateFromFlagFile()
{
    HelperFlags::ButtonState state = HelperFlags::readButtonState();
    if (state != HelperFlags::BUTTON_UNKNOWN) {
        isWanted = state == HelperFlags::BUTTON_ON;
    }
    showButton(isWanted);
}

void ScreenshotButton::showButton(bool isShown)
{
    if (isShown && !button) {
        addButton();
    } else if (!isShown && button) {
        removeButton();
    }
}

void ScreenshotButton::addButton()
{
    QBoxLayout *layout = iconLayout();
    if (!layout) {
        nh_log("status bar has no iconContainer layout, no button (unexpected firmware layout)");
        return;
    }
    QWidget *light = lightIcon();
    QSize size = light ? light->sizeHint() : QSize();
    if (!size.isValid() || size.width() < 1 || size.height() < 1) {
        size = QSize(FALLBACK_ICON_SIZE, FALLBACK_ICON_SIZE);
    }
    int imageSize = qMin(size.width(), size.height());
    pixmaps = IconPixmaps::load(imageSize);
    if (!pixmaps.isUsable()) {
        return;
    }
    QWidget *icon = NickelBridge::createStatusBarIcon(layout->parentWidget());
    if (!icon) {
        return;
    }
    icon->setObjectName(QString::fromLatin1(BUTTON_OBJECT_NAME));
    icon->setFixedSize(size);
    NickelBridge::setIconPixmaps(icon, pixmaps.normal, pixmaps.pressed);
    connect(icon, SIGNAL(tapped(bool)), this, SLOT(onTapped()));
    // Just before the brightness icon; first in the row when that icon is not found.
    int index = light ? layout->indexOf(light) : 0;
    layout->insertWidget(qMax(index, 0), icon);
    icon->show();
    button = icon;
    HelperLog::debug("button added at index %d, %dx%d px", index, size.width(), size.height());
}

void ScreenshotButton::removeButton()
{
    QWidget *icon = button.data();
    icon->hide();
    if (iconLayout()) {
        iconLayout()->removeWidget(icon);
    }
    icon->deleteLater();
    HelperLog::debug("button removed");
}

bool ScreenshotButton::eventFilter(QObject *watched, QEvent *event)
{
    if (watched != statusBar) {
        return QObject::eventFilter(watched, event);
    }
    if (event->type() == QEvent::Show) {
        updateFromFlagFile();
    }
    return QObject::eventFilter(watched, event);
}

// ---- Capture ----
// Blank the button (same size, so the other icons stay put), wait for Qt to repaint, save, restore,
// then check the file and toast: the toast is made after the capture, so it is never in the image.

void ScreenshotButton::onTapped()
{
    if (isCapturing || !button) {
        return;
    }
    updateFromFlagFile();
    if (!isWanted || !button) {
        return; // switched off while the bar was hidden: the button is gone, take nothing
    }
    isCapturing = true;
    nameBeforeCapture = NickelBridge::screenshotName();
    blankButton();
    QTimer::singleShot(CAPTURE_DELAY_MILLISECONDS, this, SLOT(finishCapture()));
}

void ScreenshotButton::blankButton()
{
    if (button) {
        NickelBridge::setIconPixmaps(button.data(), pixmaps.blank, pixmaps.blank);
        button->update();
    }
}

void ScreenshotButton::restoreButton()
{
    if (button) {
        NickelBridge::setIconPixmaps(button.data(), pixmaps.normal, pixmaps.pressed);
        button->update();
    }
}

void ScreenshotButton::finishCapture()
{
    CaptureCheck::FileState before = CaptureCheck::statFile(qPrintable(nameBeforeCapture));
    bool wasSaveCalled = NickelBridge::saveScreenshot();
    restoreButton();
    isCapturing = false;
    QString nameAfter = NickelBridge::screenshotName();
    CaptureCheck::FileState after = CaptureCheck::statFile(qPrintable(nameAfter));
    bool isSaved = wasSaveCalled && CaptureCheck::didSaveNewFile(nameAfter != nameBeforeCapture, before, after);
    reportCapture(isSaved, nameAfter);
}

void ScreenshotButton::reportCapture(bool isSaved, const QString &fileName)
{
    HelperLog::debug("capture %s: %s", isSaved ? "saved" : "failed", qPrintable(fileName));
    if (isSaved) {
        NickelBridge::showToast(HelperStrings::text(HelperStrings::SAVED), QFileInfo(fileName).fileName());
        return;
    }
    NickelBridge::showToast(HelperStrings::text(HelperStrings::FAILED), QStringLiteral(""));
}
