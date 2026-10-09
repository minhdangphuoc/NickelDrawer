#include "entry_row.h"

#include <QEvent>
#include <QPixmap>
#include <QStringList>
#include <QTimer>
#include <QVBoxLayout>
#include <QWidget>

#include <NickelHook.h>

#include "debug_log.h"
#include "drawer_navigator.h"
#include "fallback_strings.h"
#include "model_client.h"
#include "nickel_bridge.h"

static const char *const CONTAINER_LAYOUT_NAME = "moreContainerLayout";
static const char *const ANCHOR_ROW_NAME = "betaFeaturesButton"; // our row goes right after it
static const char *const ROW_OBJECT_NAME = "nickelDrawerButton";
// Placeholder icon from Nickel's own resources until the drawer has its own.
static const char *const ROW_ICON = ":/images/reading/beta_features.png";
static const int DOT_RIGHT_MARGIN = 72;
static const char *const TAPPED_SIGNAL_SIGNATURE = "tapped()";
static const int RUN_FROM_EVENT_LOOP = 0;

static int insertIndexAfterAnchor(QWidget *moreView, QVBoxLayout *layout)
{
    QWidget *anchor = moreView->findChild<QWidget *>(QString::fromLatin1(ANCHOR_ROW_NAME));
    int anchorIndex = anchor ? layout->indexOf(anchor) : -1;
    if (anchorIndex < 0) {
        nh_log("entry row: anchor %s not found, appending at the end", ANCHOR_ROW_NAME);
        return layout->count();
    }
    return anchorIndex + 1;
}

// The exact signal EntryRow connects to, checked on the live object.
static bool hasTappedSignal(QWidget *row)
{
    int signalIndex = row->metaObject()->indexOfSignal(TAPPED_SIGNAL_SIGNATURE);
    DebugLog::write("entry row: %s %s signal index %d", row->metaObject()->className(), TAPPED_SIGNAL_SIGNATURE,
                    signalIndex);
    return signalIndex >= 0;
}

QString entrySummary(const DrawerModel &model, int *updateCount)
{
    QStringList parts;
    *updateCount = 0;
    for (int section = 0; section < model.sections.size(); section++) {
        const QList<DrawerItem> &items = model.sections[section].items;
        for (int i = 0; i < items.size(); i++) {
            if (items[i].hasUpdate) {
                (*updateCount)++;
            }
            if (items[i].isRunning && !items[i].status.isEmpty()) {
                parts << FallbackStrings::text(FallbackStrings::STATUS_LINE)
                             .replace(QStringLiteral("{name}"), items[i].name)
                             .replace(QStringLiteral("{status}"), items[i].status);
            }
        }
    }
    if (*updateCount == 1) {
        parts << FallbackStrings::text(FallbackStrings::UPDATES_COUNT_ONE);
    } else if (*updateCount > 1) {
        parts << FallbackStrings::text(FallbackStrings::UPDATES_COUNT_MANY)
                     .replace(QStringLiteral("{n}"), QString::number(*updateCount));
    }
    return parts.join(FallbackStrings::text(FallbackStrings::SUMMARY_SEPARATOR));
}

void EntryRow::addTo(QWidget *moreView)
{
    if (!NickelBridge::hasIconLeftButton()) {
        nh_log("entry row: IconLeftButton unavailable, no Apps row");
        return;
    }
    QVBoxLayout *layout = moreView->findChild<QVBoxLayout *>(QString::fromLatin1(CONTAINER_LAYOUT_NAME));
    if (!layout || !layout->parentWidget()) {
        nh_log("entry row: layout %s not found, no Apps row", CONTAINER_LAYOUT_NAME);
        return;
    }
    FallbackStrings::reloadLocale();
    QWidget *row = NickelBridge::createIconLeftButton(layout->parentWidget());
    if (!row) {
        return;
    }
    row->setObjectName(QString::fromLatin1(ROW_OBJECT_NAME));
    QPixmap icon(QString::fromLatin1(ROW_ICON));
    DebugLog::write("entry row: icon %s %s (%dx%d)", ROW_ICON, icon.isNull() ? "missing" : "loaded", icon.width(),
                    icon.height());
    NickelBridge::setIconLeftButtonPixmap(row, icon);
    NickelBridge::setIconLeftButtonText(row, FallbackStrings::text(FallbackStrings::APPS_TITLE));
    DebugLog::write("entry row: pixmap and text set");

    if (!hasTappedSignal(row)) {
        nh_log("entry row: %s has no %s signal, removing the row", row->metaObject()->className(),
               TAPPED_SIGNAL_SIGNATURE);
        delete row;
        return;
    }
    EntryRow *entry = new EntryRow(row);
    if (!QObject::connect(row, SIGNAL(tapped()), entry, SLOT(onTapped()))) {
        nh_log("entry row: could not connect tapped(), removing the row");
        delete row; // the entry follows through destroyed()
        return;
    }
    int index = insertIndexAfterAnchor(moreView, layout);
    layout->insertWidget(index, row);
    nh_log("entry row: added at index %d", index);
    DebugLog::write("entry row: row %p entry %p, %d children on the row", (void *)row, (void *)entry,
                    row->children().size());
    entry->refreshSummary();
}

EntryRow::EntryRow(QWidget *moreRow)
    : QObject(NULL), row(moreRow), updateDot(new UpdateDot(moreRow)), client(new ModelClient(this)),
      isTapPending(false)
{
    updateDot->hide();
    connect(moreRow, SIGNAL(destroyed()), this, SLOT(deleteLater()));
    moreRow->installEventFilter(this);
    connect(client, SIGNAL(modelLoaded(DrawerModel)), this, SLOT(onModelLoaded(DrawerModel)));
    connect(DrawerNavigator::instance(), SIGNAL(drawerClosed()), this, SLOT(refreshSummary()));
}

void EntryRow::onTapped()
{
    nh_log("entry row: tapped");
    DebugLog::write("entry row: tapped() received by %p, sender %p, row %s", (void *)this, (void *)sender(),
                    row ? "alive" : "gone");
    if (isTapPending) {
        DebugLog::write("entry row: tap already pending, ignoring");
        return;
    }
    isTapPending = true;
    QTimer::singleShot(RUN_FROM_EVENT_LOOP, this, SLOT(handleTap()));
}

void EntryRow::handleTap()
{
    isTapPending = false;
    if (!row) {
        DebugLog::write("entry row: row gone before the tap was handled, ignoring");
        return;
    }
    DebugLog::write("entry row: opening the drawer");
    DrawerNavigator::instance()->open();
    DebugLog::write("entry row: drawer open returned");
}

void EntryRow::refreshSummary()
{
    client->requestModel();
}

void EntryRow::onModelLoaded(const DrawerModel &model)
{
    if (!model.isUsable) {
        showSummary(QString(), false); // appsctl failed: the row just reads "Apps"
        return;
    }
    int updateCount = 0;
    QString summary = entrySummary(model, &updateCount);
    showSummary(summary, updateCount > 0);
}

void EntryRow::showSummary(const QString &summary, bool hasUpdate)
{
    if (!row || !updateDot) {
        return; // the More view was rebuilt while appsctl ran
    }
    QString title = FallbackStrings::text(FallbackStrings::APPS_TITLE);
    // IconLeftButton has one text; the status goes on a second line of it.
    NickelBridge::setIconLeftButtonText(row, summary.isEmpty() ? title : title + QLatin1Char('\n') + summary);
    updateDot->setVisible(hasUpdate);
    placeDot();
}

void EntryRow::placeDot()
{
    if (!row || !updateDot) {
        return;
    }
    updateDot->move(row->width() - DOT_RIGHT_MARGIN - updateDot->width(), (row->height() - updateDot->height()) / 2);
    updateDot->raise();
}

bool EntryRow::eventFilter(QObject *watched, QEvent *event)
{
    if (row && watched == row.data() && event->type() == QEvent::Resize) {
        placeDot();
    }
    return QObject::eventFilter(watched, event);
}
