#ifndef NICKELDRAWER_MODEL_PARSER_H
#define NICKELDRAWER_MODEL_PARSER_H

#include <QByteArray>
#include <QString>

#include "drawer_model.h"

// JSON (QJsonDocument) to the structs in drawer_model.h. Each parse returns
// false and a localized fallback message when the reply is unusable.
namespace ModelParser {

bool parseModel(const QByteArray &json, DrawerModel *model, QString *failure);
bool parseActionResult(const QByteArray &json, ActionResult *result, QString *failure);
bool parseAppInfo(const QByteArray &json, AppInfo *info, QString *failure);

// Message of an exit-2 reply {"error":{"code","message"}}; empty if absent.
QString errorMessage(const QByteArray &json);

// A model with one problem row; used when appsctl fails or is too new.
DrawerModel errorModel(const QString &message);

} // namespace ModelParser

#endif
