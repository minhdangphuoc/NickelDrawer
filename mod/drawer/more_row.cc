#include "more_row.h"

#include <QPixmap>
#include <QVBoxLayout>
#include <QWidget>

#include <NickelHook.h>

#include "debug_log.h"
#include "nickel_bridge.h"

static const char *const CONTAINER_LAYOUT_NAME = "moreContainerLayout";
static const char *const TAPPED_SIGNAL_SIGNATURE = "tapped()";

static int insertIndexAfterAnchor(QWidget *moreView, QVBoxLayout *layout, const char *anchorName)
{
    QWidget *anchor = moreView->findChild<QWidget *>(QString::fromLatin1(anchorName));
    int anchorIndex = anchor ? layout->indexOf(anchor) : -1;
    if (anchorIndex < 0) {
        nh_log("more row: anchor %s not found, appending at the end", anchorName);
        return layout->count();
    }
    return anchorIndex + 1;
}

static const int ROW_ICON_SIDE = 50; // Nickel's More icons on the Libra 2 class (beta_features.png)

// QPixmap reads the whole file and closes it before returning: nothing stays
// open on /mnt/onboard (USB mass storage must be able to take it).
static QPixmap loadIcon(const MoreRow::Spec &spec)
{
    if (spec.nickelIcon) {
        QPixmap nickelIcon(QString::fromLatin1(spec.nickelIcon));
        if (!nickelIcon.isNull()) {
            DebugLog::write("more row: Nickel icon %s (%dx%d)", spec.nickelIcon, nickelIcon.width(), nickelIcon.height());
            return nickelIcon;
        }
        nh_log("more row: Nickel icon %s missing, trying %s", spec.nickelIcon, spec.ownIcon);
    }
    QPixmap ownIcon(QString::fromLatin1(spec.ownIcon));
    if (ownIcon.isNull()) {
        nh_log("more row: icon %s missing or invalid, %s row without icon", spec.ownIcon, spec.objectName);
        return ownIcon;
    }
    DebugLog::write("more row: icon %s loaded (%dx%d)", spec.ownIcon, ownIcon.width(), ownIcon.height());
    if (ownIcon.width() > ROW_ICON_SIDE || ownIcon.height() > ROW_ICON_SIDE) {
        return ownIcon.scaled(ROW_ICON_SIDE, ROW_ICON_SIDE, Qt::KeepAspectRatio, Qt::SmoothTransformation);
    }
    return ownIcon;
}

// The exact signal the rows connect to, checked on the live object.
static bool hasTappedSignal(QWidget *row)
{
    int signalIndex = row->metaObject()->indexOfSignal(TAPPED_SIGNAL_SIGNATURE);
    DebugLog::write("more row: %s %s signal index %d", row->metaObject()->className(), TAPPED_SIGNAL_SIGNATURE,
                    signalIndex);
    return signalIndex >= 0;
}

namespace MoreRow {

QWidget *insert(QWidget *moreView, const Spec &spec, const QString &title)
{
    if (!NickelBridge::hasIconLeftButton()) {
        nh_log("more row: IconLeftButton unavailable, no %s row", spec.objectName);
        return NULL;
    }
    QVBoxLayout *layout = moreView->findChild<QVBoxLayout *>(QString::fromLatin1(CONTAINER_LAYOUT_NAME));
    if (!layout || !layout->parentWidget()) {
        nh_log("more row: layout %s not found, no %s row", CONTAINER_LAYOUT_NAME, spec.objectName);
        return NULL;
    }
    QWidget *row = NickelBridge::createIconLeftButton(layout->parentWidget());
    if (!row) {
        return NULL;
    }
    row->setObjectName(QString::fromLatin1(spec.objectName));
    NickelBridge::setIconLeftButtonPixmap(row, loadIcon(spec));
    NickelBridge::setIconLeftButtonText(row, title);
    if (!hasTappedSignal(row)) {
        nh_log("more row: %s has no %s signal, removing the %s row", row->metaObject()->className(),
               TAPPED_SIGNAL_SIGNATURE, spec.objectName);
        delete row;
        return NULL;
    }
    int index = insertIndexAfterAnchor(moreView, layout, spec.anchorName);
    layout->insertWidget(index, row);
    nh_log("more row: %s added at index %d", spec.objectName, index);
    return row;
}

} // namespace MoreRow
