#ifndef NICKELDRAWER_DEBUG_LOG_H
#define NICKELDRAWER_DEBUG_LOG_H

// Opt-in step log on the user partition, for crashes that take the RAM-only
// syslog down with them. Shared by every mod (each compiles its own copy and
// passes its own folder, so two mods never share a log file).
//
// On only when the mod's flag file exists at boot (checked once by
// enableIfFlagged() in init()). Every line is also sent to nh_log. Each write
// opens, appends, fsyncs and closes the mod's log file, so no file is ever
// held open on /mnt/onboard and the line survives a Nickel crash right after
// it. The file is restarted when it would grow past 256 KB.

namespace DebugLog {

// Reads the flag file once; modName prefixes every line (e.g. "nddrawer"). The paths are copied.
void enableIfFlagged(const char *modName, const char *flagPath, const char *logPath);
bool isEnabled();

// printf-style; one line per call (a trailing newline is added).
void write(const char *format, ...) __attribute__((format(printf, 1, 2)));

} // namespace DebugLog

#endif
