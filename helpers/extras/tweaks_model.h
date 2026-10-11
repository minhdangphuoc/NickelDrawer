#ifndef EXTRAS_TWEAKS_MODEL_H
#define EXTRAS_TWEAKS_MODEL_H

// Plain data of the Tweaks replies (helpers/extras/docs/tweaks.md v1), as Qt strings.
// All text is shown as received; extras-tweaks already localized it.

#include <QList>
#include <QString>

#include "tweaks_logic.h"

struct TweakItem {
    QString id;
    QString label;
    QString caption;     // optional muted line under the label
    TweaksLogic::Kind kind = TweaksLogic::KIND_ACTION;
    bool isOn = false;   // toggle
    QString infoText;    // info
    QString confirm;     // action: asked first when not empty

    bool isTappable() const { return kind != TweaksLogic::KIND_INFO; }
    // Info cells stack label and value; others stack label and caption when there is one.
    bool hasCaption() const { return kind != TweaksLogic::KIND_INFO && !caption.isEmpty(); }
};

// Reply of `extras-tweaks list`.
struct TweaksModel {
    bool isUsable = false; // false: failure holds the localized reason
    QString failure;
    QList<TweakItem> items;
};

// Reply of `extras-tweaks apply <id>`.
struct TweakResult {
    bool isOk = false;
    QString toast;
    QString error; // ok:false, or the call itself failed (localized)
    bool hasValue = false;
    bool value = false;
    TweaksLogic::NickelAction nickel = TweaksLogic::NICKEL_NONE;
};

#endif
