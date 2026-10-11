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

// Indexed by HelperStrings::Key; the texts equal locales/en.json (checked by a Go test).
static const Entry TABLE[HelperStrings::KEY_COUNT] = {
    {"toast.saved", "Screenshot saved"},
    {"toast.failed", "Could not take screenshot"},
    {"toast.disabled", "Extras disabled: firmware not supported"},
    {"tweaks.title", "Tweaks"},
    {"tweaks.loading", "Loading…"},
    {"tweaks.state_on", "On"},
    {"tweaks.state_off", "Off"},
    {"tweaks.confirm_ok", "OK"},
    {"tweaks.cancel", "Cancel"},
    {"tweaks.unavailable", "Tweaks are not available (see the log)."},
    {"tweaks.helper_missing", "Part of Extras is missing. Reinstall it."},
    {"tweaks.helper_timeout", "Extras did not answer in time. Try again."},
    {"tweaks.helper_failed", "Extras hit a problem. Try again."},
    {"tweaks.helper_bad_reply", "Extras got an answer it cannot read. Try again."},
    {"tweaks.update_needed", "Update Extras to use this version."},
    {"tweaks.action_failed", "Couldn't do that. Try again."},
    {"tweaks.nickel_unavailable", "Not available on this firmware."},
};

static QString localeTexts[HelperStrings::KEY_COUNT]; // null: not in any locale file
static bool isLoaded = false;

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

// Fills the texts still missing from one locale file.
static void takeMissingTexts(const QJsonObject &texts)
{
    for (int key = 0; key < HelperStrings::KEY_COUNT; key++) {
        QJsonValue value = texts.value(QString::fromLatin1(TABLE[key].jsonKey));
        if (localeTexts[key].isNull() && value.isString() && !value.toString().isEmpty()) {
            localeTexts[key] = value.toString();
        }
    }
}

namespace HelperStrings {

void reloadLocale()
{
    for (int key = 0; key < KEY_COUNT; key++) {
        localeTexts[key] = QString();
    }
    QByteArray locale = currentLocale();
    char names[MAX_LOCALE_FILES][LocaleChain::LOCALE_NAME_SIZE];
    int count = LocaleChain::fileNames(locale.constData(), names, MAX_LOCALE_FILES);
    for (int i = 0; i < count; i++) {
        QString path = QString::fromLatin1(HelperFlags::LOCALES_DIRECTORY) + QLatin1Char('/') +
                       QString::fromLatin1(names[i]) + QStringLiteral(".json");
        QJsonObject texts;
        if (readLocaleFile(path, &texts)) {
            takeMissingTexts(texts);
        }
    }
    isLoaded = true;
}

QString text(Key key)
{
    if (key < 0 || key >= KEY_COUNT) {
        return QString();
    }
    if (!isLoaded) {
        reloadLocale();
    }
    if (!localeTexts[key].isNull()) {
        return localeTexts[key];
    }
    return QString::fromUtf8(TABLE[key].english);
}

} // namespace HelperStrings
