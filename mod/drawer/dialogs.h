#ifndef NICKELDRAWER_DIALOGS_H
#define NICKELDRAWER_DIALOGS_H

#include <QList>
#include <QObject>
#include <QString>

#include "plain_dialog.h"

class QDialog;

// Result, confirm and menu dialogs. Nickel's own dialogs (through the bridge)
// when available, else PlainDialog. All non-blocking.
namespace Dialogs {

void showMessage(const QString &title, const QString &text);

} // namespace Dialogs

// One yes/no question. Emits answered() exactly once (cancel, dismiss and the
// 120 s timeout all count as false), then deletes itself.
class ConfirmQuestion : public QObject {
    Q_OBJECT
public:
    ConfirmQuestion(const QString &title, const QString &text, const QString &acceptLabel, const QString &rejectLabel);
    void ask();

signals:
    void answered(bool isAccepted);

private slots:
    void onAccepted();
    void onRejected();
    void onPlainChoice(const QString &id);

private:
    void finish(bool isAccepted);

    QString title;
    QString text;
    QString acceptLabel;
    QString rejectLabel;
    bool hasAnswered;
};

// The long-press row menu; emits the chosen id (empty when dismissed).
PlainDialog *createRowMenu(const QString &title, const QList<DialogChoice> &choices);

#endif
