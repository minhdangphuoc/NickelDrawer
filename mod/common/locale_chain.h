#ifndef NICKELDRAWER_LOCALE_CHAIN_H
#define NICKELDRAWER_LOCALE_CHAIN_H

// Which locale files to try for a Nickel CurrentLocale, shared by the mods
// (each compiles its own copy). Pure: no Qt, no C++ standard library.
namespace LocaleChain {

// Longest file name in the chain, NUL included.
static const int LOCALE_NAME_SIZE = 16;

// Locale files to try, best first: the exact locale ("pt_BR"), its language ("pt"), then "en".
// A locale with characters other than letters, '_' and '-' (a path, say) gives only "en".
// Each name is written into names[i]; the number of names is returned (at most maxNames).
int fileNames(const char *locale, char names[][LOCALE_NAME_SIZE], int maxNames);

} // namespace LocaleChain

#endif
