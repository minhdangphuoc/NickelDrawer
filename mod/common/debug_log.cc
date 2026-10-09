#include "debug_log.h"

#include <errno.h>
#include <fcntl.h>
#include <stdarg.h>
#include <stdio.h>
#include <string.h>
#include <sys/stat.h>
#include <sys/time.h>
#include <time.h>
#include <unistd.h>

#include <NickelHook.h>

static const off_t MAX_LOG_BYTES = 64 * 1024;
static const size_t MAX_LINE_BYTES = 512;
static const size_t MAX_MOD_NAME_BYTES = 24;

static bool isLogEnabled = false;
static char logModName[MAX_MOD_NAME_BYTES] = "nickeldrawer";

// "2026-10-09 14:03:07.123 nddrawer: " into out; returns its length.
static int formatLinePrefix(char *out, size_t outSize)
{
    struct timeval now;
    gettimeofday(&now, NULL);
    struct tm local;
    localtime_r(&now.tv_sec, &local);
    char clock[24];
    strftime(clock, sizeof(clock), "%Y-%m-%d %H:%M:%S", &local);
    int length = snprintf(out, outSize, "%s.%03ld %s: ", clock, (long)(now.tv_usec / 1000), logModName);
    if (length < 0) {
        return 0;
    }
    return (size_t)length < outSize ? length : (int)outSize - 1;
}

// Truncates instead of appending when this line would pass the cap, so the
// newest lines (the ones before a crash) are always kept.
static int openForAppend(size_t lineBytes)
{
    struct stat info;
    bool isFull = stat(DebugLog::LOG_PATH, &info) == 0 && info.st_size + (off_t)lineBytes > MAX_LOG_BYTES;
    int flags = O_WRONLY | O_CREAT | O_APPEND | O_CLOEXEC | (isFull ? O_TRUNC : 0);
    return open(DebugLog::LOG_PATH, flags, 0644);
}

static void appendToFile(const char *line, size_t lineBytes)
{
    int file = openForAppend(lineBytes);
    if (file < 0) {
        nh_log("debug log: cannot open %s: %s (debug lines go to syslog only)", DebugLog::LOG_PATH, strerror(errno));
        return;
    }
    if (::write(file, line, lineBytes) != (ssize_t)lineBytes) {
        nh_log("debug log: short write to %s: %s", DebugLog::LOG_PATH, strerror(errno));
    }
    fsync(file); // the next step may crash Nickel; the line must be on disk first
    close(file); // never keep files open on /mnt/onboard
}

namespace DebugLog {

void enableIfFlagged(const char *modName)
{
    snprintf(logModName, sizeof(logModName), "%s", modName);
    isLogEnabled = access(FLAG_PATH, F_OK) == 0;
    if (isLogEnabled) {
        nh_log("debug log on: %s", LOG_PATH);
    }
}

bool isEnabled()
{
    return isLogEnabled;
}

void write(const char *format, ...)
{
    if (!isLogEnabled) {
        return;
    }
    char line[MAX_LINE_BYTES];
    int prefixBytes = formatLinePrefix(line, sizeof(line) - 1);
    va_list arguments;
    va_start(arguments, format);
    vsnprintf(line + prefixBytes, sizeof(line) - 1 - prefixBytes, format, arguments);
    va_end(arguments);

    nh_log("%s", line + prefixBytes);
    size_t lineBytes = strlen(line);
    line[lineBytes++] = '\n'; // room was kept by the "- 1" above
    appendToFile(line, lineBytes);
}

} // namespace DebugLog
