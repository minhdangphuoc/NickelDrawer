#include "device_state.h"

#include <stdio.h>
#include <string.h>
#include <unistd.h>

#include <QSettings>

PrototypeSwitches prototypeSwitches = {false, false, false};

static const char *const VERSION_FILE_PATH = "/mnt/onboard/.kobo/version";
static const int FIRMWARE_FIELD_INDEX = 2; // "serial,3.0.0,4.38.23828,..."
static const char *const NICKEL_CONF_PATH = "/mnt/onboard/.kobo/Kobo/Kobo eReader.conf";
static const char *const LOCALE_KEY = "ApplicationPreferences/CurrentLocale";

static const char *const SUPPORTED_FIRMWARE[] = {
    "4.38.23552", "4.38.23648", "4.38.23684", "4.38.23697", "4.38.23828",
};

namespace DeviceState {

void readFirmware(char *out, size_t outSize)
{
    snprintf(out, outSize, "unknown");

    char line[128];
    FILE *file = fopen(VERSION_FILE_PATH, "r");
    if (!file) {
        return;
    }
    char *got = fgets(line, sizeof(line), file);
    fclose(file); // never keep files open on /mnt/onboard
    if (!got) {
        return;
    }

    char *cursor = line;
    for (int field = 0; field < FIRMWARE_FIELD_INDEX; field++) {
        cursor = strchr(cursor, ',');
        if (!cursor) {
            return;
        }
        cursor++;
    }
    cursor[strcspn(cursor, ",\r\n")] = '\0';
    if (*cursor) {
        snprintf(out, outSize, "%s", cursor);
    }
}

bool isSupportedFirmware(const char *firmware)
{
    for (size_t i = 0; i < sizeof(SUPPORTED_FIRMWARE) / sizeof(SUPPORTED_FIRMWARE[0]); i++) {
        if (!strcmp(firmware, SUPPORTED_FIRMWARE[i])) {
            return true;
        }
    }
    return false;
}

bool hasFlagFile(const char *path)
{
    return access(path, F_OK) == 0;
}

QString appsLabel()
{
    // QSettings reads the whole file in its constructor and closes it; we never
    // set a value, so nothing is written back to Nickel's config.
    QSettings nickelConf(QString::fromUtf8(NICKEL_CONF_PATH), QSettings::IniFormat);
    QString locale = nickelConf.value(QString::fromLatin1(LOCALE_KEY)).toString();
    if (locale.startsWith(QLatin1String("vi"))) {
        return QString::fromUtf8("Ứng dụng");
    }
    return QStringLiteral("Apps");
}

} // namespace DeviceState
