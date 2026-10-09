#include "more_row.h"

#include <stdlib.h>

#include <QMetaObject>
#include <QPixmap>
#include <QTimer>
#include <QVBoxLayout>
#include <QWidget>

#include <NickelHook.h>

#include "debug_log.h"
#include "device_state.h"
#include "drawer_launcher.h"
#include "nickel_api.h"

static const char *const CONTAINER_LAYOUT_NAME = "moreContainerLayout";
static const char *const ANCHOR_ROW_NAME = "betaFeaturesButton"; // our row goes right after it
static const char *const ROW_OBJECT_NAME = "nickelDrawerButton";
// Placeholder icon from Nickel's own resources until the drawer has its own.
static const char *const ROW_ICON = ":/images/reading/beta_features.png";
// IconLeftButton is 0x58 bytes on 23828; allocate generously like NickelMenu
// does for MainNavButton, zeroed, so a slightly larger class on another build fits.
static const size_t ICON_LEFT_BUTTON_ALLOCATION = 256;
static const char *const TAPPED_SIGNAL_SIGNATURE = "tapped()";
static const int RUN_FROM_EVENT_LOOP = 0;

static QWidget *createIconLeftButton(QWidget *parent)
{
    void *memory = calloc(1, ICON_LEFT_BUTTON_ALLOCATION);
    if (!memory) {
        nh_log("more row: out of memory for IconLeftButton");
        return NULL;
    }
    DebugLog::write("more row: constructing IconLeftButton at %p, parent %p", memory, (void *)parent);
    IconLeftButton_construct(memory, parent);
    QWidget *button = reinterpret_cast<QWidget *>(memory);
    DebugLog::write("more row: constructed %s", button->metaObject()->className());
    return button;
}

// The exact signal the receiver connects to, checked on the live object.
static bool hasTappedSignal(QWidget *row)
{
    int signalIndex = row->metaObject()->indexOfSignal(TAPPED_SIGNAL_SIGNATURE);
    DebugLog::write("more row: %s signal index %d", TAPPED_SIGNAL_SIGNATURE, signalIndex);
    return signalIndex >= 0;
}

static int insertIndexAfterAnchor(QWidget *moreView, QVBoxLayout *layout)
{
    QWidget *anchor = moreView->findChild<QWidget *>(QString::fromLatin1(ANCHOR_ROW_NAME));
    int anchorIndex = anchor ? layout->indexOf(anchor) : -1;
    if (anchorIndex < 0) {
        nh_log("more row: anchor %s not found, appending at the end", ANCHOR_ROW_NAME);
        return layout->count();
    }
    return anchorIndex + 1;
}

namespace MoreRow {

void addTo(QWidget *moreView)
{
    if (!NickelApi::hasIconLeftButton()) {
        nh_log("more row: IconLeftButton API missing, prototype A disabled");
        return;
    }
    QVBoxLayout *layout = moreView->findChild<QVBoxLayout *>(QString::fromLatin1(CONTAINER_LAYOUT_NAME));
    if (!layout || !layout->parentWidget()) {
        nh_log("more row: layout %s not found, prototype A disabled", CONTAINER_LAYOUT_NAME);
        return;
    }

    QWidget *row = createIconLeftButton(layout->parentWidget());
    if (!row) {
        return;
    }
    row->setObjectName(QString::fromLatin1(ROW_OBJECT_NAME));
    QPixmap icon(QString::fromLatin1(ROW_ICON));
    DebugLog::write("more row: icon %s %s (%dx%d)", ROW_ICON, icon.isNull() ? "missing" : "loaded", icon.width(),
                    icon.height());
    IconLeftButton_setPixmap(row, icon);
    IconLeftButton_setText(row, DeviceState::appsLabel());
    DebugLog::write("more row: pixmap and text set");

    if (!hasTappedSignal(row)) {
        nh_log("more row: %s has no %s signal, removing the row", row->metaObject()->className(),
               TAPPED_SIGNAL_SIGNATURE);
        delete row;
        return;
    }
    MoreRowTapReceiver *receiver = new MoreRowTapReceiver(row);
    if (!QObject::connect(row, SIGNAL(tapped()), receiver, SLOT(onTapped()))) {
        nh_log("more row: could not connect tapped(), removing the row");
        delete row; // the receiver follows through destroyed()
        return;
    }

    int index = insertIndexAfterAnchor(moreView, layout);
    layout->insertWidget(index, row);
    nh_log("more row: added at index %d", index);
    DebugLog::write("more row: row %p receiver %p, %d children on the row", (void *)row, (void *)receiver,
                    row->children().size());
}

} // namespace MoreRow

MoreRowTapReceiver::MoreRowTapReceiver(QWidget *moreRow) : QObject(NULL), row(moreRow), isTapPending(false)
{
    connect(moreRow, SIGNAL(destroyed()), this, SLOT(deleteLater()));
}

void MoreRowTapReceiver::onTapped()
{
    nh_log("more row: tapped");
    DebugLog::write("more row: tapped() received by %p, sender %p, row %s", (void *)this, (void *)sender(),
                    row ? "alive" : "gone");
    if (isTapPending) {
        DebugLog::write("more row: tap already pending, ignoring");
        return;
    }
    isTapPending = true;
    QTimer::singleShot(RUN_FROM_EVENT_LOOP, this, SLOT(handleTap()));
}

void MoreRowTapReceiver::handleTap()
{
    isTapPending = false;
    DebugLog::write("more row: handling tap, row %s", row ? "alive" : "gone");
    if (prototypeSwitches.isDrawerEnabled) {
        DebugLog::write("more row: opening prototype C");
        DrawerLauncher::open();
        DebugLog::write("more row: prototype C open returned");
        return;
    }
    DebugLog::write("more row: before toast");
    NickelApi::showToast(QStringLiteral("NickelDrawer: row tapped"));
    DebugLog::write("more row: after toast");
}

// NickelHook PLT hook target for MoreView::MoreView(QWidget*). Must call the
// original first and must never throw or crash.
extern "C" __attribute__((visibility("default"))) void nddrawer_MoreView_construct(void *moreView, QWidget *parent)
{
    MoreView_construct_original(moreView, parent);
    DebugLog::write("more view: constructed %p", moreView);
    if (prototypeSwitches.isMoreRowEnabled) {
        // MoreView derives from QWidget as its first base.
        MoreRow::addTo(reinterpret_cast<QWidget *>(moreView));
    }
}
