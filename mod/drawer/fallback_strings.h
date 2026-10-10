#ifndef NICKELDRAWER_FALLBACK_STRINGS_H
#define NICKELDRAWER_FALLBACK_STRINGS_H

#include <QString>

// Strings of the drawer itself ("mod.*" keys of host/locales/<lang>.json). They are read from
// /mnt/onboard/.adds/koboapps/locales/<locale>.json (exact locale, language, en) when the drawer
// opens, so a new language needs no rebuild. A compiled-in English and Vietnamese table is the
// last resort for a key or a file that is missing. App text always comes from appsctl already
// localized.
namespace FallbackStrings {

enum Key {
    APPS_TITLE,
    LOADING,
    ERROR_TITLE,
    APPSCTL_MISSING,
    APPSCTL_TIMEOUT,
    APPSCTL_FAILED,
    APPSCTL_BAD_REPLY,
    UPDATE_NICKELDRAWER,
    TRY_AGAIN,
    PROBLEMS_SECTION,
    EMPTY_TITLE,
    EMPTY_HINT,
    FIND_APPS,
    LONG_PRESS_HINT,
    MENU_INFO,
    MENU_REMOVE,
    REMOVE_BUTTON,
    CANCEL,
    CLOSE,
    REMOVE_CONFIRM_TEXT,
    WIFI_OFF_TEXT,
    WIFI_TURN_ON,
    WIFI_FAILED,
    INFO_VERSION,
    INFO_READS,
    INFO_WRITES,
    INFO_NETWORK,
    INFO_FOLDER,
    YES,
    NO,
    UPDATES_COUNT_ONE,
    UPDATES_COUNT_MANY,
    STATUS_LINE,
    SUMMARY_SEPARATOR,
    DRAWER_UNAVAILABLE,
    SCREEN_ON,
    SCREEN_OFF,
    SCREEN_BACK,
    SCREEN_CANT_OPEN,
    SCREEN_EDIT_ON_PHONE,
    SCREEN_KEYBOARD_GO,
    ACTION_FAILED,
    TWEAKS_TITLE,
    TWEAK_CONFIRM_OK,
    NICKEL_ACTION_UNAVAILABLE,
    UNSUPPORTED_FIRMWARE,
    UNSUPPORTED_MODEL,
    QT6_UPGRADE_PENDING,
    SCREEN_PENDING,
    KEY_COUNT
};

// Reads [ApplicationPreferences] CurrentLocale from Nickel's eReader.conf
// (opened, read, closed by QSettings; never written) and the locale files. Call when the
// drawer or the More row is built, not in init().
void reloadLocale();

QString text(Key key);

} // namespace FallbackStrings

#endif
