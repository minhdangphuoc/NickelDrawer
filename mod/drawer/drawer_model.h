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

// Reply of `drawer run` and `drawer remove`.
struct ActionResult {
    bool isOk = false;
    bool needsWifi = false;
    QString text;
    QString rescan; // "none", "quick", "full" or empty
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
