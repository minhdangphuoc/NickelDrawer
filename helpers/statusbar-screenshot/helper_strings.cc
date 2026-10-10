#include "helper_strings.h"

#include <QFile>
#include <QJsonDocument>
#include <QJsonObject>
#include <QSettings>

#include <NickelHook.h>

#include "helper_flags.h"
#include "locale_chain.h"

static const char *const NICKEL_CONF_PATH = "/mnt/onboard/.kobo/Kobo/Kobo eReader.conf";
static const char *const LOCALE_KEY = "ApplicationPreferences/CurrentLocale";
static const int MAX_LOCALE_FILES = 3; // exact, language, en

struct Entry {
    const char *jsonKey;
    const char *english; // compiled fallback
};

// Indexed by HelperStrings::Key.
static const Entry TABLE[HelperStrings::KEY_COUNT] = {
    {"toast.saved", "Screenshot saved"},
    {"toast.failed", "Could not take screenshot"},
    {"toast.disabled", "Screenshot button disabled: firmware not supported"},
};

static QByteArray currentLocale()
{
    QSettings nickelConf(QString::fromUtf8(NICKEL_CONF_PATH), QSettings::IniFormat);
    return nickelConf.value(QString::fromLatin1(LOCALE_KEY)).toString().toLatin1();
}

// QFile reads the whole file and closes it: nothing stays open on /mnt/onboard.
static bool readLocaleFile(const QString &path, QJsonObject *texts)
{
    QFile file(path);
    if (!file.open(QIODevice::ReadOnly)) {
        return false;
    }
    QByteArray content = file.readAll();
    file.close();
    QJsonDocument document = QJsonDocument::fromJson(content);
    if (!document.isObject()) {
        nh_log("strings: %s is not a JSON object", qPrintable(path));
        return false;
    }
    *texts = document.object();
    return true;
}

namespace HelperStrings {

QString text(Key key)
{
    if (key < 0 || key >= KEY_COUNT) {
        return QString();
    }
    QByteArray locale = currentLocale();
    char names[MAX_LOCALE_FILES][LocaleChain::LOCALE_NAME_SIZE];
    int count = LocaleChain::fileNames(locale.constData(), names, MAX_LOCALE_FILES);
    for (int i = 0; i < count; i++) {
        QString path = QString::fromLatin1(HelperFlags::FOLDER) + QStringLiteral("/locales/") +
                       QString::fromLatin1(names[i]) + QStringLiteral(".json");
        QJsonObject texts;
        QString translated = readLocaleFile(path, &texts) ? texts.value(QString::fromLatin1(TABLE[key].jsonKey)).toString() : QString();
        if (!translated.isEmpty()) {
            return translated;
        }
    }
    return QString::fromUtf8(TABLE[key].english);
}

} // namespace HelperStrings
