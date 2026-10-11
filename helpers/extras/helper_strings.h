#ifndef EXTRAS_HELPER_STRINGS_H
#define EXTRAS_HELPER_STRINGS_H

#include <QString>

// The helper's texts (toasts, the Tweaks panel). English is the source language; translations are
// <helper folder>/locales/<lang>.json, picked by CurrentLocale from Kobo eReader.conf (exact locale,
// language, en), so a new language needs no rebuild. A compiled English table is the last resort for a
// key or a file that is missing; a test keeps it equal to locales/en.json. The Tweaks list itself is
// localized by the extras-tweaks binary from the same files.
namespace HelperStrings {

enum Key {
    SAVED,
    FAILED,
    DISABLED,
    TWEAKS_TITLE,
    LOADING,
    STATE_ON,
    STATE_OFF,
    CONFIRM_OK,
    CANCEL,
    TWEAKS_UNAVAILABLE,
    HELPER_MISSING,
    HELPER_TIMEOUT,
    HELPER_FAILED,
    HELPER_BAD_REPLY,
    UPDATE_NEEDED,
    ACTION_FAILED,
    NICKEL_UNAVAILABLE,
    KEY_COUNT
};

// Re-reads the locale and the locale files. Call when a screen is built or a toast is about to be
// shown (the user may have changed the language), never from init().
void reloadLocale();

// The text for the locale last loaded (loaded on first use).
QString text(Key key);

} // namespace HelperStrings

#endif
