#ifndef NICKELDRAWER_SCREEN_MODEL_H
#define NICKELDRAWER_SCREEN_MODEL_H

// Plain data of one app screen as `appsctl drawer screen` sends it
// (docs/spec/drawer-protocol.md): texts localized, bindings filled in.

#include <QList>
#include <QMap>
#include <QPair>
#include <QString>
#include <QStringList>

#include "drawer_model.h"

typedef QPair<QString, QString> TextPair;

// One picture of a gallery or viewer (docs/spec/wallpaper-gallery.md section 4).
// thumb / image are "" when the file is missing or was rejected by appsctl.
struct PictureTile {
    QString id;
    QString thumb;
    QString image;
    QString label;
    QString info;
    bool isCurrent = false;
};

// A long-press menu entry. Gallery: exactly one of action / push / isInfo.
// List row (ui-json.md): an id, sent back in the menu event. hasConfirm with an
// empty confirm (`"confirm": true`) asks the generic destructive question.
struct TileMenuItem {
    QString id;
    QString label;
    QString action;
    QString push;
    bool isInfo = false;
    bool hasConfirm = false;
    DeclaredDialog confirm;
};

// A button of the viewer's bottom bar.
struct BarButton {
    QString label;
    QString labelCurrent; // primary only: shown, disabled, when the picture is current
    QString action;
    bool hasConfirm = false;
    DeclaredDialog confirm;

    bool isDefined() const { return !label.isEmpty() && !action.isEmpty(); }
};

// The gallery's own empty state.
struct GalleryEmpty {
    QString icon;
    QString title;
    QString text;
    QString button;
    QString action;
};

// A header icon button: icon is import, settings, refresh or add.
struct HeaderIconAction {
    QString icon;
    QString label;
    QString push;
    QString action;
};

struct ScreenComponent {
    QString type; // the ui.json component key
    QString text;
    QString style;
    QString align;
    QString title;
    QString detail;
    QString caption;
    QString action;
    QString push; // row: screen pushed on tap, instead of an action
    QString bind;
    QString value;        // radio: the selected option id; field: the shown text
    bool isOn = false;    // toggle value
    bool isEnabled = true;
    QList<TextPair> options; // radio: id, label
    QString placeholder;
    QString keyboard;
    bool isSecret = false;
    int digits = 0;
    QList<TextPair> rows; // kv: key, value
    QString url;
    QString code;
    QString hint;
    QString expires;
    QString label;
    QString done;
    QString total;
    bool hasConfirm = false;
    DeclaredDialog confirm;
    bool isAsync = false;
    QString emptyText;
    QString tap;
    QList<TextPair> items; // list: primary, secondary
    QString button;        // empty: the label of its primary button
    QStringList optionDetails; // radio: parallel to options, sub-line under each option

    // gallery / viewer
    int gridColumns = 4;
    int gridRows = 3;
    int pending = 0; // gallery: dashed placeholder tiles after the items
    QList<PictureTile> tiles;
    QString tapPush;   // gallery: screen pushed with the tile id
    QString tapAction; // gallery: action run with the tile id
    QList<TileMenuItem> menu;
    GalleryEmpty galleryEmpty;
    QString currentStatus; // viewer: header status while the shown picture is current
    BarButton primary;
    BarButton secondary;

    bool isPrimaryButton() const { return type == QLatin1String("button") && style == QLatin1String("primary"); }
};

struct ScreenModel {
    QString app;
    QString screen;
    QString title;
    QString status;
    QString headerLabel; // header action, empty when none
    QString headerPush;
    QString headerAction;
    QList<ScreenComponent> body;
    QList<ScreenComponent> footer;
    QMap<QString, DeclaredDialog> dialogs;
    QMap<QString, QString> toasts;
    bool isLive = false;
    bool isStale = false; // the app's state action failed: values are defaults, keep the previous screen
    QString value;        // the screen value the reply was made for (viewer: the picture id)
    QList<HeaderIconAction> headerActions;
};

#endif
