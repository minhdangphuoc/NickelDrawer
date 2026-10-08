#include "more_row.h"

#include <stdlib.h>

#include <QPixmap>
#include <QVBoxLayout>
#include <QWidget>

#include <NickelHook.h>

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

static QWidget *createIconLeftButton(QWidget *parent)
{
    void *memory = calloc(1, ICON_LEFT_BUTTON_ALLOCATION);
    if (!memory) {
        nh_log("more row: out of memory for IconLeftButton");
        return NULL;
    }
    IconLeftButton_construct(memory, parent);
    return reinterpret_cast<QWidget *>(memory);
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
    IconLeftButton_setPixmap(row, QPixmap(QString::fromLatin1(ROW_ICON)));
    IconLeftButton_setText(row, DeviceState::appsLabel());

    MoreRowTapReceiver *receiver = new MoreRowTapReceiver(row);
    if (!QObject::connect(row, SIGNAL(tapped()), receiver, SLOT(onTapped()))) {
        nh_log("more row: could not connect tapped(), removing the row");
        delete row; // also deletes the receiver
        return;
    }

    int index = insertIndexAfterAnchor(moreView, layout);
    layout->insertWidget(index, row);
    nh_log("more row: added at index %d", index);
}

} // namespace MoreRow

MoreRowTapReceiver::MoreRowTapReceiver(QObject *row) : QObject(row) {}

void MoreRowTapReceiver::onTapped()
{
    nh_log("more row: tapped");
    if (prototypeSwitches.isDrawerEnabled) {
        DrawerLauncher::open();
        return;
    }
    NickelApi::showToast(QStringLiteral("NickelDrawer: row tapped"));
}

// NickelHook PLT hook target for MoreView::MoreView(QWidget*). Must call the
// original first and must never throw or crash.
extern "C" __attribute__((visibility("default"))) void nddrawer_MoreView_construct(void *moreView, QWidget *parent)
{
    MoreView_construct_original(moreView, parent);
    if (prototypeSwitches.isMoreRowEnabled) {
        // MoreView derives from QWidget as its first base.
        MoreRow::addTo(reinterpret_cast<QWidget *>(moreView));
    }
}
