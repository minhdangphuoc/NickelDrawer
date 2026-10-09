#include "status_bar_inset.h"

#include <QLayout>
#include <QWidget>

#include "debug_log.h"

static const char STATUS_BAR_OBJECT_NAME[] = "StatusBarView";
// Nickel: 126 px bar, 128 px topMargin (Storm entries of its QSS).
static const int GAP_BELOW_STATUS_BAR = 2;
// More than this share of the page "under the bar" means the measurement is wrong.
static const int MAX_INSET_DIVISOR = 3;

static QWidget *findStatusBar(const QWidget *view)
{
    QWidget *window = view->window();
    if (!window || window == view) {
        return NULL; // not in Nickel's window yet
    }
    return window->findChild<QWidget *>(QString::fromLatin1(STATUS_BAR_OBJECT_NAME));
}

static int globalTop(const QWidget *widget)
{
    return widget->mapToGlobal(QPoint(0, 0)).y();
}

static int globalBottom(const QWidget *widget)
{
    return widget->mapToGlobal(QPoint(0, widget->height())).y(); // exclusive
}

namespace StatusBarInset {

int topInsetFor(const QWidget *view)
{
    QWidget *statusBar = findStatusBar(view);
    if (!statusBar) {
        DebugLog::write("status bar: %s not found above %s, using fallback inset %d", STATUS_BAR_OBJECT_NAME,
                        qPrintable(view->objectName()), FALLBACK_TOP_INSET);
        return FALLBACK_TOP_INSET;
    }
    if (statusBar->isHidden()) {
        DebugLog::write("status bar: hidden, no inset");
        return 0;
    }
    int covered = globalBottom(statusBar) - globalTop(view);
    if (covered <= 0) {
        DebugLog::write("status bar: ends %d px above the view, no inset", -covered);
        return 0;
    }
    int inset = covered + GAP_BELOW_STATUS_BAR;
    if (view->height() > 0 && inset > view->height() / MAX_INSET_DIVISOR) {
        DebugLog::write("status bar: measured inset %d too large for view height %d, using fallback %d", inset,
                        view->height(), FALLBACK_TOP_INSET);
        return FALLBACK_TOP_INSET;
    }
    DebugLog::write("status bar: covers %d px of the view, inset %d", covered, inset);
    return inset;
}

static void logWidget(const char *stage, const char *role, const QWidget *widget)
{
    QRect local = widget->geometry();
    DebugLog::write("geometry %s: %s %s (%s) local %d,%d %dx%d global y %d..%d visible %d hidden %d", stage, role,
                    qPrintable(widget->objectName()), widget->metaObject()->className(), local.x(), local.y(),
                    local.width(), local.height(), globalTop(widget), globalBottom(widget), widget->isVisible(),
                    widget->isHidden());
}

void logGeometry(const char *stage, const QWidget *view, const QWidget *header)
{
    if (!DebugLog::isEnabled()) {
        return;
    }
    logWidget(stage, "view", view);
    if (view->parentWidget()) {
        logWidget(stage, "view parent", view->parentWidget());
    }
    if (view->layout()) {
        QMargins margins = view->layout()->contentsMargins();
        DebugLog::write("geometry %s: view layout margins l%d t%d r%d b%d", stage, margins.left(), margins.top(),
                        margins.right(), margins.bottom());
    }
    QWidget *statusBar = findStatusBar(view);
    if (statusBar) {
        logWidget(stage, "status bar", statusBar);
    } else {
        DebugLog::write("geometry %s: status bar %s not found", stage, STATUS_BAR_OBJECT_NAME);
    }
    if (!header) {
        return;
    }
    logWidget(stage, "header", header);
    if (statusBar && !statusBar->isHidden()) {
        int overlap = globalBottom(statusBar) - globalTop(header);
        DebugLog::write("geometry %s: header overlap with status bar %d px (%s)", stage, overlap > 0 ? overlap : 0,
                        overlap > 0 ? "header hidden under the bar" : "header below the bar");
    }
}

} // namespace StatusBarInset
