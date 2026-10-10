#include "tweaks_row.h"

#include <QTimer>

#include <NickelHook.h>

#include "debug_log.h"
#include "entry_row.h"
#include "fallback_strings.h"
#include "more_row.h"
#include "tweaks_session.h"

static const char *const ROW_OBJECT_NAME = "nickelDrawerTweaksButton";
// The gear of Nickel's own Settings row (Ui_MoreView::setupUi); the Tabler
// "adjustments-horizontal" icon installed by koboapps when it is missing.
static const char *const NICKEL_ROW_ICON = ":/images/reading/settings_gear.png";
static const char *const OWN_ROW_ICON = "/mnt/onboard/.adds/koboapps/icons/ui/adjustments-horizontal-64.png";
// Without the Apps row (IconLeftButton fine but its row failed) it follows Beta Features.
static const char *const SECOND_ANCHOR_NAME = "betaFeaturesButton";
static const int RUN_FROM_EVENT_LOOP = 0;

void TweaksRow::addTo(QWidget *moreView)
{
    const char *appsRowName = EntryRow::rowObjectName();
    bool hasAppsRow = moreView->findChild<QWidget *>(QString::fromLatin1(appsRowName)) != NULL;
    MoreRow::Spec spec = {ROW_OBJECT_NAME, NICKEL_ROW_ICON, OWN_ROW_ICON,
                          hasAppsRow ? appsRowName : SECOND_ANCHOR_NAME};
    QWidget *row = MoreRow::insert(moreView, spec, FallbackStrings::text(FallbackStrings::TWEAKS_TITLE));
    if (!row) {
        return;
    }
    TweaksRow *entry = new TweaksRow(row);
    if (!QObject::connect(row, SIGNAL(tapped()), entry, SLOT(onTapped()))) {
        nh_log("tweaks row: could not connect tapped(), removing the row");
        delete row; // the entry follows through destroyed()
    }
}

TweaksRow::TweaksRow(QWidget *moreRow) : QObject(NULL), row(moreRow), isTapPending(false)
{
    connect(moreRow, SIGNAL(destroyed()), this, SLOT(deleteLater()));
}

void TweaksRow::onTapped()
{
    DebugLog::write("tweaks row: tapped()");
    if (isTapPending) {
        return;
    }
    isTapPending = true;
    QTimer::singleShot(RUN_FROM_EVENT_LOOP, this, SLOT(handleTap()));
}

void TweaksRow::handleTap()
{
    isTapPending = false;
    if (!row) {
        DebugLog::write("tweaks row: row gone before the tap was handled, ignoring");
        return;
    }
    TweaksSession::open();
}
