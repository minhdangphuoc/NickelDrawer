#ifndef STATUSBARSHOT_HELPER_LOG_H
#define STATUSBARSHOT_HELPER_LOG_H

// Verbose syslog lines, on only while the "debug" flag file existed at boot. Messages that matter
// always go through nh_log directly.
namespace HelperLog {

void enableIfFlagged();
// printf-style; one line per call, to the syslog through nh_log.
void debug(const char *format, ...) __attribute__((format(printf, 1, 2)));

} // namespace HelperLog

#endif
