#include "screenshot_button.h"

#include <QBoxLayout>
#include <QEvent>
#include <QFileInfo>
#include <QLabel>
#include <QLayoutItem>
#include <QMargins>
#include <QList>
#include <QTimer>
#include <QWidget>

#include <NickelHook.h>

#include "capture_check.h"
#include "debug_log.h"
#include "helper_flags.h"
#include "helper_strings.h"
#include "icon_metrics.h"
#include "nickel_bridge.h"

static const char BUTTON_OBJECT_NAME[] = "nhScreenshotIcon";
static const char ICON_CONTAINER_NAME[] = "iconContainer";
static const char LIGHT_ICON_NAME[] = "lightIcon";
static const char LIGHT_ICON_CLASS[] = "StatusBarLightIcon";
static const int RUN_FROM_EVENT_LOOP = 0;
// Time for Qt to repaint the bar without the button and compose it into the screen image that
// saveCurrentScreen() copies. hide() or update() alone only schedule a paint, so the capture waits a
// few event-loop passes. Tune on the device (helpers/extras/docs/statusbar-screenshot.md, section 3).
static const int CAPTURE_DELAY_MILLISECONDS = 250;

static QList<QPointer<ScreenshotButton> > liveControllers;

ScreenshotButton::ScreenshotButton(QWidget *bar)
    : QObject(bar), statusBar(bar), isWanted(true), isCapturing(false), isGeometryLogPending(false)
{
}

void ScreenshotButton::attachTo(QWidget *statusBar)
{
    if (statusBar->findChild<ScreenshotButton *>()) {
        return; // a second construction hook for the same bar
    }
    ScreenshotButton *controller = new ScreenshotButton(statusBar);
    liveControllers.append(QPointer<ScreenshotButton>(controller));
    statusBar->installEventFilter(controller);
    controller->updateFromFlagFile();
}

void ScreenshotButton::refreshAll()
{
    for (int i = liveControllers.size() - 1; i >= 0; i--) {
        if (liveControllers[i].isNull()) {
            liveControllers.removeAt(i);
        } else {
            liveControllers[i]->updateFromFlagFile();
        }
    }
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

// The side of the pixmap Nickel shows in a status-bar icon, 0 when it cannot be read.
static int pixmapSideOf(QWidget *icon)
{
    QLabel *label = qobject_cast<QLabel *>(icon);
    const QPixmap *pixmap = label ? label->pixmap() : NULL;
    return pixmap && !pixmap->isNull() ? qMin(pixmap->width(), pixmap->height()) : 0;
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

// Copies everything that makes Nickel's neighbouring icon sit where it does, nothing computed by
// hand: its size policy, minimum and maximum size, contents margins and pixmap alignment. No fixed
// size: the layout stretches every icon to the bar's height, as for Nickel's own.
void ScreenshotButton::matchNickelIcon(QWidget *icon, QWidget *light) const
{
    if (!light) {
        return;
    }
    QLabel *iconLabel = qobject_cast<QLabel *>(icon);
    QLabel *lightLabel = qobject_cast<QLabel *>(light);
    if (iconLabel && lightLabel) {
        iconLabel->setAlignment(lightLabel->alignment());
    }
    icon->setSizePolicy(light->sizePolicy());
    icon->setMinimumSize(light->minimumSize());
    icon->setMaximumSize(light->maximumSize());
    QMargins margins = light->contentsMargins();
    icon->setContentsMargins(margins.left(), margins.top(), margins.right(), margins.bottom());
}

// The alignment Nickel's layout gave the neighbour, so ours gets exactly the same.
static Qt::Alignment layoutAlignmentOf(QBoxLayout *layout, QWidget *neighbour)
{
    QLayoutItem *item = neighbour ? layout->itemAt(layout->indexOf(neighbour)) : NULL;
    return item ? item->alignment() : Qt::Alignment();
}

// The brightness icon, else the first widget of the row.
QWidget *ScreenshotButton::referenceIcon(QBoxLayout *layout) const
{
    QWidget *light = lightIcon();
    if (light) {
        return light;
    }
    QLayoutItem *first = layout->count() > 0 ? layout->itemAt(0) : NULL;
    return first ? first->widget() : NULL;
}

void ScreenshotButton::addButton()
{
    QBoxLayout *layout = iconLayout();
    if (!layout) {
        nh_log("status bar has no iconContainer layout, no button (unexpected firmware layout)");
        return;
    }
    QWidget *light = referenceIcon(layout);
    pixmaps = IconPixmaps::load(pixmapSideOf(light));
    if (!pixmaps.isUsable()) {
        return;
    }
    QWidget *icon = NickelBridge::createStatusBarIcon(layout->parentWidget());
    if (!icon) {
        return;
    }
    icon->setObjectName(QString::fromLatin1(BUTTON_OBJECT_NAME));
    matchNickelIcon(icon, light);
    NickelBridge::setIconPixmaps(icon, pixmaps.normal, pixmaps.pressed);
    connect(icon, SIGNAL(tapped(bool)), this, SLOT(onTapped()));
    // Just before the reference icon, with the alignment Nickel gave it (none: addWidget(widget, 0)).
    int index = light ? layout->indexOf(light) : 0;
    layout->insertWidget(qMax(index, 0), icon, 0, layoutAlignmentOf(layout, light));
    icon->show();
    button = icon;
    DebugLog::write("button added at index %d, canvas %d px", index, pixmaps.normal.width());
    scheduleGeometryLog();
}

void ScreenshotButton::removeButton()
{
    QWidget *icon = button.data();
    icon->hide();
    if (iconLayout()) {
        iconLayout()->removeWidget(icon);
    }
    icon->deleteLater();
    DebugLog::write("button removed");
}

bool ScreenshotButton::eventFilter(QObject *watched, QEvent *event)
{
    if (watched != statusBar) {
        return QObject::eventFilter(watched, event);
    }
    if (event->type() == QEvent::Show) {
        updateFromFlagFile();
        scheduleGeometryLog();
    }
    return QObject::eventFilter(watched, event);
}

// ---- Geometry log (device calibration) ----

void ScreenshotButton::scheduleGeometryLog()
{
    if (!DebugLog::isEnabled() || isGeometryLogPending || !button) {
        return;
    }
    isGeometryLogPending = true;
    QTimer::singleShot(RUN_FROM_EVENT_LOOP, this, SLOT(logGeometry()));
}

void ScreenshotButton::describeIcon(const char *role, QWidget *icon, QBoxLayout *layout) const
{
    QLabel *label = qobject_cast<QLabel *>(icon);
    const QPixmap *pixmap = label ? label->pixmap() : NULL;
    QLayoutItem *item = layout ? layout->itemAt(layout->indexOf(icon)) : NULL;
    int alignment = label ? static_cast<int>(label->alignment()) : -1;
    QRect geometry = icon->geometry();
    QSize pixmapSize = pixmap ? pixmap->size() : QSize(0, 0);
    int expectedTop = IconMetrics::pixmapTop(geometry.height(), pixmapSize.height(),
                                             IconMetrics::verticalAlignmentOf(alignment));
    DebugLog::write("%s %s '%s': geometry %d,%d %dx%d, hint %dx%d, min %dx%d, max %dx%d, policy %d/%d, label "
                    "alignment 0x%x, layout alignment 0x%x, pixmap %dx%d, pixmap top at y=%d (bar %d)",
                    role, icon->metaObject()->className(), qPrintable(icon->objectName()), geometry.x(),
                    geometry.y(), geometry.width(), geometry.height(), icon->sizeHint().width(),
                    icon->sizeHint().height(), icon->minimumWidth(), icon->minimumHeight(), icon->maximumWidth(),
                    icon->maximumHeight(), static_cast<int>(icon->sizePolicy().horizontalPolicy()),
                    static_cast<int>(icon->sizePolicy().verticalPolicy()), alignment,
                    item ? static_cast<int>(item->alignment()) : -1, pixmapSize.width(), pixmapSize.height(),
                    geometry.y() + expectedTop, statusBar->height());
}

// After the layout ran (0 ms timer and the bar's Show event): the geometry of the brightness icon and
// of ours side by side, so one device run shows whether the glyphs share a baseline.
void ScreenshotButton::logGeometry()
{
    isGeometryLogPending = false;
    if (!button) {
        return;
    }
    QBoxLayout *layout = iconLayout();
    QWidget *light = referenceIcon(layout);
    if (light) {
        describeIcon("light icon", light, layout);
    } else {
        DebugLog::write("geometry: no light icon found");
    }
    describeIcon("our button", button.data(), layout);
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
    DebugLog::write("capture %s: %s", isSaved ? "saved" : "failed", qPrintable(fileName));
    HelperStrings::reloadLocale();
    if (isSaved) {
        NickelBridge::showToast(HelperStrings::text(HelperStrings::SAVED), QFileInfo(fileName).fileName());
        return;
    }
    NickelBridge::showToast(HelperStrings::text(HelperStrings::FAILED), QStringLiteral(""));
}
