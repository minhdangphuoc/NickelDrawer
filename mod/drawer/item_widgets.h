#ifndef NICKELDRAWER_ITEM_WIDGETS_H
#define NICKELDRAWER_ITEM_WIDGETS_H

#include <QStringList>

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
    void createDot();

    QLabel *statusLabel;
    UpdateDot *updateDot;
    int statusWidth; // the one line the status may take; 0: no status line (grid cells)

private:
    QString id;
};

// List row, 152 px: icon (64 px, 1:1), name (row token), status line under it (caption, muted),
// chevron icon, dot. Name and status are one measured line each, elided to
// DrawerLogic::rowTextWidth, so nothing wraps or overflows the fixed row; a problem row's full
// message is in the Info dialog.
class AppRow : public ItemWidget {
public:
    AppRow(const DrawerItem &item, QWidget *parent);
};

// Grid cell (docs/spec/ui-kit.md §5.2): no border, the 96 px icon centred, the
// name centred under it (two lines at most, elided), the update dot at the
// icon's top-right. No status line. The caller places it with
// DrawerLogic::gridCellRect.
class AppTile : public ItemWidget {
public:
    AppTile(const DrawerItem &item, int width, int height, QWidget *parent);

protected:
    void paintEvent(QPaintEvent *event);

private:
    QStringList nameLines;
    int nameTop;
};

// Section label: the title uppercased (the host's strings are sentence case), then elided to the
// content width in the bold section font, over the black 2 px rule.
QLabel *createSectionLabel(const QString &title, QWidget *parent);

#endif
