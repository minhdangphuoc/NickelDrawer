#ifndef STATUSBARSHOT_HELPER_STRINGS_H
#define STATUSBARSHOT_HELPER_STRINGS_H

#include <QString>

// The helper's three toast texts. English is the source language; translations are
// <helper folder>/locales/<lang>.json, picked by CurrentLocale from Kobo eReader.conf (exact locale,
// language, en), so a new language needs no rebuild. A compiled English table is the last resort for
// a key or a file that is missing.
namespace HelperStrings {

enum Key { SAVED, FAILED, DISABLED, KEY_COUNT };

// Re-reads the locale and the files; call before showing a text (the user may change language).
QString text(Key key);

} // namespace HelperStrings

#endif
