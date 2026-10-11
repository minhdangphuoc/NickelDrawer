#include "confirm_question.h"

#include <QDialog>
#include <QTimer>

#include <NickelHook.h>

#include "debug_log.h"
#include "nickel_bridge.h"

// Without an answer the question counts as cancelled (ui-kit §3.15).
static const int CONFIRM_TIMEOUT_MILLISECONDS = 120000;

ConfirmQuestion::ConfirmQuestion(const QString &questionTitle, const QString &questionText, const QString &accept,
                                 const QString &reject)
    : QObject(NULL), title(questionTitle), text(questionText), acceptLabel(accept), rejectLabel(reject),
      hasAnswered(false)
{
}

bool ConfirmQuestion::ask()
{
    QDialog *native = NickelBridge::createConfirmDialog(title, text, acceptLabel, rejectLabel);
    if (!native) {
        nh_log("confirm: Nickel's confirmation dialog is unavailable");
        return false;
    }
    dialog = native;
    connect(native, SIGNAL(accepted()), this, SLOT(onAccepted()));
    connect(native, SIGNAL(rejected()), this, SLOT(onRejected()));
    connect(native, SIGNAL(finished(int)), native, SLOT(deleteLater()));
    QTimer::singleShot(CONFIRM_TIMEOUT_MILLISECONDS, native, SLOT(reject()));
    native->open();
    DebugLog::write("confirm: dialog %p opened", static_cast<void *>(native));
    return true;
}

void ConfirmQuestion::cancel()
{
    if (dialog) {
        dialog->reject();
    } else {
        finish(false);
    }
}

void ConfirmQuestion::onAccepted()
{
    finish(true);
}

void ConfirmQuestion::onRejected()
{
    finish(false);
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
