#include "helper_flags.h"

#include <unistd.h>

namespace HelperFlags {

bool hasFlag(const char *path)
{
    return access(path, F_OK) == 0;
}

ButtonState readButtonState(const char *offFlagPath, const char *folderPath)
{
    if (hasFlag(offFlagPath)) {
        return BUTTON_OFF;
    }
    // No off flag counts as "on" only while the folder it would live in is visible; with the
    // partition unmounted every flag looks absent.
    return hasFlag(folderPath) ? BUTTON_ON : BUTTON_UNKNOWN;
}

ButtonState readButtonState()
{
    return readButtonState(OFF_PATH, FOLDER);
}

} // namespace HelperFlags
