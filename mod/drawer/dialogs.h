#ifndef NICKELDRAWER_DIALOGS_H
#define NICKELDRAWER_DIALOGS_H

#include <QList>
#include <QObject>
#include <QPoint>
#include <QPointer>
#include <QString>

#include "plain_dialog.h"

class QMenu;
class QWidget;

// Result, confirm and menu popups. Nickel's own (through the bridge) when
// available, else PlainDialog. All non-blocking, all tracked by OpenPopups so
// the drawer can close them, all closed by a tap outside unless they guard a
// destructive choice; each keeps a visible OK or Cancel (menus close by a tap
// outside, as Nickel's own popup menus do).
namespace Dialogs {

// An info or result text with an OK button.
void showMessage(const QString &title, const QString &text);

} // namespace Dialogs

// One yes/no question. Emits answered() exactly once (cancel, dismiss and the
// 120 s timeout all count as false), then deletes itself.
class ConfirmQuestion : public QObject {
    Q_OBJECT
public:
    // A destructive question (Remove) is not closed by a tap outside: only
    // its Cancel or its accept button answers it.
    enum Kind { NOT_DESTRUCTIVE, DESTRUCTIVE };

    ConfirmQuestion(const QString &title, const QString &text, const QString &acceptLabel, const QString &rejectLabel,
                    Kind kind);
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
    Kind kind;
    bool hasAnswered;
};

// The long-press row menu: Nickel's NickelTouchMenu (bold title row, then one
// row per choice in the given order) popped up at the pressed item, else
// PlainDialog's menu rows. Emits chosen() exactly once, from the event loop
// (empty id when dismissed), then deletes itself.
class RowMenu : public QObject {
    Q_OBJECT
public:
    RowMenu(const QString &title, const QList<DialogChoice> &choices);

    // anchor: the pressed item's centre in global coordinates; null: the screen centre.
    void popup(const QPoint &anchor);

signals:
    void chosen(const QString &id);

private slots:
    void onNickelRowTapped();
    void onNickelMenuHidden();
    void onPlainChoice(const QString &id);
    void finish();

private:
    bool popupNickelMenu(const QPoint &anchor);
    QMenu *buildNickelMenu();
    void presentPlainMenu();

    QString title;
    QList<DialogChoice> choices;
    QList<QWidget *> nickelRows; // parallel to choices
    QPointer<QMenu> nickelMenu;
    QString choiceId;
    bool hasFinished;
};

#endif
