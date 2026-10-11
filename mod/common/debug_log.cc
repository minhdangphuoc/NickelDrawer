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

static const off_t MAX_LOG_BYTES = 256 * 1024;
static const size_t MAX_LINE_BYTES = 512;
static const size_t MAX_MOD_NAME_BYTES = 24;
static const size_t MAX_PATH_BYTES = 256;

static bool isLogEnabled = false;
static char logModName[MAX_MOD_NAME_BYTES] = "mod";
static char logPath[MAX_PATH_BYTES] = "";

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

// When this line would pass the cap, the full log becomes debug.log.1 (one
// previous file kept) and a new log starts, so a whole session survives and
// the newest lines (the ones before a crash) are always kept.
static int openForAppend(size_t lineBytes)
{
    struct stat info;
    bool isFull = stat(logPath, &info) == 0 && info.st_size + (off_t)lineBytes > MAX_LOG_BYTES;
    if (isFull) {
        char previousPath[MAX_PATH_BYTES + 4];
        snprintf(previousPath, sizeof(previousPath), "%s.1", logPath);
        rename(logPath, previousPath);
    }
    return open(logPath, O_WRONLY | O_CREAT | O_APPEND | O_CLOEXEC, 0644);
}

static void appendToFile(const char *line, size_t lineBytes)
{
    int file = openForAppend(lineBytes);
    if (file < 0) {
        nh_log("debug log: cannot open %s: %s (debug lines go to syslog only)", logPath, strerror(errno));
        return;
    }
    if (::write(file, line, lineBytes) != (ssize_t)lineBytes) {
        nh_log("debug log: short write to %s: %s", logPath, strerror(errno));
    }
    fsync(file); // the next step may crash Nickel; the line must be on disk first
    close(file); // never keep files open on /mnt/onboard
}

namespace DebugLog {

void enableIfFlagged(const char *modName, const char *flagPath, const char *logFilePath)
{
    snprintf(logModName, sizeof(logModName), "%s", modName);
    snprintf(logPath, sizeof(logPath), "%s", logFilePath);
    isLogEnabled = access(flagPath, F_OK) == 0;
    if (isLogEnabled) {
        nh_log("debug log on: %s", logPath);
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
