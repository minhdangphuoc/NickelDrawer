#include "model_parser.h"

#include <QJsonArray>
#include <QJsonDocument>
#include <QJsonObject>
#include <QJsonValue>

#include <NickelHook.h>

#include "drawer_logic.h"
#include "fallback_strings.h"
#include "row_menu_logic.h"

static const char *const KIND_BACKGROUND = "background";
static const char *const PROBLEM_ROW_ID = "_problem";
static const int MAX_HEADER_ICONS = 2;

static bool readObject(const QByteArray &json, QJsonObject *object)
{
    QJsonParseError error;
    QJsonDocument document = QJsonDocument::fromJson(json, &error);
    if (error.error != QJsonParseError::NoError || !document.isObject()) {
        nh_log("parser: bad JSON (%s) at offset %d", qPrintable(error.errorString()), error.offset);
        return false;
    }
    *object = document.object();
    return true;
}

static QString stringField(const QJsonObject &object, const char *key)
{
    return object.value(QLatin1String(key)).toString();
}

static bool boolField(const QJsonObject &object, const char *key)
{
    return object.value(QLatin1String(key)).toBool(false);
}

static QStringList stringListField(const QJsonObject &object, const char *key)
{
    QStringList values;
    QJsonArray array = object.value(QLatin1String(key)).toArray();
    for (int i = 0; i < array.size(); i++) {
        values.append(array.at(i).toString());
    }
    return values;
}

// Checks "v"; on failure fills the localized reason.
static bool hasSupportedVersion(const QJsonObject &object, QString *failure)
{
    QJsonValue version = object.value(QLatin1String("v"));
    DrawerLogic::ProtocolCheck check = DrawerLogic::checkProtocolVersion(version.isDouble(), version.toInt());
    if (check == DrawerLogic::PROTOCOL_OK) {
        return true;
    }
    bool isTooNew = check == DrawerLogic::PROTOCOL_TOO_NEW;
    nh_log("parser: protocol version %d %s", version.toInt(), isTooNew ? "too new" : "missing");
    *failure = FallbackStrings::text(isTooNew ? FallbackStrings::UPDATE_NICKELDRAWER : FallbackStrings::APPSCTL_BAD_REPLY);
    return false;
}

static DrawerAction parseAction(const QJsonObject &object)
{
    DrawerAction action;
    action.id = stringField(object, "id");
    action.label = stringField(object, "label");
    action.isBackground = stringField(object, "kind") == QLatin1String(KIND_BACKGROUND);
    return action;
}

static DrawerItem problemItem(const QString &name, const QString &message)
{
    DrawerItem item;
    item.id = QString::fromLatin1(PROBLEM_ROW_ID);
    item.name = name;
    item.status = message;
    item.hasUpdate = false;
    item.isRunning = false;
    item.isProblem = true;
    return item;
}

static DrawerItem parseItem(const QJsonObject &object)
{
    DrawerItem item;
    item.id = stringField(object, "id");
    item.name = stringField(object, "name");
    item.status = stringField(object, "status");
    item.iconPath = stringField(object, "icon");
    item.hasUpdate = boolField(object, "badge");
    item.isRunning = boolField(object, "running");
    item.isProblem = false;
    item.hasScreen = boolField(object, "screen");
    QJsonArray actions = object.value(QLatin1String("actions")).toArray();
    for (int i = 0; i < actions.size(); i++) {
        item.actions.append(parseAction(actions.at(i).toObject()));
    }
    return item;
}

static DeclaredDialog parseDeclaredDialog(const QJsonObject &object)
{
    DeclaredDialog dialog;
    dialog.title = stringField(object, "title");
    dialog.text = stringField(object, "text");
    dialog.ok = stringField(object, "ok");
    dialog.cancel = stringField(object, "cancel");
    dialog.action = stringField(object, "action");
    return dialog;
}

// [[a, b], ...] as pairs; malformed entries are skipped.
static QList<TextPair> pairListField(const QJsonObject &object, const char *key)
{
    QList<TextPair> pairs;
    QJsonArray array = object.value(QLatin1String(key)).toArray();
    for (int i = 0; i < array.size(); i++) {
        QJsonArray pair = array.at(i).toArray();
        if (pair.size() == 2) {
            pairs.append(TextPair(pair.at(0).toString(), pair.at(1).toString()));
        }
    }
    return pairs;
}

static QList<PictureTile> parseTiles(const QJsonArray &array)
{
    QList<PictureTile> tiles;
    for (int i = 0; i < array.size(); i++) {
        QJsonObject object = array.at(i).toObject();
        PictureTile tile;
        tile.id = stringField(object, "id");
        if (tile.id.isEmpty()) {
            nh_log("parser: picture without id skipped");
            continue;
        }
        tile.thumb = stringField(object, "thumb");
        tile.image = stringField(object, "image");
        tile.label = stringField(object, "label");
        tile.info = stringField(object, "info");
        tile.isCurrent = boolField(object, "current");
        tiles.append(tile);
    }
    return tiles;
}

// confirm: an object (declared dialog) or `true` (generic destructive question).
static void parseMenuConfirm(const QJsonValue &confirm, TileMenuItem *item)
{
    item->hasConfirm = confirm.isObject() || (confirm.isBool() && confirm.toBool());
    if (confirm.isObject()) {
        item->confirm = parseDeclaredDialog(confirm.toObject());
    }
}

// Gallery menus (action / push / info, no cap) and list-row menus (an id, at
// most RowMenuLogic::MAX_LIST_ITEMS) share the entry format; the rules for
// keeping an entry are RowMenuLogic's.
static QList<TileMenuItem> parseMenu(const QJsonArray &array, bool isListMenu)
{
    int maxItems = isListMenu ? RowMenuLogic::MAX_LIST_ITEMS : RowMenuLogic::NO_LIMIT;
    QList<TileMenuItem> menu;
    for (int i = 0; i < array.size(); i++) {
        QJsonObject object = array.at(i).toObject();
        TileMenuItem item;
        item.id = isListMenu ? stringField(object, "id") : QString();
        item.label = stringField(object, "label");
        item.action = stringField(object, "action");
        item.push = stringField(object, "push");
        item.isInfo = boolField(object, "info");
        parseMenuConfirm(object.value(QLatin1String("confirm")), &item);
        bool hasTarget = !item.id.isEmpty() || !item.action.isEmpty() || !item.push.isEmpty() || item.isInfo;
        QByteArray label = item.label.toUtf8();
        RowMenuLogic::EntryVerdict verdict = RowMenuLogic::judgeEntry(label.constData(), hasTarget, menu.size(), maxItems);
        if (verdict != RowMenuLogic::ENTRY_KEPT) {
            nh_log("parser: %s menu entry skipped (%s)", isListMenu ? "list" : "gallery",
                   verdict == RowMenuLogic::ENTRY_NO_LABEL    ? "no label"
                   : verdict == RowMenuLogic::ENTRY_NO_TARGET ? "no target"
                                                              : "more than 6 entries");
            continue;
        }
        menu.append(item);
    }
    return menu;
}

static BarButton parseBarButton(const QJsonObject &object)
{
    BarButton button;
    button.label = stringField(object, "label");
    button.labelCurrent = stringField(object, "label_current");
    button.action = stringField(object, "action");
    QJsonValue confirm = object.value(QLatin1String("confirm"));
    button.hasConfirm = confirm.isObject();
    if (button.hasConfirm) {
        button.confirm = parseDeclaredDialog(confirm.toObject());
    }
    return button;
}

static void parsePictureFields(const QJsonObject &object, ScreenComponent *component)
{
    component->gridColumns = object.value(QLatin1String("columns")).toInt(4);
    component->gridRows = object.value(QLatin1String("rows")).toInt(3);
    component->pending = object.value(QLatin1String("pending")).toInt(0);
    component->tiles = parseTiles(object.value(QLatin1String("tiles")).toArray());
    QJsonObject tap = object.value(QLatin1String("tap")).toObject();
    if (!tap.isEmpty()) { // gallery: {push|action}; list: a string (handled by the caller)
        component->tapPush = stringField(tap, "push");
        component->tapAction = stringField(tap, "action");
    }
    component->menu = parseMenu(object.value(QLatin1String("menu")).toArray(), false);
    QJsonObject empty = object.value(QLatin1String("empty")).toObject();
    component->galleryEmpty.icon = stringField(empty, "icon");
    component->galleryEmpty.title = stringField(empty, "title");
    component->galleryEmpty.text = stringField(empty, "text");
    component->galleryEmpty.button = stringField(empty, "button");
    component->galleryEmpty.action = stringField(empty, "action");
    component->currentStatus = stringField(object, "current_status");
    component->primary = parseBarButton(object.value(QLatin1String("primary")).toObject());
    component->secondary = parseBarButton(object.value(QLatin1String("secondary")).toObject());
}

static ScreenComponent parseComponent(const QJsonObject &object)
{
    ScreenComponent component;
    component.type = stringField(object, "type");
    component.text = stringField(object, "text");
    component.style = stringField(object, "style");
    component.align = stringField(object, "align");
    component.title = stringField(object, "title");
    component.detail = stringField(object, "detail");
    component.caption = stringField(object, "caption");
    component.action = stringField(object, "action");
    component.push = stringField(object, "push");
    component.bind = stringField(object, "bind");
    QJsonValue value = object.value(QLatin1String("value"));
    component.isOn = value.toBool(false);
    component.value = value.isString() ? value.toString() : QString();
    component.isEnabled = object.value(QLatin1String("enabled")).toBool(true);
    component.options = pairListField(object, "options");
    component.placeholder = stringField(object, "placeholder");
    component.keyboard = stringField(object, "keyboard");
    component.isSecret = boolField(object, "secret");
    component.digits = object.value(QLatin1String("digits")).toInt(0);
    component.rows = pairListField(object, "rows");
    component.url = stringField(object, "url");
    component.code = stringField(object, "code");
    component.hint = stringField(object, "hint");
    component.expires = stringField(object, "expires");
    component.label = stringField(object, "label");
    component.done = stringField(object, "done");
    component.total = stringField(object, "total");
    QJsonValue confirm = object.value(QLatin1String("confirm"));
    component.hasConfirm = confirm.isObject();
    if (component.hasConfirm) {
        component.confirm = parseDeclaredDialog(confirm.toObject());
    }
    component.isAsync = boolField(object, "async");
    component.emptyText = stringField(object, "empty_text");
    component.tap = object.value(QLatin1String("tap")).toString();
    component.items = pairListField(object, "items");
    component.button = stringField(object, "button");
    component.optionDetails = stringListField(object, "option_details");
    if (component.type == QLatin1String("gallery") || component.type == QLatin1String("viewer")) {
        parsePictureFields(object, &component);
    }
    if (component.type == QLatin1String("list")) {
        component.menu = parseMenu(object.value(QLatin1String("menu")).toArray(), true);
    }
    return component;
}

static QList<ScreenComponent> parseComponents(const QJsonArray &array)
{
    QList<ScreenComponent> components;
    for (int i = 0; i < array.size(); i++) {
        ScreenComponent component = parseComponent(array.at(i).toObject());
        if (component.type.isEmpty()) {
            nh_log("parser: screen component without type skipped");
            continue;
        }
        components.append(component);
    }
    return components;
}

static DrawerSection parseSection(const QJsonObject &object)
{
    DrawerSection section;
    section.title = stringField(object, "title");
    QJsonArray items = object.value(QLatin1String("items")).toArray();
    for (int i = 0; i < items.size(); i++) {
        DrawerItem item = parseItem(items.at(i).toObject());
        if (item.id.isEmpty()) {
            nh_log("parser: item without id skipped");
            continue;
        }
        section.items.append(item);
    }
    return section;
}

// errors[].message is English and shown as-is (drawer-protocol.md).
static DrawerSection parseProblems(const QJsonArray &errors)
{
    DrawerSection section;
    section.title = FallbackStrings::text(FallbackStrings::PROBLEMS_SECTION);
    for (int i = 0; i < errors.size(); i++) {
        QJsonObject problem = errors.at(i).toObject();
        DrawerItem item = problemItem(stringField(problem, "app"), stringField(problem, "message"));
        item.id = item.name; // the app id: a broken app can still be removed
        section.items.append(item);
    }
    return section;
}

namespace ModelParser {

bool parseModel(const QByteArray &json, DrawerModel *model, QString *failure)
{
    QJsonObject object;
    if (!readObject(json, &object)) {
        *failure = FallbackStrings::text(FallbackStrings::APPSCTL_BAD_REPLY);
        return false;
    }
    if (!hasSupportedVersion(object, failure)) {
        return false;
    }
    model->title = stringField(object, "title");
    model->hint = stringField(object, "hint");
    model->sections.clear();
    model->isUsable = true;
    QJsonArray sections = object.value(QLatin1String("sections")).toArray();
    for (int i = 0; i < sections.size(); i++) {
        DrawerSection section = parseSection(sections.at(i).toObject());
        if (!section.items.isEmpty()) {
            model->sections.append(section);
        }
    }
    QJsonArray errors = object.value(QLatin1String("errors")).toArray();
    if (!errors.isEmpty()) {
        model->sections.append(parseProblems(errors));
    }
    return true;
}

bool parseActionResult(const QByteArray &json, ActionResult *result, QString *failure)
{
    QJsonObject object;
    if (!readObject(json, &object)) {
        *failure = FallbackStrings::text(FallbackStrings::APPSCTL_BAD_REPLY);
        return false;
    }
    if (!hasSupportedVersion(object, failure)) {
        return false;
    }
    result->isOk = boolField(object, "ok");
    result->needsWifi = boolField(object, "needs_wifi");
    result->text = stringField(object, "text");
    result->rescan = stringField(object, "rescan");
    result->toast = stringField(object, "toast");
    QJsonValue dialog = object.value(QLatin1String("dialog"));
    result->hasDialog = dialog.isObject();
    if (result->hasDialog) {
        result->dialog = parseDeclaredDialog(dialog.toObject());
    }
    return true;
}

bool parseAppInfo(const QByteArray &json, AppInfo *info, QString *failure)
{
    QJsonObject object;
    if (!readObject(json, &object)) {
        *failure = FallbackStrings::text(FallbackStrings::APPSCTL_BAD_REPLY);
        return false;
    }
    if (!hasSupportedVersion(object, failure)) {
        return false;
    }
    info->name = stringField(object, "name");
    info->version = stringField(object, "version");
    info->description = stringField(object, "description");
    QJsonObject storage = object.value(QLatin1String("storage")).toObject();
    info->reads = stringListField(storage, "reads");
    info->writes = stringListField(storage, "writes");
    info->usesNetwork = boolField(object, "network");
    info->path = stringField(object, "path");
    return true;
}

bool parseScreen(const QByteArray &json, ScreenModel *screen, QString *failure)
{
    QJsonObject object;
    if (!readObject(json, &object)) {
        *failure = FallbackStrings::text(FallbackStrings::APPSCTL_BAD_REPLY);
        return false;
    }
    if (!hasSupportedVersion(object, failure)) {
        return false;
    }
    screen->app = stringField(object, "app");
    screen->screen = stringField(object, "screen");
    screen->title = stringField(object, "title");
    screen->status = stringField(object, "status");
    QJsonObject header = object.value(QLatin1String("action")).toObject();
    screen->headerLabel = stringField(header, "label");
    screen->headerPush = stringField(header, "push");
    screen->headerAction = stringField(header, "action");
    screen->body = parseComponents(object.value(QLatin1String("body")).toArray());
    screen->footer = parseComponents(object.value(QLatin1String("footer")).toArray());
    QJsonObject dialogs = object.value(QLatin1String("dialogs")).toObject();
    for (QJsonObject::const_iterator it = dialogs.constBegin(); it != dialogs.constEnd(); ++it) {
        screen->dialogs.insert(it.key(), parseDeclaredDialog(it.value().toObject()));
    }
    QJsonObject toasts = object.value(QLatin1String("toasts")).toObject();
    for (QJsonObject::const_iterator it = toasts.constBegin(); it != toasts.constEnd(); ++it) {
        screen->toasts.insert(it.key(), it.value().toString());
    }
    screen->isLive = boolField(object, "live");
    screen->isStale = boolField(object, "stale");
    screen->value = stringField(object, "value");
    QJsonArray icons = object.value(QLatin1String("actions")).toArray();
    for (int i = 0; i < icons.size() && i < MAX_HEADER_ICONS; i++) {
        QJsonObject icon = icons.at(i).toObject();
        HeaderIconAction headerIcon;
        headerIcon.icon = stringField(icon, "icon");
        headerIcon.label = stringField(icon, "label");
        headerIcon.push = stringField(icon, "push");
        headerIcon.action = stringField(icon, "action");
        screen->headerActions.append(headerIcon);
    }
    return true;
}

QString errorMessage(const QByteArray &json)
{
    QJsonObject object;
    if (!readObject(json, &object)) {
        return QString();
    }
    QJsonObject error = object.value(QLatin1String("error")).toObject();
    nh_log("parser: appsctl error %s", qPrintable(stringField(error, "code")));
    return stringField(error, "message");
}

DrawerModel errorModel(const QString &message)
{
    DrawerModel model;
    model.title = FallbackStrings::text(FallbackStrings::APPS_TITLE);
    model.isUsable = false;
    DrawerSection section;
    section.title = FallbackStrings::text(FallbackStrings::PROBLEMS_SECTION);
    section.items.append(problemItem(FallbackStrings::text(FallbackStrings::ERROR_TITLE), message));
    DrawerItem retry = problemItem(FallbackStrings::text(FallbackStrings::TRY_AGAIN), QString());
    retry.id = QString::fromLatin1(RETRY_ROW_ID);
    retry.isProblem = false;
    section.items.append(retry);
    model.sections.append(section);
    return model;
}

} // namespace ModelParser
