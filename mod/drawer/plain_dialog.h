#ifndef NICKELDRAWER_PLAIN_DIALOG_H
#define NICKELDRAWER_PLAIN_DIALOG_H

#include <QDialog>
#include <QList>
#include <QString>

struct DialogChoice {
    QString id;
    QString label;
};

// Plain Qt dialog, black on white with a 2 px border and 120 px buttons. Used
// for the long-press menu and whenever Nickel's own dialogs are unavailable.
// Deletes itself when closed.
class PlainDialog : public QDialog {
    Q_OBJECT
public:
    enum Layout { BUTTONS_IN_A_ROW, BUTTONS_STACKED };

    PlainDialog(const QString &title, const QString &text, const QList<DialogChoice> &choices, Layout layout);

    // Shows it centred on the screen, non-blocking (QDialog::open).
    void present();

signals:
    // Empty id: dismissed (outside tap, back, or the 120 s timeout).
    void chosen(const QString &id);

protected:
    void paintEvent(QPaintEvent *event);

private slots:
    void onChoiceTapped();
    void onDismissed();

private:
    QWidget *createButton(const DialogChoice &choice);

    QList<QWidget *> buttons;
    QList<QString> buttonIds;
    bool hasAnswered;
};

#endif
