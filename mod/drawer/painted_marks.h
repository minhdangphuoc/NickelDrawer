#ifndef NICKELDRAWER_PAINTED_MARKS_H
#define NICKELDRAWER_PAINTED_MARKS_H

#include <QPixmap>
#include <QWidget>

#include "press_area.h"

// Small marks drawn with QPainter instead of font glyphs, so they look the
// same whatever fonts the device has. Black on white only (e-ink).

// The black update dot (docs/spec/ui-kit.md §5.1, §5.2).
class UpdateDot : public QWidget {
public:
    explicit UpdateDot(QWidget *parent);

protected:
    void paintEvent(QPaintEvent *event);
};

// Header view toggle: three bars (list) or 2 x 2 squares (grid); the active
// one is underlined (state is never shown by colour alone).
class ViewToggleIcon : public PressArea {
public:
    enum Shape { SHAPE_LIST, SHAPE_GRID };

    ViewToggleIcon(Shape shape, QWidget *parent);
    void setActive(bool isActive);

protected:
    void paintEvent(QPaintEvent *event);

private:
    void paintList(QPainter *painter, const QRect &area);
    void paintGrid(QPainter *painter, const QRect &area);

    Shape shape;
    bool isActive;
};

// A 120 px icon button at the right of an app screen header (ui.json header
// "actions"): tray-down (import), gear (settings), circular arrow (refresh) or
// plus (add). Pressed: Nickel's black circle with the icon in white.
class HeaderIconButton : public PressArea {
public:
    enum Icon { ICON_IMPORT, ICON_SETTINGS, ICON_REFRESH, ICON_ADD };

    // False for a name the kit does not define.
    static bool iconNamed(const QString &name, Icon *icon);

    HeaderIconButton(Icon icon, QWidget *parent);

protected:
    void paintEvent(QPaintEvent *event);

private:
    Icon icon;
};

// An app's outline icon (from its icon file) or, without one, its initial in
// an outlined square. Reads the icon file once when built.
class AppMark : public QWidget {
public:
    AppMark(const QString &iconPath, const QString &name, int side, QWidget *parent);

protected:
    void paintEvent(QPaintEvent *event);

private:
    QPixmap icon;
    QString initial;
};

#endif
