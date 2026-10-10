#include "capture_check.h"

#include <sys/stat.h>

namespace CaptureCheck {

FileState statFile(const char *path)
{
    FileState state = {false, 0};
    struct stat info;
    if (stat(path, &info) == 0 && S_ISREG(info.st_mode)) {
        state.exists = true;
        state.size = info.st_size;
    }
    return state;
}

bool didSaveNewFile(bool isNameChanged, const FileState &before, const FileState &after)
{
    bool hasContent = after.exists && after.size > 0;
    return hasContent && (isNameChanged || !before.exists);
}

} // namespace CaptureCheck
