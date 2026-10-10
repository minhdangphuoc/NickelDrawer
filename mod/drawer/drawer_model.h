#ifndef NICKELDRAWER_DRAWER_MODEL_H
#define NICKELDRAWER_DRAWER_MODEL_H

// Plain data parsed from appsctl replies (docs/spec/drawer-protocol.md v1).
// All text is shown as received; appsctl already localized it.

#include <QList>
#include <QString>
#include <QStringList>

struct DrawerAction {
    QString id;
    QString label;
    bool isBackground = false; // kind "background": start, then toast
};

struct DrawerItem {
    QString id;
    QString name;
    QString status;
    QString iconPath; // empty when the app has no icon
    bool hasUpdate = false; // "badge"
    bool isRunning = false;
    bool isProblem = false; // synthetic row for errors[] or a failed appsctl call
    bool hasScreen = false; // "screen": the app has a ui.json screen
    QList<DrawerAction> actions; // the first one is the primary (tap) action

    bool hasPrimaryAction() const { return !actions.isEmpty(); }
    bool isHostItem() const { return id.startsWith(QLatin1Char('_')); }
};

// Host row the drawer adds under an appsctl error; tapping it reloads the model.
static const char *const RETRY_ROW_ID = "_retry";

struct DrawerSection {
    QString title;
    QList<DrawerItem> items;
};

struct DrawerModel {
    QString title;
    QString hint;
    QList<DrawerSection> sections;
    bool isUsable = false; // false: the sections hold only an error row
};

// A dialog declared on an app screen, or a button's confirm question.
struct DeclaredDialog {
    QString title;
    QString text;
    QString ok;
    QString cancel; // empty: the drawer's own "Cancel"
    QString action; // run as a screen event when OK is tapped; may be empty
};

// Reply of `drawer run`, `drawer event` and `drawer remove`.
struct ActionResult {
    bool isOk = false;
    bool needsWifi = false;
    QString text;
    QString rescan; // "none", "quick", "full" or empty
    QString toast;  // drawer event: a toast declared on the screen
    bool hasDialog = false;
    DeclaredDialog dialog; // drawer event: a dialog declared on the screen
};

// One tap on an app screen (`drawer event`).
struct ScreenEvent {
    QString actionId;
    QString value;
    bool hasValue = false; // false: no --value, NICKELDRAWER_VALUE stays unset
    QString screenId;      // lets the script open the screen's declared toasts and dialogs
    // List row menu event: actionId is the menu item's id, value the row's primary
    // text; sent as `--list <bind> --index <n>` (docs/spec/drawer-protocol.md).
    QString listBind;
    int listIndex = -1;
};

// Reply of `drawer info`.
struct AppInfo {
    QString name;
    QString version;
    QString description;
    QStringList reads;
    QStringList writes;
    bool usesNetwork = false;
    QString path;
};

#endif
