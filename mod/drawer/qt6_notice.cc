#include "qt6_notice.h"

#include <QSettings>
#include <QStringList>
#include <QVariant>

#include "debug_log.h"
#include "drawer_logic.h"

static const char *const NICKEL_CONF_PATH = "/mnt/onboard/.kobo/Kobo/Kobo eReader.conf";
static const char *const OFFERED_KEY = "ApplicationPreferences/AvailableQt6Upgrade";
static const char *const SKIPPED_KEY = "ApplicationPreferences/SkippedQt6UpgradeUrl";

// QSettings splits an unquoted value at commas into a string list, whose
// toString() is empty: join it back to the URL as written.
static QByteArray urlValue(const QSettings &settings, const char *key)
{
    QVariant value = settings.value(QString::fromLatin1(key));
    QString text = value.type() == QVariant::StringList ? value.toStringList().join(QLatin1String(",")) : value.toString();
    return text.trimmed().toLatin1();
}

namespace Qt6Notice {

bool isUpgradePending()
{
    QSettings nickelConf(QString::fromUtf8(NICKEL_CONF_PATH), QSettings::IniFormat);
    QByteArray offered = urlValue(nickelConf, OFFERED_KEY);
    QByteArray skipped = urlValue(nickelConf, SKIPPED_KEY);
    bool isPending = DrawerLogic::isQt6UpgradePending(offered.constData(), skipped.constData());
    DebugLog::write("qt6 notice: offer %s, pending %s", offered.isEmpty() ? "none" : "present", isPending ? "yes" : "no");
    return isPending;
}

} // namespace Qt6Notice
