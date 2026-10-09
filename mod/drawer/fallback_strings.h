#ifndef NICKELDRAWER_FALLBACK_STRINGS_H
#define NICKELDRAWER_FALLBACK_STRINGS_H

#include <QString>

// Host strings of the drawer itself (en + vi). App text always comes from
// appsctl already localized; these are only for the drawer's own chrome and
// for when appsctl fails.
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
    KEY_COUNT
};

// Reads [ApplicationPreferences] CurrentLocale from Nickel's eReader.conf
// (opened, read, closed by QSettings; never written). Call when the drawer or
// the More row is built, not in init().
void reloadLocale();

QString text(Key key);

} // namespace FallbackStrings

#endif
