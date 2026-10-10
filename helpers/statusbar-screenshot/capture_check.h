#ifndef STATUSBARSHOT_CAPTURE_CHECK_H
#define STATUSBARSHOT_CAPTURE_CHECK_H

// Did saveCurrentScreen() really write a new file? Nickel ignores the result of the PNG save and says
// nothing in USB mode, so the helper compares the file before and after with stat() and keeps no
// handle open on /mnt/onboard. No Qt, no libnickel, no C++ standard library (host-tested).

namespace CaptureCheck {

struct FileState {
    bool exists;
    long long size;
};

// stat() of path; exists is false when it cannot be seen.
FileState statFile(const char *path);

// A screenshot was saved when the file Nickel names now exists with content, and is a file that was
// not there before: either the name moved on (the counter counts up past files that exist) or the
// file did not exist at the first look. Same name and already there: Nickel wrote nothing (USB mode).
bool didSaveNewFile(bool isNameChanged, const FileState &before, const FileState &after);

} // namespace CaptureCheck

#endif
