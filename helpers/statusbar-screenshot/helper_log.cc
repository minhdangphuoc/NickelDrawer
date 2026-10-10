#include "helper_log.h"

#include <stdarg.h>
#include <stdio.h>

#include <NickelHook.h>

#include "helper_flags.h"

namespace HelperLog {

static bool isDebugOn = false;

void enableIfFlagged()
{
    isDebugOn = HelperFlags::hasFlag(HelperFlags::DEBUG_PATH);
}

void debug(const char *format, ...)
{
    if (!isDebugOn) {
        return;
    }
    char line[256]; // nh_log truncates at 256 bytes anyway
    va_list arguments;
    va_start(arguments, format);
    vsnprintf(line, sizeof(line), format, arguments);
    va_end(arguments);
    nh_log("%s", line);
}

} // namespace HelperLog
