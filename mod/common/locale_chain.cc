#include "locale_chain.h"

#include <string.h>

namespace LocaleChain {

static bool isLocaleCharacter(char letter)
{
    return (letter >= 'a' && letter <= 'z') || (letter >= 'A' && letter <= 'Z') || letter == '_' || letter == '-';
}

static int addName(char names[][LOCALE_NAME_SIZE], int count, int maxNames, const char *name, int length)
{
    if (length <= 0 || length >= LOCALE_NAME_SIZE || count >= maxNames) {
        return count;
    }
    for (int i = 0; i < count; i++) {
        if (strlen(names[i]) == (size_t)length && strncmp(names[i], name, length) == 0) {
            return count;
        }
    }
    memcpy(names[count], name, length);
    names[count][length] = '\0';
    return count + 1;
}

int fileNames(const char *locale, char names[][LOCALE_NAME_SIZE], int maxNames)
{
    int count = 0;
    int length = locale ? (int)strlen(locale) : 0;
    bool isSafe = length > 0 && length < LOCALE_NAME_SIZE;
    int languageLength = length;
    for (int i = 0; i < length; i++) {
        isSafe = isSafe && isLocaleCharacter(locale[i]);
        if (languageLength == length && (locale[i] == '_' || locale[i] == '-')) {
            languageLength = i;
        }
    }
    if (isSafe) {
        count = addName(names, count, maxNames, locale, length);
        count = addName(names, count, maxNames, locale, languageLength);
    }
    return addName(names, count, maxNames, "en", 2);
}

} // namespace LocaleChain
