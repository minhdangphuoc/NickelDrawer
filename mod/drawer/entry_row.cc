#include "entry_row.h"

#include <QEvent>
#include <QPixmap>
#include <QStringList>
#include <QVBoxLayout>
#include <QWidget>

#include <NickelHook.h>

#include "drawer_navigator.h"
#include "fallback_strings.h"
#include "model_client.h"
#include "nickel_bridge.h"
#include "painted_marks.h"

static const char *const CONTAINER_LAYOUT_NAME = "moreContainerLayout";
static const char *const ANCHOR_ROW_NAME = "betaFeaturesButton"; // our row goes right after it
static const char *const ROW_OBJECT_NAME = "nickelDrawerButton";
// Placeholder icon from Nickel's own resources until the drawer has its own.
static const char *const ROW_ICON = ":/images/reading/beta_features.png";
static const int DOT_RIGHT_MARGIN = 72;

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
    NickelBridge::setIconLeftButtonPixmap(row, QPixmap(QString::fromLatin1(ROW_ICON)));
    NickelBridge::setIconLeftButtonText(row, FallbackStrings::text(FallbackStrings::APPS_TITLE));

    EntryRow *entry = new EntryRow(row);
    if (!QObject::connect(row, SIGNAL(tapped()), entry, SLOT(onTapped()))) {
        nh_log("entry row: could not connect tapped(), removing the row");
        delete row; // also deletes the entry (its child)
        return;
    }
    int index = insertIndexAfterAnchor(moreView, layout);
    layout->insertWidget(index, row);
    nh_log("entry row: added at index %d", index);
    entry->refreshSummary();
}

EntryRow::EntryRow(QWidget *moreRow)
    : QObject(moreRow), row(moreRow), updateDot(new UpdateDot(moreRow)), client(new ModelClient(this))
{
    updateDot->hide();
    row->installEventFilter(this);
    connect(client, SIGNAL(modelLoaded(DrawerModel)), this, SLOT(onModelLoaded(DrawerModel)));
    connect(DrawerNavigator::instance(), SIGNAL(drawerClosed()), this, SLOT(refreshSummary()));
}

void EntryRow::onTapped()
{
    nh_log("entry row: tapped");
    DrawerNavigator::instance()->open();
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
    QString title = FallbackStrings::text(FallbackStrings::APPS_TITLE);
    // IconLeftButton has one text; the status goes on a second line of it.
    NickelBridge::setIconLeftButtonText(row, summary.isEmpty() ? title : title + QLatin1Char('\n') + summary);
    updateDot->setVisible(hasUpdate);
    placeDot();
}

void EntryRow::placeDot()
{
    updateDot->move(row->width() - DOT_RIGHT_MARGIN - updateDot->width(), (row->height() - updateDot->height()) / 2);
    updateDot->raise();
}

bool EntryRow::eventFilter(QObject *watched, QEvent *event)
{
    if (watched == row && event->type() == QEvent::Resize) {
        placeDot();
    }
    return QObject::eventFilter(watched, event);
}
