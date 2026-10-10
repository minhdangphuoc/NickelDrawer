#ifndef NICKELDRAWER_TILE_MENU_H
#define NICKELDRAWER_TILE_MENU_H

#include <QList>
#include <QObject>
#include <QPoint>
#include <QString>

#include "screen_model.h"

// The long-press menu of a gallery tile: the drawer's Nickel popup menu
// (RowMenu, dialogs.h) titled with the tile's label, one row per declared
// entry. An entry with a confirm asks the destructive ConfirmQuestion first
// (only its buttons answer). Owns copies of everything it needs, so the screen
// may reload while it is open. Deletes itself after closed().
class TileMenu : public QObject {
    Q_OBJECT
public:
    TileMenu(const QString &title, const QList<TileMenuItem> &items);

    // anchor: the tile's centre in global coordinates.
    void popup(const QPoint &anchor);

signals:
    // The entry (index into the items) the user chose and, when it has a
    // confirm, accepted. Emitted before closed(), from the event loop.
    void itemChosen(int index);
    // Always emitted once: chosen, dismissed or declined.
    void closed();

private slots:
    void onMenuChoice(const QString &choiceId);
    void onConfirmAnswer(bool isAccepted);

private:
    void finish();

    QString title;
    QList<TileMenuItem> items;
    int confirmingIndex;
};

#endif
