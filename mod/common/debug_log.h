#ifndef NICKELDRAWER_DEBUG_LOG_H
#define NICKELDRAWER_DEBUG_LOG_H

// Opt-in step log on the user partition, for crashes that take the RAM-only
// syslog down with them. Shared by mod/drawer-proto and mod/drawer.
//
// On only when the flag file /mnt/onboard/.adds/nickeldrawer/debug exists at
// boot (checked once by enableIfFlagged() in init()). Every line is also sent
// to nh_log. Each write opens, appends, fsyncs and closes
// /mnt/onboard/.adds/nickeldrawer/debug.log, so no file is ever held open on
// /mnt/onboard and the line survives a Nickel crash right after it. The file
// is restarted when it would grow past 64 KB.

namespace DebugLog {

static const char FLAG_PATH[] = "/mnt/onboard/.adds/nickeldrawer/debug";
static const char LOG_PATH[] = "/mnt/onboard/.adds/nickeldrawer/debug.log";

// Reads the flag file once; modName prefixes every line (e.g. "nddrawer").
void enableIfFlagged(const char *modName);
bool isEnabled();

// printf-style; one line per call (a trailing newline is added).
void write(const char *format, ...) __attribute__((format(printf, 1, 2)));

} // namespace DebugLog

#endif
