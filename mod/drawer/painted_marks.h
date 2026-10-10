#ifndef NICKELDRAWER_PAINTED_MARKS_H
#define NICKELDRAWER_PAINTED_MARKS_H

#include <QPixmap>
#include <QWidget>

#include "press_area.h"

// The drawer's small marks and icon buttons. Icons are never drawn here: they
// come from system_icon.h (Nickel's own images, else installed Tabler PNGs).
// Black on white only (e-ink).

// The black update dot (docs/spec/ui-kit.md §5.1, §5.2).
class UpdateDot : public QWidget {
public:
    explicit UpdateDot(QWidget *parent);

protected:
    void paintEvent(QPaintEvent *event);
};

// Header view toggle: the list or grid icon (Tabler; Nickel has none); the
// active one is underlined (state is never shown by colour alone). Without the
// icon installed hasIcon() is false and the owner hides the toggle.
class ViewToggleIcon : public PressArea {
public:
    enum Shape { SHAPE_LIST, SHAPE_GRID };

    ViewToggleIcon(Shape shape, QWidget *parent);
    void setActive(bool isActive);
    bool hasIcon() const;

protected:
    void paintEvent(QPaintEvent *event);

private:
    Shape shape;
    bool isActive;
};

// A 120 px icon button at the right of an app screen header (ui.json header
// "actions"): Nickel's download (import), gear (settings), reload (refresh) or
// plus (add) icon, pressed: Nickel's "_hit" image (white on a black disc).
// With a Tabler icon the press inverts it in a circle; with no icon at all the
// button shows the action's label instead.
class HeaderIconButton : public PressArea {
public:
    enum Icon { ICON_IMPORT, ICON_SETTINGS, ICON_REFRESH, ICON_ADD };

    // False for a name the kit does not define.
    static bool iconNamed(const QString &name, Icon *icon);

    HeaderIconButton(Icon icon, const QString &label, QWidget *parent);

protected:
    void paintEvent(QPaintEvent *event);

private:
    Icon icon;
    QString label; // shown only when no icon is available
};

// A system icon (system_icon.h) centred at its own size in a side x side box, or fallbackText
// in the row font when the icon is not installed (the chevron of a drawer row, the pager arrows).
class IconMark : public QWidget {
public:
    IconMark(int iconId, int side, const QString &fallbackText, QWidget *parent);

protected:
    void paintEvent(QPaintEvent *event);

private:
    int iconId;
    QString fallbackText;
};

// An app's outline icon (from its icon file) or, without one, the generic
// "apps" icon (system_icon.h); nothing when neither is readable. Reads the
// icon file once when built. Icons are never scaled when a file of the wanted size sits beside
// the app's icon: "icon.png" is also looked up as "icon-<side>.png" (the 64 px row icon).
class AppMark : public QWidget {
public:
    AppMark(const QString &iconPath, int side, QWidget *parent);

protected:
    void paintEvent(QPaintEvent *event);

private:
    QPixmap icon;
};

#endif
