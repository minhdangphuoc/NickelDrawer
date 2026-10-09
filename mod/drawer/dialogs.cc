#include "dialogs.h"

#include <QApplication>
#include <QDesktopWidget>
#include <QDialog>
#include <QMenu>
#include <QTimer>

#include <NickelHook.h>

#include "debug_log.h"
#include "drawer_logic.h"
#include "fallback_strings.h"
#include "nickel_bridge.h"
#include "open_popups.h"

static const char *const ACCEPT_CHOICE = "accept";
static const char *const REJECT_CHOICE = "reject";
static const char *const CLOSE_CHOICE = "close";
static const int RUN_FROM_EVENT_LOOP = 0;

// Nickel's MenuTextItem label, bold and upright for the title row (the item's
// own style sheet wins over the menu's NickelTouchMenu.qss).
static const char *const MENU_TITLE_STYLE = "#menuTextItemLabel { font-weight: bold; font-style: normal; }";

namespace Dialogs {

void showMessage(const QString &title, const QString &text)
{
    QDialog *native = NickelBridge::showMessageDialog(title, text);
    if (native) {
        OpenPopups::track(native, "message dialog");
        return;
    }
    nh_log("dialogs: Nickel OK dialog unavailable, using the plain dialog");
    QList<DialogChoice> choices;
    DialogChoice close = {QString::fromLatin1(CLOSE_CHOICE), FallbackStrings::text(FallbackStrings::CLOSE)};
    choices.append(close);
    PlainDialog *plain =
        new PlainDialog(title, text, choices, PlainDialog::BUTTONS_IN_A_ROW, PlainDialog::OUTSIDE_TAP_DISMISSES);
    plain->present();
    OpenPopups::track(plain, "message dialog (plain)");
}

} // namespace Dialogs

// ---- Confirm ----

ConfirmQuestion::ConfirmQuestion(const QString &questionTitle, const QString &questionText, const QString &accept,
                                 const QString &reject, Kind questionKind)
    : QObject(NULL), title(questionTitle), text(questionText), acceptLabel(accept), rejectLabel(reject),
      kind(questionKind), hasAnswered(false)
{
}

void ConfirmQuestion::ask()
{
    bool closesOnOutsideTap = kind == NOT_DESTRUCTIVE;
    QDialog *native = NickelBridge::createConfirmDialog(title, text, acceptLabel, rejectLabel, closesOnOutsideTap);
    if (native) {
        connect(native, SIGNAL(accepted()), this, SLOT(onAccepted()));
        connect(native, SIGNAL(rejected()), this, SLOT(onRejected()));
        connect(native, SIGNAL(finished(int)), native, SLOT(deleteLater()));
        // ui-kit §3.15: 120 s without an answer counts as cancel.
        QTimer::singleShot(DrawerLogic::CONFIRM_TIMEOUT_MILLISECONDS, native, SLOT(reject()));
        native->open();
        OpenPopups::track(native, closesOnOutsideTap ? "confirm dialog" : "confirm dialog (destructive)");
        return;
    }
    nh_log("dialogs: Nickel confirm dialog unavailable, using the plain dialog");
    QList<DialogChoice> choices;
    DialogChoice cancel = {QString::fromLatin1(REJECT_CHOICE), rejectLabel};
    DialogChoice confirm = {QString::fromLatin1(ACCEPT_CHOICE), acceptLabel};
    choices << cancel << confirm; // cancel left, confirm right
    PlainDialog *plain =
        new PlainDialog(title, text, choices, PlainDialog::BUTTONS_IN_A_ROW,
                        closesOnOutsideTap ? PlainDialog::OUTSIDE_TAP_DISMISSES : PlainDialog::OUTSIDE_TAP_IGNORED);
    connect(plain, SIGNAL(chosen(QString)), this, SLOT(onPlainChoice(QString)));
    plain->present();
    OpenPopups::track(plain, "confirm dialog (plain)");
}

void ConfirmQuestion::onAccepted()
{
    finish(true);
}

void ConfirmQuestion::onRejected()
{
    finish(false);
}

void ConfirmQuestion::onPlainChoice(const QString &id)
{
    finish(id == QLatin1String(ACCEPT_CHOICE));
}

void ConfirmQuestion::finish(bool isAccepted)
{
    if (hasAnswered) {
        return;
    }
    hasAnswered = true;
    emit answered(isAccepted);
    deleteLater();
}

// ---- Row menu ----

RowMenu::RowMenu(const QString &menuTitle, const QList<DialogChoice> &menuChoices)
    : QObject(NULL), title(menuTitle), choices(menuChoices), hasFinished(false)
{
}

void RowMenu::popup(const QPoint &anchor)
{
    if (popupNickelMenu(anchor)) {
        return;
    }
    presentPlainMenu();
}

bool RowMenu::popupNickelMenu(const QPoint &anchor)
{
    QMenu *menu = buildNickelMenu();
    if (!menu) {
        return false;
    }
    nickelMenu = menu;
    // NickelMenu's lifetime rule: the menu deletes itself once hidden.
    connect(menu, SIGNAL(aboutToHide()), this, SLOT(onNickelMenuHidden()));
    connect(menu, SIGNAL(aboutToHide()), menu, SLOT(deleteLater()));
    menu->ensurePolished();
    QSize size = menu->sizeHint();
    QPoint centre = anchor.isNull() ? QApplication::desktop()->screenGeometry().center() : anchor;
    QPoint topLeft(centre.x() - size.width() / 2, anchor.isNull() ? centre.y() - size.height() / 2 : centre.y());
    menu->popup(topLeft); // QMenu keeps it on the screen (flips above the item near the bottom)
    OpenPopups::track(menu, "row menu");
    DebugLog::write("row menu: Nickel menu %p at %d,%d (%dx%d), %d rows", static_cast<void *>(menu), topLeft.x(),
                    topLeft.y(), size.width(), size.height(), choices.size());
    return true;
}

// NULL (logged) when Nickel's menu classes are missing or a row cannot be built.
QMenu *RowMenu::buildNickelMenu()
{
    QMenu *menu = NickelBridge::createTouchMenu();
    if (!menu) {
        DebugLog::write("row menu: Nickel's NickelTouchMenu unavailable, plain menu");
        return NULL;
    }
    QWidget *titleRow = NickelBridge::addTouchMenuItem(menu, title, true, NULL, NULL);
    if (titleRow) {
        titleRow->setStyleSheet(QString::fromLatin1(MENU_TITLE_STYLE));
    }
    for (int i = 0; i < choices.size(); i++) {
        QWidget *row = NickelBridge::addTouchMenuItem(menu, choices[i].label, !titleRow && i == 0, this,
                                                      SLOT(onNickelRowTapped()));
        if (!row) {
            DebugLog::write("row menu: Nickel row %d failed, plain menu instead", i);
            delete menu;
            nickelRows.clear();
            return NULL;
        }
        nickelRows.append(row);
    }
    return menu;
}

// Inside Nickel's tap gesture handler: remember the row, hide the menu; the
// answer goes out from the event loop once the menu is gone.
void RowMenu::onNickelRowTapped()
{
    int index = nickelRows.indexOf(qobject_cast<QWidget *>(sender()));
    if (index < 0 || !choiceId.isEmpty()) {
        return;
    }
    choiceId = choices[index].id;
    DebugLog::write("row menu: row %d (%s) tapped", index, qPrintable(choiceId));
    if (nickelMenu) {
        nickelMenu->hide();
    }
}

void RowMenu::onNickelMenuHidden()
{
    DebugLog::write("row menu: Nickel menu hidden (%s)", choiceId.isEmpty() ? "dismissed" : "row chosen");
    QTimer::singleShot(RUN_FROM_EVENT_LOOP, this, SLOT(finish()));
}

void RowMenu::presentPlainMenu()
{
    PlainDialog *plain =
        new PlainDialog(title, QString(), choices, PlainDialog::MENU_ROWS, PlainDialog::OUTSIDE_TAP_DISMISSES);
    connect(plain, SIGNAL(chosen(QString)), this, SLOT(onPlainChoice(QString)));
    plain->present();
    OpenPopups::track(plain, "row menu (plain)");
}

void RowMenu::onPlainChoice(const QString &id)
{
    choiceId = id;
    QTimer::singleShot(RUN_FROM_EVENT_LOOP, this, SLOT(finish()));
}

void RowMenu::finish()
{
    if (hasFinished) {
        return;
    }
    hasFinished = true;
    emit chosen(choiceId);
    deleteLater();
}
