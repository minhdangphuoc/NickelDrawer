#ifndef EXTRAS_TWEAKS_PARSER_H
#define EXTRAS_TWEAKS_PARSER_H

#include <QByteArray>
#include <QString>

#include "tweaks_model.h"

// The Tweaks replies (TweaksLogic, Qt-free and unit-tested) into the Qt
// structs of tweaks_model.h. Each parse returns false and a localized
// fallback message when the reply is unusable.
namespace TweaksParser {

bool parseTweaks(const QByteArray &json, TweaksModel *model, QString *failure);
// Also reads extras-tweaks' exit-2 replies: {"v":1,"ok":false,"error":"…"} and
// a usage error {"error":{"code","message"}} (result.error, returns true).
bool parseTweakResult(const QByteArray &json, TweakResult *result, QString *failure);

} // namespace TweaksParser

#endif
