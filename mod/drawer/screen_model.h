#ifndef NICKELDRAWER_SCREEN_MODEL_H
#define NICKELDRAWER_SCREEN_MODEL_H

// Plain data of one app screen as `appsctl drawer screen` sends it
// (docs/spec/drawer-protocol.md): texts localized, bindings filled in.

#include <QList>
#include <QMap>
#include <QPair>
#include <QString>

#include "drawer_model.h"

typedef QPair<QString, QString> TextPair;

struct ScreenComponent {
    QString type; // the ui.json component key
    QString text;
    QString style;
    QString align;
    QString title;
    QString detail;
    QString caption;
    QString action;
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
};

#endif
