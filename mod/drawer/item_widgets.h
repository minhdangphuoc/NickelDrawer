#ifndef NICKELDRAWER_ITEM_WIDGETS_H
#define NICKELDRAWER_ITEM_WIDGETS_H

#include "drawer_model.h"
#include "press_area.h"

class QLabel;
class UpdateDot;

// One app in the drawer: a list row or a grid tile. Live state (status line,
// update dot) can change in place, which repaints only this widget (partial
// e-ink update).
class ItemWidget : public PressArea {
public:
    ItemWidget(const DrawerItem &item, QWidget *parent);

    QString itemId() const { return id; }
    // True when the change was applied in place.
    bool applyLiveState(const DrawerItem &item);

protected:
    void createStatusAndDot();

    QLabel *statusLabel;
    UpdateDot *updateDot;

private:
    QString id;
};

// List row, 152 px: icon (52 px), name, status line under it, chevron, dot.
class AppRow : public ItemWidget {
public:
    AppRow(const DrawerItem &item, QWidget *parent);
};

// Grid tile: outlined square with the mark at the top, status line, name and
// the dot in the top-right corner.
class AppTile : public ItemWidget {
public:
    AppTile(const DrawerItem &item, int side, QWidget *parent);

protected:
    void paintEvent(QPaintEvent *event);
    void resizeEvent(QResizeEvent *event);
};

// Small-caps section label ("INSTALLED", "SYSTEM").
QLabel *createSectionLabel(const QString &title, QWidget *parent);

#endif
