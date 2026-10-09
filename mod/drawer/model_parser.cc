#include "model_parser.h"

#include <QJsonArray>
#include <QJsonDocument>
#include <QJsonObject>
#include <QJsonValue>

#include <NickelHook.h>

#include "drawer_logic.h"
#include "fallback_strings.h"

static const char *const KIND_BACKGROUND = "background";
static const char *const PROBLEM_ROW_ID = "_problem";

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
    QJsonArray actions = object.value(QLatin1String("actions")).toArray();
    for (int i = 0; i < actions.size(); i++) {
        item.actions.append(parseAction(actions.at(i).toObject()));
    }
    return item;
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
