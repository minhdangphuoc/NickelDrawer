#include "dialogs.h"

#include <QDialog>
#include <QTimer>

#include <NickelHook.h>

#include "drawer_logic.h"
#include "fallback_strings.h"
#include "nickel_bridge.h"

static const char *const ACCEPT_CHOICE = "accept";
static const char *const REJECT_CHOICE = "reject";
static const char *const CLOSE_CHOICE = "close";

namespace Dialogs {

void showMessage(const QString &title, const QString &text)
{
    if (NickelBridge::showMessageDialog(title, text)) {
        return;
    }
    nh_log("dialogs: Nickel OK dialog unavailable, using the plain dialog");
    QList<DialogChoice> choices;
    DialogChoice close = {QString::fromLatin1(CLOSE_CHOICE), FallbackStrings::text(FallbackStrings::CLOSE)};
    choices.append(close);
    (new PlainDialog(title, text, choices, PlainDialog::BUTTONS_IN_A_ROW))->present();
}

} // namespace Dialogs

ConfirmQuestion::ConfirmQuestion(const QString &questionTitle, const QString &questionText, const QString &accept,
                                 const QString &reject)
    : QObject(NULL), title(questionTitle), text(questionText), acceptLabel(accept), rejectLabel(reject),
      hasAnswered(false)
{
}

void ConfirmQuestion::ask()
{
    QDialog *native = NickelBridge::createConfirmDialog(title, text, acceptLabel, rejectLabel);
    if (native) {
        connect(native, SIGNAL(accepted()), this, SLOT(onAccepted()));
        connect(native, SIGNAL(rejected()), this, SLOT(onRejected()));
        connect(native, SIGNAL(finished(int)), native, SLOT(deleteLater()));
        // ui-kit §3.15: 120 s without an answer counts as cancel.
        QTimer::singleShot(DrawerLogic::CONFIRM_TIMEOUT_MILLISECONDS, native, SLOT(reject()));
        native->open();
        return;
    }
    nh_log("dialogs: Nickel confirm dialog unavailable, using the plain dialog");
    QList<DialogChoice> choices;
    DialogChoice cancel = {QString::fromLatin1(REJECT_CHOICE), rejectLabel};
    DialogChoice confirm = {QString::fromLatin1(ACCEPT_CHOICE), acceptLabel};
    choices << cancel << confirm; // cancel left, confirm right
    PlainDialog *plain = new PlainDialog(title, text, choices, PlainDialog::BUTTONS_IN_A_ROW);
    connect(plain, SIGNAL(chosen(QString)), this, SLOT(onPlainChoice(QString)));
    plain->present();
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

PlainDialog *createRowMenu(const QString &title, const QList<DialogChoice> &choices)
{
    return new PlainDialog(title, QString(), choices, PlainDialog::BUTTONS_STACKED);
}
