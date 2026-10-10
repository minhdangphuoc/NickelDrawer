#ifndef NICKELDRAWER_TWEAKS_MODEL_H
#define NICKELDRAWER_TWEAKS_MODEL_H

// Plain data of the Tweaks replies (docs/spec/tweaks.md v1), as Qt strings.
// All text is shown as received; appsctl already localized it.

#include <QList>
#include <QString>

#include "tweaks_logic.h"

struct TweakItem {
    QString id;
    QString label;
    TweaksLogic::Kind kind = TweaksLogic::KIND_ACTION;
    bool isOn = false;   // toggle
    QString infoText;    // info
    QString confirm;     // action: asked first when not empty

    bool isTappable() const { return kind != TweaksLogic::KIND_INFO; }
};

// Reply of `drawer tweaks`.
struct TweaksModel {
    bool isUsable = false; // false: failure holds the localized reason
    QString failure;
    QList<TweakItem> items;
};

// Reply of `drawer tweak <id>`.
struct TweakResult {
    bool isOk = false;
    QString toast;
    QString error; // ok:false, or the call itself failed (localized)
    bool hasValue = false;
    bool value = false;
    TweaksLogic::NickelAction nickel = TweaksLogic::NICKEL_NONE;
};

#endif
