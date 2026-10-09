#ifndef NICKELDRAWER_PLAIN_DIALOG_H
#define NICKELDRAWER_PLAIN_DIALOG_H

#include <QDialog>
#include <QList>
#include <QPixmap>
#include <QString>

class QVBoxLayout;

struct DialogChoice {
    QString id;
    QString label;
};

// Plain Qt popup, used whenever Nickel's own dialog or menu is unavailable.
// Black on white. It covers the whole screen with a snapshot of the page
// under it, so a tap outside its box reaches it: that tap dismisses it unless
// it guards a destructive choice (the box keeps its buttons for that).
//   BUTTONS_IN_A_ROW: a message or a question, bordered 120 px buttons.
//   MENU_ROWS:        Nickel's popup menu look (NickelTouchMenu.qss): a
//                     bordered box, a bold title row, plain 120 px text rows
//                     with thin dividers, the pressed row inverted.
// Deletes itself when closed.
class PlainDialog : public QDialog {
    Q_OBJECT
public:
    enum Layout { BUTTONS_IN_A_ROW, MENU_ROWS };
    enum OutsideTap { OUTSIDE_TAP_DISMISSES, OUTSIDE_TAP_IGNORED };

    PlainDialog(const QString &title, const QString &text, const QList<DialogChoice> &choices, Layout layout,
                OutsideTap outsideTap);

    // Shows it over the whole screen, the box centred, non-blocking (QDialog::open).
    void present();

signals:
    // Empty id: dismissed (outside tap, closed by the drawer, or the 120 s timeout).
    void chosen(const QString &id);

protected:
    void paintEvent(QPaintEvent *event);
    void resizeEvent(QResizeEvent *event);

private slots:
    void onChoiceTapped();
    void onOutsideTapped();
    void onDismissed();

private:
    void addMessageContent(const QString &title, const QString &text, const QList<DialogChoice> &choices);
    void addMenuContent(const QString &title, const QList<DialogChoice> &choices);
    QWidget *createButton(const DialogChoice &choice);
    QWidget *createMenuRow(const DialogChoice &choice);
    QWidget *createDivider();
    void captureBackdrop();

    QWidget *outsideArea; // under the box, covering the screen
    QWidget *box;
    QVBoxLayout *boxLayout;
    QPixmap backdrop; // the page under the popup; white when it cannot be grabbed
    QList<QWidget *> buttons;
    QList<QString> buttonIds;
    OutsideTap outsideTapRule;
    bool hasAnswered;
};

#endif
