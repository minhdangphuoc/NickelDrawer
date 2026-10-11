#ifndef EXTRAS_CONFIRM_QUESTION_H
#define EXTRAS_CONFIRM_QUESTION_H

#include <QObject>
#include <QPointer>
#include <QString>

class QDialog;

// One yes/no question in Nickel's own ConfirmationDialog. Emits answered() exactly once (cancel, a tap
// outside the dialog, cancel() and the 120 s timeout all count as false), then deletes itself. There
// is no stand-in dialog: when Nickel's is unavailable ask() says so and the caller must not run the
// action it was guarding.
class ConfirmQuestion : public QObject {
    Q_OBJECT
public:
    ConfirmQuestion(const QString &title, const QString &text, const QString &acceptLabel,
                    const QString &rejectLabel);

    // false (logged, nothing shown, answered() never emitted) when Nickel's dialog is unavailable.
    bool ask();
    // Closes the dialog as a "no" (the panel went away).
    void cancel();

signals:
    void answered(bool isAccepted);

private slots:
    void onAccepted();
    void onRejected();

private:
    void finish(bool isAccepted);

    QString title;
    QString text;
    QString acceptLabel;
    QString rejectLabel;
    QPointer<QDialog> dialog;
    bool hasAnswered;
};

#endif
